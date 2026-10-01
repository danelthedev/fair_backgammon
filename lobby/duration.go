package lobby

import (
	"io"
	"net/http"
	"sync"
	"time"
)

// gifDuration returns the full animation runtime in ms for a GIF url,
// parsed from its Graphic Control Extension delays. 0 on any failure.
// ponytail: cards outlive the fallback 5s only when the gif is genuinely longer.
var durCache = struct {
	sync.Mutex
	m map[string]int
}{m: make(map[string]int)}

func gifDuration(url string) int {
	durCache.Lock()
	if d, ok := durCache.m[url]; ok {
		durCache.Unlock()
		return d
	}
	durCache.Unlock()
	d := fetchGifDuration(url)
	durCache.Lock()
	durCache.m[url] = d
	// ponytail: tiny cache, evict a slice when it outgrows
	if len(durCache.m) > 500 {
		n := 0
		for k := range durCache.m {
			delete(durCache.m, k)
			n++
			if n >= 100 || len(durCache.m) <= 400 {
				break
			}
		}
	}
	durCache.Unlock()
	return d
}

func fetchGifDuration(url string) int {
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	return gifTotalDelayMs(io.LimitReader(resp.Body, 30<<20))
}

// gifTotalDelayMs scans the GIF block stream summing frame delays (1/100s each).
func gifTotalDelayMs(r io.Reader) int {
	magic := make([]byte, 6)
	if _, err := io.ReadFull(r, magic); err != nil {
		return 0
	}
	if string(magic) != "GIF87a" && string(magic) != "GIF89a" {
		return 0
	}
	l := make([]byte, 7)
	if _, err := io.ReadFull(r, l); err != nil {
		return 0
	}
	skipColorTable(r, l[4])
	delays := 0
	buf := make([]byte, 9)
	one := make([]byte, 1)
	for {
		if _, err := io.ReadFull(r, one); err != nil {
			return delays
		}
		switch one[0] {
		case 0x3B: // trailer
			return delays
		case 0x21: // extension
			if _, err := io.ReadFull(r, one); err != nil {
				return delays
			}
			if one[0] == 0xF9 { // graphic control: delay lives in the 4 data bytes
				if _, err := io.ReadFull(r, buf[:6]); err != nil { // size + 4 data + terminator
					return delays
				}
				delays += (int(buf[2]) | int(buf[3])<<8) * 10 // ms per frame
			} else if err := skipSubBlocks(r); err != nil {
				return delays
			}
		case 0x2C: // image descriptor
			if _, err := io.ReadFull(r, buf[:9]); err != nil {
				return delays
			}
			skipColorTable(r, buf[8])
			if _, err := io.ReadFull(r, one); err != nil { // LZW min code size
				return delays
			}
			if err := skipSubBlocks(r); err != nil {
				return delays
			}
		default: // unknown block, skip its sub-blocks
			if err := skipSubBlocks(r); err != nil {
				return delays
			}
		}
	}
}

// skipColorTable discards the global/local color table whose size is in the packed flags byte.
func skipColorTable(r io.Reader, packed byte) {
	if packed&0x80 == 0 {
		return
	}
	n := 3 << (((packed >> 4) & 7) + 1)
	_, _ = io.CopyN(io.Discard, r, int64(n))
}

func skipSubBlocks(r io.Reader) error {
	sz := make([]byte, 1)
	for {
		if _, err := io.ReadFull(r, sz); err != nil {
			return err
		}
		if sz[0] == 0 {
			return nil
		}
		if _, err := io.CopyN(io.Discard, r, int64(sz[0])); err != nil {
			return err
		}
	}
}

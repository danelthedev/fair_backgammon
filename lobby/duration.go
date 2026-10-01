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

// mp4Duration reads an mp4/webm(mvhd-compatible) sibling's runtime via a tiny Range fetch.
// Video-rip gifs compress long clips into few frames, so the video length is authoritative.
func mp4Duration(url string) int {
	durCache.Lock()
	if d, ok := durCache.m["v:"+url]; ok {
		durCache.Unlock()
		return d
	}
	durCache.Unlock()
	client := &http.Client{Timeout: 5 * time.Second}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Range", "bytes=0-16384") // moov is faststart on Klipy — first 16KB is enough
	resp, err := client.Do(req)
	if err != nil {
		return 0
	}
	defer resp.Body.Close()
	head, _ := io.ReadAll(io.LimitReader(resp.Body, 32<<10))
	d := mvhdDurationMs(head)
	durCache.Lock()
	durCache.m["v:"+url] = d
	durCache.Unlock()
	return d
}

// mvhdDurationMs scans box headers for moov>mvhd and returns the movie's duration in ms.
func mvhdDurationMs(b []byte) int {
	i := 0
	for i+8 <= len(b) {
		size := int(uint32(b[i])<<24 | uint32(b[i+1])<<16 | uint32(b[i+2])<<8 | uint32(b[i+3]))
		if size == 1 {
			if i+16 > len(b) {
				break
			}
			size = int(uint64(b[i+8])<<56 | uint64(b[i+9])<<48 | uint64(b[i+10])<<40 | uint64(b[i+11])<<32 | uint64(b[i+12])<<24 | uint64(b[i+13])<<16 | uint64(b[i+14])<<8 | uint64(b[i+15]))
		}
		if size < 8 {
			break
		}
		typ := string(b[i+4 : i+8])
		if typ == "moov" {
			return mvhdDurationMs(b[i+8:])
		}
		if typ == "mvhd" {
			body := b[i+8 : min(len(b), i+size)]
			if len(body) < 20 {
				return 0
			}
			if body[0] == 1 { // version 1
				if len(body) < 32 {
					return 0
				}
				ts := int(uint32(body[20])<<24 | uint32(body[21])<<16 | uint32(body[22])<<8 | uint32(body[23]))
				dur := int(uint64(body[24])<<56 | uint64(body[25])<<48 | uint64(body[26])<<40 | uint64(body[27])<<32 | uint64(body[28])<<24 | uint64(body[29])<<16 | uint64(body[30])<<8 | uint64(body[31]))
				if ts > 0 {
					return dur * 1000 / ts
				}
			} else {
				ts := int(uint32(body[12])<<24 | uint32(body[13])<<16 | uint32(body[14])<<8 | uint32(body[15]))
				dur := int(uint32(body[16])<<24 | uint32(body[17])<<16 | uint32(body[18])<<8 | uint32(body[19]))
				if ts > 0 {
					return dur * 1000 / ts
				}
			}
			return 0
		}
		if size <= 0 {
			break
		}
		i += size
	}
	return 0
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
				d := int(buf[2]) | int(buf[3])<<8
				if d <= 10 { // ponytail: browsers floor sub-100ms gif delays to 100ms; video-rip gifs rely on it
					d = 10
				}
				delays += d * 10 // ms per frame
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

// gifDelayStats: ms total, count of zero-delay frames, frame count.
func gifDelayStats(r io.Reader) (int, int, int) { return 0, 0, 0 }

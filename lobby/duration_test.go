package lobby

import (
	"bytes"
	"testing"
)

// makeGif builds a minimal multi-frame gif; delays are in 1/100s units.
func makeGif(delays ...int) []byte {
	var b bytes.Buffer
	b.WriteString("GIF89a")
	b.Write([]byte{1, 0, 1, 0, 0x80, 0, 0}) // 1x1, GCT flag + size 0 (2 colors)
	b.Write(make([]byte, 6))                // global color table
	for _, d := range delays {
		b.Write([]byte{0x21, 0xF9, 0x04, 0, byte(d), byte(d >> 8), 0, 0x00})
		b.Write([]byte{0x2C, 0, 0, 0, 0, 1, 0, 1, 0, 0x00}) // image descriptor
		b.Write([]byte{0x02})                               // LZW min code size
		b.Write([]byte{0x01, 0x00, 0x00})                   // data sub-block + terminator
	}
	b.WriteByte(0x3B) // trailer
	return b.Bytes()
}

func TestGifTotalDelayMs(t *testing.T) {
	if n := gifTotalDelayMs(bytes.NewReader(makeGif(10, 200))); n != 2100 {
		t.Fatalf("want 2100ms got %d", n)
	}
	if n := gifTotalDelayMs(bytes.NewReader(makeGif(100))); n != 1000 {
		t.Fatalf("want 1000ms got %d", n)
	}
	if n := gifTotalDelayMs(bytes.NewReader(makeGif(0))); n != 100 {
		t.Fatalf("zero-delay frame floors to 100ms, got %d", n)
		mvhd := []byte{0, 0, 0, 32, 'm', 'o', 'o', 'v', 0, 0, 0, 108, 'm', 'v', 'h', 'd', 0} // moov+mvhd header, version 0
		mvhd = append(mvhd, make([]byte, 11)...)
		mvhd = append(mvhd, 0, 0, 3, 232) // timescale 1000
		mvhd = append(mvhd, 0, 0, 0, 30)  // duration 30s
		if n := mvhdDurationMs(mvhd); n != 30000 {
			t.Fatalf("want 30000ms got %d", n)
		}
	}
	if n := gifTotalDelayMs(bytes.NewReader([]byte("not a gif"))); n != 0 {
		t.Fatalf("want 0 got %d", n)
	}
	// 100 frames at 2s each = 200s; must sum them all
	ds := make([]int, 100)
	for i := range ds {
		ds[i] = 200
	}
	if n := gifTotalDelayMs(bytes.NewReader(makeGif(ds...))); n != 200000 {
		t.Fatalf("want 200000ms got %d", n)
	}
}

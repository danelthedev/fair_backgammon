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
	b.Write(make([]byte, 6))               // global color table
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
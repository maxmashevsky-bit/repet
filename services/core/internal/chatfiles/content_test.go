package chatfiles

import (
	"bytes"
	"encoding/binary"
	"hash/crc32"
	"image"
	"image/png"
	"strings"
	"testing"
)

func TestNormalize(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"x.html", []byte("<script>alert(1)</script>")}, {"x.svg", []byte("<svg/>")}, {"x.pdf", []byte("%PDF-1.4")}, {"x.zip", []byte("PK")},
		{"x.png", []byte("not an image")}, {"../x.txt", []byte("text")}, {"x\r\n.txt", []byte("text")}, {"x\u202e.txt", []byte("text")},
		{"x.txt", []byte{0xff}}, {"x.txt", []byte{0}}, {"x.txt", []byte("\u202e")}, {"x.txt", nil}, {"x.txt", []byte(strings.Repeat("a", MaxBytes+1))},
	} {
		if _, err := Normalize(tc.name, tc.data); err == nil {
			t.Errorf("accepted %q", tc.name)
		}
	}
	v, err := Normalize("Условие.txt", []byte("Текст\r\n1/2 < 3"))
	if err != nil || string(v.Bytes) != "Текст\n1/2 < 3" {
		t.Fatal(v, err)
	}
	var source bytes.Buffer
	if err := png.Encode(&source, image.NewNRGBA(image.Rect(0, 0, 2, 3))); err != nil {
		t.Fatal(err)
	}
	raw := append(source.Bytes(), []byte("<script>bad</script>")...)
	v, err = Normalize("image.png", raw)
	if err != nil || bytes.Contains(v.Bytes, []byte("script")) {
		t.Fatal("payload not stripped", err)
	}
	if _, err = Normalize("image.jpg", raw); err == nil {
		t.Fatal("extension spoof accepted")
	}
	// Valid CRC with an excessive pixel count must be rejected before allocating a bitmap.
	large := append([]byte(nil), source.Bytes()...)
	binary.BigEndian.PutUint32(large[16:20], 4096)
	binary.BigEndian.PutUint32(large[20:24], 4096)
	binary.BigEndian.PutUint32(large[29:33], crc32.ChecksumIEEE(large[12:29]))
	if _, err = Normalize("large.png", large); err == nil {
		t.Fatal("image bomb accepted")
	}
}

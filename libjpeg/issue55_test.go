package libjpeg_test

import (
	"bytes"
	"testing"

	"github.com/AlanRace/go-bio/libjpeg"
)

// https://github.com/pixiv/go-libjpeg/issues/55
func TestDecodeAndEncodeRGBJPEG(t *testing.T) {
	data := []byte("\xff\xd8\xff\xdb\x00C\x000000000000000" +
		"00000000000000000000" +
		"00000000000000000000" +
		"00000000000\xff\xc0\x00\x11\b\x00000" +
		"\x03R\"\x00G\x11\x00B\x11\x00\xff\xda\x00\f\x03R\x00G\x00B" +
		"\x00")

	img, err := libjpeg.Decode(bytes.NewReader(data), &libjpeg.DecoderOptions{})
	if err != nil {
		t.Log(err)
		return
	}

	var w bytes.Buffer
	err = libjpeg.Encode(&w, img, &libjpeg.EncoderOptions{})
	if err != nil {
		t.Errorf("encoding after decoding failed: %v", err)
	}
}

package spec_test

import (
	"bytes"
	"testing"

	"github.com/eak1mov/go-libtiles/pm/spec"
	"github.com/google/go-cmp/cmp"
)

func TestCompression(t *testing.T) {
	dataCases := []struct {
		Name string
		Data []byte
	}{
		{Name: "Repeat", Data: bytes.Repeat([]byte{42}, 100500)},
		{Name: "Foobar", Data: []byte("foobar")},
	}
	compressionCases := []struct {
		Name        string
		Compression spec.Compression
	}{
		{Name: "None", Compression: spec.CompressionNone},
		{Name: "Gzip", Compression: spec.CompressionGzip},
	}
	for _, dc := range dataCases {
		for _, cc := range compressionCases {
			t.Run(dc.Name+cc.Name, func(t *testing.T) {
				compressed, err := spec.Compress(dc.Data, cc.Compression)
				if err != nil {
					t.Fatalf("Compress failed: %v", err)
				}
				decompressed, err := spec.Decompress(compressed, cc.Compression)
				if err != nil {
					t.Fatalf("Decompress failed: %v", err)
				}
				if !cmp.Equal(dc.Data, decompressed) {
					t.Errorf("Decompress(Compress(input)) != input")
				}
			})
		}
	}
}

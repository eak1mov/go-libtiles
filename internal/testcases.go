package internal

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"iter"
	"os"
	"testing"
)

func TestdataCases(t *testing.T, filePath string) iter.Seq2[string, []byte] {
	return func(yield func(string, []byte) bool) {
		t.Helper()

		file, err := os.Open(filePath)
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()

		gzReader, err := gzip.NewReader(file)
		if err != nil {
			t.Fatal(err)
		}
		defer gzReader.Close()

		tarReader := tar.NewReader(gzReader)

		for {
			hdr, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			if err != nil {
				t.Fatal(err)
			}
			if got, want := hdr.Typeflag, tar.TypeReg; got != byte(want) {
				t.Fatalf("hdr.Typeflag = %v, want = %v", got, want)
			}

			fileData, err := io.ReadAll(tarReader)
			if err != nil {
				t.Fatal(err)
			}

			if !yield(hdr.Name, fileData) {
				return
			}
		}
	}
}

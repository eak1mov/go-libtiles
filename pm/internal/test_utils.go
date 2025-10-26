package internal

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"iter"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestdataCases(t *testing.T, filePath string) iter.Seq2[string, []byte] {
	return func(yield func(string, []byte) bool) {
		file, err := os.Open(filePath)
		require.NoError(t, err)
		defer file.Close()

		gzReader, err := gzip.NewReader(file)
		require.NoError(t, err)
		defer gzReader.Close()

		tarReader := tar.NewReader(gzReader)

		for {
			hdr, err := tarReader.Next()
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
			require.True(t, hdr.Typeflag == tar.TypeReg)

			fileData, err := io.ReadAll(tarReader)
			require.NoError(t, err)

			if !yield(hdr.Name, fileData) {
				return
			}
		}
	}
}

package internal

import (
	"archive/zip"
	"io"
	"path"
)

const testdataPrefix = "libtiles-testdata-index-v0.2.0"

func ReadTestdata(archivePath, fileName string) ([]byte, error) {
	r, err := zip.OpenReader(archivePath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	f, err := r.Open(path.Join(testdataPrefix, fileName))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return io.ReadAll(f)
}

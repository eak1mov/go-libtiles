package spec_test

import (
	"bytes"
	"os"
	"testing"

	"github.com/eak1mov/go-libtiles/pm/spec"
	"github.com/stretchr/testify/require"
)

func TestCompression(t *testing.T) {
	data1 := bytes.Repeat([]byte{42}, 100500)
	data2 := []byte("foobar")

	data1c, err := spec.Compress(data1, spec.CompressionGzip)
	require.NoError(t, err)
	data1d, err := spec.Decompress(data1c, spec.CompressionGzip)
	require.NoError(t, err)
	require.Equal(t, data1, data1d)

	data1c, err = spec.Compress(data1, spec.CompressionNone)
	require.NoError(t, err)
	data1d, err = spec.Decompress(data1c, spec.CompressionNone)
	require.NoError(t, err)
	require.Equal(t, data1, data1d)

	data2c, err := spec.Compress(data2, spec.CompressionGzip)
	require.NoError(t, err)
	data2d, err := spec.Decompress(data2c, spec.CompressionGzip)
	require.NoError(t, err)
	require.Equal(t, data2, data2d)

	data2c, err = spec.Compress(data2, spec.CompressionNone)
	require.NoError(t, err)
	data2d, err = spec.Decompress(data2c, spec.CompressionNone)
	require.NoError(t, err)
	require.Equal(t, data2, data2d)
}

func BenchmarkCompression(b *testing.B) {
	fileData, err := os.ReadFile("../testdata/input.tar.gz")
	require.NoError(b, err)

	data1, err := spec.Decompress(fileData, spec.CompressionGzip)
	require.NoError(b, err)

	data1c, err := spec.Compress(data1, spec.CompressionGzip)
	require.NoError(b, err)
	data1d, err := spec.Decompress(data1c, spec.CompressionGzip)
	require.NoError(b, err)
	require.Equal(b, data1, data1d)

	data2c, err := spec.Compress(data1, spec.CompressionNone)
	require.NoError(b, err)
	data2d, err := spec.Decompress(data2c, spec.CompressionNone)
	require.NoError(b, err)
	require.Equal(b, data1, data2d)
}

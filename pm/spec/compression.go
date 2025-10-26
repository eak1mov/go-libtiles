package spec

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

func Compress(data []byte, compression Compression) ([]byte, error) {
	if compression == CompressionNone {
		return data, nil
	}

	if compression != CompressionGzip {
		return nil, fmt.Errorf("compression not supported (%v)", compression)
	}

	var buffer bytes.Buffer
	writer, _ := gzip.NewWriterLevel(&buffer, gzip.BestCompression)

	_, err := writer.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to compress: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to compress: %w", err)
	}

	return buffer.Bytes(), nil
}

func Decompress(data []byte, compression Compression) ([]byte, error) {
	if compression == CompressionNone {
		return data, nil
	}

	if compression != CompressionGzip {
		return nil, fmt.Errorf("compression not supported (%v)", compression)
	}

	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}
	defer reader.Close()

	result, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decompress: %w", err)
	}

	return result, nil
}

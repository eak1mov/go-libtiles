package spec_test

import (
	"encoding/binary"
	"errors"
	"io"
	"testing"

	"github.com/eak1mov/go-libtiles/pm/spec"
	"github.com/stretchr/testify/require"
)

func TestHeaderLength(t *testing.T) {
	require.Equal(t, binary.Size(spec.Header{}), spec.HeaderLength)
}

func TestHeaderSerializer(t *testing.T) {
	header1 := spec.Header{HeaderMagic: spec.HeaderMagicV3}
	headerData := spec.SerializeHeader(&header1)
	header2, err := spec.DeserializeHeader(headerData)
	require.Nil(t, err)
	require.Equal(t, header1, *header2)
}

func TestHeaderErrors(t *testing.T) {
	buf := []byte("foobar")
	_, err := spec.DeserializeHeader(buf)
	require.Truef(t, errors.Is(err, spec.ErrInvalidHeader), "%v", err)
	require.Truef(t, errors.Is(err, io.ErrUnexpectedEOF), "%v", err)
}

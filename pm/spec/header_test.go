package spec_test

import (
	"io"
	"testing"

	"github.com/eak1mov/go-libtiles/pm/spec"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestHeaderLength(t *testing.T) {
	if got, want := len(spec.SerializeHeader(&spec.Header{})), spec.HeaderLength; got != want {
		t.Errorf("SerializeHeader length = %v, want = %v", got, want)
	}
}

func TestHeaderSerializer(t *testing.T) {
	input := spec.Header{HeaderMagic: spec.HeaderMagicV3}
	got, err := spec.DeserializeHeader(spec.SerializeHeader(&input))
	if err != nil {
		t.Fatalf("DeserializeHeader failed: %v", err)
	}
	if diff := cmp.Diff(input, *got); diff != "" {
		t.Errorf("DeserializeHeader(SerializeHeader(%v)) mismatch (-want +got):\n%s", input, diff)
	}
}

func TestHeaderErrors(t *testing.T) {
	_, err := spec.DeserializeHeader([]byte("foobar"))
	if got, want := err, spec.ErrInvalidHeader; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Errorf("DeserializeHeader(invalid data) = %q, want error presence = %q", got, want)
	}
	if got, want := err, io.ErrUnexpectedEOF; !cmp.Equal(got, want, cmpopts.EquateErrors()) {
		t.Errorf("DeserializeHeader(invalid data) = %q, want error presence = %q", got, want)
	}
}

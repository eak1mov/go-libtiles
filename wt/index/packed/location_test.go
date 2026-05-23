package packed_test

import (
	"testing"

	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt/fbs"
	"github.com/eak1mov/go-libtiles/wt/index/packed"
	"github.com/google/go-cmp/cmp"
)

func TestLocation(t *testing.T) {
	got := packed.Unpack(packed.Read([]byte{1, 0, 0, 0, 0, 1, 0, 0}))
	want := tile.Location{Offset: 1, Length: 1}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Read() mismatch (-want+got):\n%v", diff)
	}
}

func TestFbs(t *testing.T) {
	location := packed.Pack(tile.Location{Offset: 1, Length: 1})

	fbsData := make([]byte, packed.LocationLength)
	fbsLocation := fbs.Location{}
	fbsLocation.Init(fbsData, 0)

	packed.Write(fbsData, location)
	if got, want := packed.ReadFbs(fbsLocation), location; got != want {
		t.Errorf("ReadFbs(Write(%x)) = %x, want = %x", location, got, want)
	}

	v0, v1 := packed.WriteFbs(location)
	fbsLocation.MutateV0(v0)
	fbsLocation.MutateV1(v1)
	if got, want := packed.Read(fbsData), location; got != want {
		t.Errorf("Read(WriteFbs(%x)) = %x, want = %x", location, got, want)
	}
	if got, want := packed.ReadFbs(fbsLocation), location; got != want {
		t.Errorf("ReadFbs(WriteFbs(%x)) = %x, want = %x", location, got, want)
	}
}

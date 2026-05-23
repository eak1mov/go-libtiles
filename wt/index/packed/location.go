package packed

import (
	"encoding/binary"

	"github.com/eak1mov/go-libtiles/tile"
	"github.com/eak1mov/go-libtiles/wt/fbs"
)

const (
	MaxOffset = (1<<40 - 1) // 1 TiB
	MaxLength = (1<<24 - 1) // 16 MiB
)

type Location uint64

const LocationLength = 8

func (l Location) Offset() uint64 {
	return uint64(l) & (1<<40 - 1)
}

func (l Location) Length() uint64 {
	return uint64(l) >> 40
}

func Pack(location tile.Location) Location {
	return Location(location.Length<<40 + location.Offset)
}

func Unpack(location Location) tile.Location {
	return tile.Location{
		Offset: location.Offset(),
		Length: location.Length(),
	}
}

func Read(data []byte) Location {
	return Location(binary.LittleEndian.Uint64(data))
}

func Write(data []byte, location Location) {
	binary.LittleEndian.PutUint64(data, uint64(location))
}

func ReadFbs(location fbs.Location) Location {
	table := location.Table()
	data := table.Bytes[table.Pos : table.Pos+LocationLength]
	return Read(data)
}

func WriteFbs(location Location) (v0, v1 uint32) {
	v0 = uint32(uint64(location))
	v1 = uint32(uint64(location) >> 32)
	return
}

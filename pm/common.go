package pm

import "github.com/eak1mov/go-libtiles/pm/spec"

type TileId = spec.TileId

type HeaderMetadata struct {
	TileCompression spec.Compression
	TileType        spec.TileType
	MinZoom         uint8
	MaxZoom         uint8
	MinLonE7        int32
	MinLatE7        int32
	MaxLonE7        int32
	MaxLatE7        int32
	CenterZoom      uint8
	CenterLonE7     int32
	CenterLatE7     int32
}

func (m *HeaderMetadata) CopyFromHeader(header *spec.Header) {
	m.TileCompression = header.TileCompression
	m.TileType = header.TileType
	m.MinZoom = header.MinZoom
	m.MaxZoom = header.MaxZoom
	m.MinLonE7 = header.MinLonE7
	m.MinLatE7 = header.MinLatE7
	m.MaxLonE7 = header.MaxLonE7
	m.MaxLatE7 = header.MaxLatE7
	m.CenterZoom = header.CenterZoom
	m.CenterLonE7 = header.CenterLonE7
	m.CenterLatE7 = header.CenterLatE7
}

func (m *HeaderMetadata) CopyToHeader(header *spec.Header) {
	header.TileCompression = m.TileCompression
	header.TileType = m.TileType
	header.MinZoom = m.MinZoom
	header.MaxZoom = m.MaxZoom
	header.MinLonE7 = m.MinLonE7
	header.MinLatE7 = m.MinLatE7
	header.MaxLonE7 = m.MaxLonE7
	header.MaxLatE7 = m.MaxLatE7
	header.CenterZoom = m.CenterZoom
	header.CenterLonE7 = m.CenterLonE7
	header.CenterLatE7 = m.CenterLatE7
}

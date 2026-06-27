package internal

import "strings"

func DeduceFormat(format, filePath string) string {
	switch {
	case format != "":
		return format
	case strings.HasSuffix(filePath, ".mbtiles"):
		return "mbtiles"
	case strings.HasSuffix(filePath, ".pmtiles"):
		return "pmtiles"
	case strings.HasSuffix(filePath, ".wtiles"):
		return "wtiles"
	default:
		return format
	}
}

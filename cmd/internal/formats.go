package internal

import "strings"

func DeduceFormat(format, filePath string) string {
	if format == "" && strings.HasSuffix(filePath, ".mbtiles") {
		return "mbtiles"
	}
	if format == "" && strings.HasSuffix(filePath, ".pmtiles") {
		return "pmtiles"
	}
	if format == "" && strings.HasSuffix(filePath, ".wtiles") {
		return "wtiles"
	}
	return format
}

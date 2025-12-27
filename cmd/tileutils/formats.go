package main

import "strings"

func deduceFormat(format, filePath string) string {
	if format == "" && strings.HasSuffix(filePath, ".mbtiles") {
		return "mbtiles"
	}
	if format == "" && strings.HasSuffix(filePath, ".pmtiles") {
		return "pmtiles"
	}
	return format
}

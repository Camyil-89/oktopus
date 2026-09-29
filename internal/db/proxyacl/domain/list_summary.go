package domain

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// NamedListSummary — list без body (таблица в UI).
type NamedListSummary struct {
	ID                  uuid.UUID
	Name                string
	ListType            string
	SourceMode          string
	SourceURL           string
	PollIntervalMinutes int
	BodyLineCount       int
	BodyPreview         string
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// NamedListBodyStats считает строки и превью первой непустой строки.
func NamedListBodyStats(body string) (lineCount int, preview string) {
	if body == "" {
		return 0, ""
	}
	lineCount = 1
	var lineStart int
	previewSet := false
	for i := 0; i < len(body); i++ {
		if body[i] != '\n' {
			continue
		}
		if !previewSet {
			preview = previewLine(body[lineStart:i])
			previewSet = preview != ""
		}
		lineCount++
		lineStart = i + 1
	}
	if !previewSet {
		preview = previewLine(body[lineStart:])
	}
	return lineCount, preview
}

func previewLine(line string) string {
	line = strings.TrimSpace(strings.ReplaceAll(line, "\r", ""))
	if line == "" {
		return ""
	}
	const max = 48
	if len(line) <= max {
		return line
	}
	return line[:max] + "…"
}

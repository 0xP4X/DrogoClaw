package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode"

	"github.com/jung-kurt/gofpdf"
)

// SaveDocument writes Markdown or PDF content to a local directory.
func SaveDocument(title, content, format, directory string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		title = "drogonclaw_document"
	}
	content = strings.TrimSpace(content)
	if content == "" {
		return "", fmt.Errorf("document content is empty")
	}

	format = strings.ToLower(strings.TrimPrefix(strings.TrimSpace(format), "."))
	if format == "" {
		format = "md"
	}
	if format != "md" && format != "pdf" {
		return "", fmt.Errorf("unsupported document format %q; use md or pdf", format)
	}

	dir, err := resolveOutputDir(directory)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("create output directory: %w", err)
	}

	filename := fmt.Sprintf("%s_%s.%s", slugifyFilename(title), time.Now().Format("20060102_150405"), format)
	path := filepath.Join(dir, filename)

	switch format {
	case "md":
		if err := os.WriteFile(path, []byte(content+"\n"), 0644); err != nil {
			return "", fmt.Errorf("write markdown: %w", err)
		}
	case "pdf":
		if err := writeTextPDF(path, title, content); err != nil {
			return "", err
		}
	}

	return path, nil
}

func resolveOutputDir(directory string) (string, error) {
	directory = strings.TrimSpace(directory)
	if directory == "" || strings.EqualFold(directory, "desktop") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, "Desktop"), nil
	}
	if strings.HasPrefix(directory, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
		return filepath.Join(home, strings.TrimPrefix(directory, "~")), nil
	}
	return filepath.Clean(directory), nil
}

func slugifyFilename(title string) string {
	var b strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(title) {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case !lastDash:
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "drogonclaw-document"
	}
	if len(out) > 80 {
		out = strings.TrimRight(out[:80], "-")
	}
	return out
}

func writeTextPDF(path, title, content string) error {
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(18, 18, 18)
	pdf.AddPage()
	pdf.SetFont("Arial", "B", 18)
	pdf.MultiCell(174, 8, title, "", "L", false)
	pdf.Ln(4)
	pdf.SetFont("Arial", "", 11)

	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			pdf.Ln(4)
			continue
		}
		pdf.MultiCell(174, 6, line, "", "L", false)
	}

	if err := pdf.OutputFileAndClose(path); err != nil {
		return fmt.Errorf("write pdf: %w", err)
	}
	return nil
}

// Command pikchrgen generates SVG assets for Pikchr fenced code blocks.
package main

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gopikchr/gopikchr"
)

var idPattern = regexp.MustCompile(`id[[:space:]]*=[[:space:]]*([^[:space:]}]+)`)
var validID = regexp.MustCompile(`^[A-Za-z0-9_-]+$`)

type diagram struct {
	id     string
	source string
	file   string
	line   int
}

func main() {
	contentDir := "content"
	assetDir := filepath.Join("assets", "pikchr")
	if err := generate(contentDir, assetDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(contentDir, assetDir string) error {
	var diagrams []diagram
	seen := make(map[string]string)

	err := filepath.WalkDir(contentDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || strings.ToLower(filepath.Ext(path)) != ".md" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		found, err := parseFile(path, string(data))
		if err != nil {
			return err
		}
		for _, d := range found {
			if previous, ok := seen[d.id]; ok {
				return fmt.Errorf("duplicate Pikchr id %q in %s:%d (already used in %s)", d.id, d.file, d.line, previous)
			}
			seen[d.id] = fmt.Sprintf("%s:%d", d.file, d.line)
			diagrams = append(diagrams, d)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if err := os.MkdirAll(assetDir, 0755); err != nil {
		return err
	}
	entries, err := os.ReadDir(assetDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !entry.IsDir() && strings.EqualFold(filepath.Ext(entry.Name()), ".svg") {
			if err := os.Remove(filepath.Join(assetDir, entry.Name())); err != nil {
				return err
			}
		}
	}

	for _, d := range diagrams {
		svg, _, _, err := gopikchr.Convert(d.source, gopikchr.WithSVGClass("pikchr"))
		if err != nil {
			return fmt.Errorf("%s:%d: Pikchr conversion failed: %w\n%s", d.file, d.line, err, svg)
		}
		if err := os.WriteFile(filepath.Join(assetDir, d.id+".svg"), []byte(svg), 0644); err != nil {
			return err
		}
	}

	fmt.Printf("generated %d Pikchr SVG(s)\n", len(diagrams))
	return nil
}

func parseFile(filename, content string) ([]diagram, error) {
	lines := strings.Split(strings.ReplaceAll(content, "\r\n", "\n"), "\n")
	var result []diagram

	for i := 0; i < len(lines); i++ {
		marker, info, ok := openingFence(lines[i])
		if !ok || !isPikchr(info) {
			continue
		}

		id := blockID(info)
		if id == "" {
			return nil, fmt.Errorf("%s:%d: Pikchr code block must have an id attribute", filename, i+1)
		}
		if !validID.MatchString(id) {
			return nil, fmt.Errorf("%s:%d: invalid Pikchr id %q (use only letters, numbers, '_' or '-')", filename, i+1, id)
		}

		bodyStart := i + 1
		end := bodyStart
		for ; end < len(lines); end++ {
			if closingFence(lines[end], marker) {
				break
			}
		}
		if end == len(lines) {
			return nil, fmt.Errorf("%s:%d: unterminated Pikchr code block", filename, i+1)
		}

		source := strings.TrimSpace(strings.Join(lines[bodyStart:end], "\n"))
		if source == "" {
			return nil, fmt.Errorf("%s:%d: Pikchr code block is empty", filename, i+1)
		}
		result = append(result, diagram{id: id, source: source, file: filename, line: i + 1})
		i = end
	}
	return result, nil
}

func openingFence(line string) (byte, string, bool) {
	line = strings.TrimLeft(line, " \t")
	if len(line) < 3 || (line[0] != '`' && line[0] != '~') {
		return 0, "", false
	}
	marker := line[0]
	n := 0
	for n < len(line) && line[n] == marker {
		n++
	}
	if n < 3 {
		return 0, "", false
	}
	return marker, strings.TrimSpace(line[n:]), true
}

func closingFence(line string, marker byte) bool {
	line = strings.TrimSpace(line)
	if len(line) < 3 {
		return false
	}
	for i := 0; i < len(line); i++ {
		if line[i] != marker {
			return false
		}
	}
	return true
}

func isPikchr(info string) bool {
	fields := strings.Fields(info)
	return len(fields) > 0 && fields[0] == "pikchr"
}

func blockID(info string) string {
	matches := idPattern.FindStringSubmatch(info)
	if len(matches) < 2 {
		return ""
	}
	return strings.Trim(matches[1], `"'`)
}

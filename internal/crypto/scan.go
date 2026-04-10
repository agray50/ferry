package crypto

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// SecretMatch represents a secret pattern found in a file.
type SecretMatch struct {
	FilePath string
	Line     int
	Pattern  string
	Preview  string
}

// SecretPatterns is the list of regexes used to detect secrets.
var SecretPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)export\s+\w+_TOKEN\s*=\s*\S+`),
	regexp.MustCompile(`(?i)export\s+\w+_SECRET\s*=\s*\S+`),
	regexp.MustCompile(`(?i)export\s+\w+_KEY\s*=\s*\S+`),
	regexp.MustCompile(`(?i)export\s+\w+_PASSWORD\s*=\s*\S+`),
	regexp.MustCompile(`(?i)\w+_API_KEY\s*=\s*\S+`),
	regexp.MustCompile(`(?i)password\s*=\s*\S+`),
	regexp.MustCompile(`(?im)^token\s*=\s*\S+`),
}

// ScanFile scans a single file for secret patterns.
func ScanFile(path string) ([]SecretMatch, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("ScanFile %s: %w", path, err)
	}
	defer f.Close()

	var matches []SecretMatch
	scanner := bufio.NewScanner(f)
	// Increase buffer to handle long lines (e.g. minified files, base64 blobs).
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmed := strings.TrimLeft(line, " \t")
		// skip comments
		if strings.HasPrefix(trimmed, "#") {
			continue
		}
		for _, re := range SecretPatterns {
			if re.MatchString(line) {
				matches = append(matches, SecretMatch{
					FilePath: path,
					Line:     lineNum,
					Pattern:  re.String(),
					Preview:  RedactPreview(line),
				})
				break
			}
		}
	}
	return matches, scanner.Err()
}

// ScanFiles scans multiple files. Returns map of filepath → matches.
func ScanFiles(paths []string) (map[string][]SecretMatch, error) {
	result := make(map[string][]SecretMatch)
	for _, path := range paths {
		matches, err := ScanFile(path)
		if err != nil {
			return nil, err
		}
		if len(matches) > 0 {
			result[path] = matches
		}
	}
	return result, nil
}

// RedactPreview returns the line with the value after = replaced by ***.
func RedactPreview(line string) string {
	idx := strings.Index(line, "=")
	if idx < 0 {
		return line
	}
	return line[:idx+1] + "***"
}

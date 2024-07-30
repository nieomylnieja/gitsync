package config

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"
)

func TestReadmeExample(t *testing.T) {
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err, "failed to read README.md")
	}
	scan := bufio.NewScanner(bytes.NewReader(data))
	readJSON := false
	jsonBuilder := bytes.Buffer{}
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		switch {
		case line == "```json5":
			readJSON = true
		case line == "```":
			readJSON = false
		case readJSON && !strings.HasPrefix(line, "//"):
			jsonBuilder.WriteString(line)
			jsonBuilder.WriteString("\n")
		}
	}
	if err = scan.Err(); err != nil {
		t.Fatal(err, "failed to scan README.md contents")
	}
	var config Config
	if err = json.Unmarshal(jsonBuilder.Bytes(), &config); err != nil {
		t.Fatal(err, "failed to unmarshal JSON config")
	}
	if err = config.validate(); err != nil {
		t.Fatal(err, "config validation failed")
	}
}

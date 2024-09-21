package diff

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"
)

// UnifiedFormat represents a [unified GNU diff format].
//
// [unified GNU diff format]: https://www.gnu.org/software/diffutils/manual/html_node/Detailed-Unified.html.
type UnifiedFormat struct {
	// Header contains the file context, i.e. '---' and '+++' lines.
	Header string
	// Hunks is a slice of diff hunks as represented by [Hunk].
	Hunks []Hunk
}

func (u UnifiedFormat) String(original bool) string {
	var sb strings.Builder
	sb.WriteString(u.Header)
	sb.WriteString("\n")
	for _, hunk := range u.Hunks {
		if original {
			sb.WriteString(hunk.Original)
		} else {
			sb.WriteString(hunk.String())
		}
	}
	return sb.String()
}

// Hunk represents a single diff hunk in of [UnifiedFormat].
type Hunk struct {
	// Lines is the '@@' header containing the line numbers.
	Lines string `json:"lines,omitempty"`
	// Changes contains only the changed lines, without any context.
	Changes []string `json:"changes"`
	// Original is the original string representation of the hunk.
	// It may include color codes.
	Original string `json:"-"`
}

func (h Hunk) String() string {
	var sb strings.Builder
	sb.WriteString(h.Lines)
	sb.WriteString("\n")
	for _, line := range h.Changes {
		sb.WriteString(line)
		sb.WriteString("\n")
	}
	return sb.String()
}

// Equal compares two [Hunk]s for equality.
// It ignores color codes and will not compare [Hunk.Lines] if the receiver [Hunk] has no lines defined.
func (h Hunk) Equal(other Hunk) bool {
	if (h.Lines != "" && h.Lines != other.Lines) || len(h.Changes) != len(other.Changes) {
		return false
	}
	for i := range h.Changes {
		if h.Changes[i] != other.Changes[i] {
			return false
		}
	}
	return true
}

var colorCodeRegex = regexp.MustCompile(`\x1b\[\d+m(?P<content>.*)\x1b\[\d+m`)

func ParseDiffOutput(output io.Reader) (*UnifiedFormat, error) {
	uf := &UnifiedFormat{}
	scan := bufio.NewScanner(output)
	hunkIndex := -1

	stripColorCodes := func(line string) string {
		return colorCodeRegex.ReplaceAllString(line, "${content}")
	}

	for scan.Scan() {
		line := scan.Text()
		originalLine := line
		line = stripColorCodes(line)
		switch {
		case strings.HasPrefix(line, "---"):
			uf.Header += line + "\n"
		case strings.HasPrefix(line, "+++"):
			uf.Header += line
		case strings.HasPrefix(line, "@@"):
			uf.Hunks = append(uf.Hunks, Hunk{Lines: line, Original: originalLine + "\n"})
			hunkIndex++
		default:
			if hunkIndex == -1 {
				return nil, errors.New("invalid diff output, missing hunk header")
			}
			uf.Hunks[hunkIndex].Changes = append(uf.Hunks[hunkIndex].Changes, line)
			uf.Hunks[hunkIndex].Original += originalLine + "\n"
		}
	}
	if err := scan.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse diff output: %w", err)
	}
	return uf, nil
}

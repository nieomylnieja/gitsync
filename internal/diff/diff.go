package diff

import (
	"bufio"
	"io"
	"strings"

	"github.com/pkg/errors"
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

func (u UnifiedFormat) String() string {
	var sb strings.Builder
	sb.WriteString(u.Header)
	sb.WriteString("\n")
	for _, hunk := range u.Hunks {
		sb.WriteString(hunk.String())
	}
	return sb.String()
}

// Hunk represents a single diff hunk in of [UnifiedFormat].
type Hunk struct {
	// Lines is the '@@' header containing the line numbers.
	Lines string `json:"lines"`
	// Changes contains only the changed lines, without any context.
	Changes []string `json:"changes"`
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

func (h Hunk) Equal(other Hunk) bool {
	if h.Lines != other.Lines || len(h.Changes) != len(other.Changes) {
		return false
	}
	for i := range h.Changes {
		if h.Changes[i] != other.Changes[i] {
			return false
		}
	}
	return true
}

func ParseDiffOutput(output io.Reader) (*UnifiedFormat, error) {
	uf := &UnifiedFormat{}
	scan := bufio.NewScanner(output)
	hunkIndex := -1
	for scan.Scan() {
		line := scan.Text()
		switch {
		case strings.HasPrefix(line, "---"):
			uf.Header += line + "\n"
		case strings.HasPrefix(line, "+++"):
			uf.Header += line
		case strings.HasPrefix(line, "@@"):
			uf.Hunks = append(uf.Hunks, Hunk{Lines: line})
			hunkIndex++
		default:
			uf.Hunks[hunkIndex].Changes = append(uf.Hunks[hunkIndex].Changes, line)
		}
	}
	if err := scan.Err(); err != nil {
		return nil, errors.Wrap(err, "failed to parse diff output")
	}
	return uf, nil
}

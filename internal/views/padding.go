package views

import (
	"strings"
	"time"
	"unicode"

	"github.com/derailed/k9s/internal/resource"
	"k8s.io/apimachinery/pkg/util/duration"
)

type maxyPad []int

func computeMaxColumns(pads maxyPad, sortCol int, table resource.TableData) {
	for index, h := range table.Header {
		pads[index] = len(h)
		if index == sortCol {
			pads[index] = len(h) + 2
		}
	}

	row, ageIndex := 0, len(table.Header)-1
	for _, res := range table.Rows {
		for index, field := range res.Fields {
			w := fieldWidth(field, index, ageIndex)
			if w > pads[index] {
				pads[index] = w
			}
		}
		row++
	}
}

const colPadding = 1

func fieldWidth(f string, col, ageIndex int) int {
	// Date field comes out as timestamp.
	if col == ageIndex {
		dur, err := time.ParseDuration(f)
		if err == nil {
			f = duration.HumanDuration(dur)
		}
	}
	return len(f) + colPadding
}

func isASCII(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] > unicode.MaxASCII {
			return false
		}
	}
	return true
}

// Pad a string up to the given length or truncates if greater than length.
func pad(s string, width int) string {
	if len(s) == width {
		return s
	}

	if len(s) > width {
		return resource.Truncate(s, width)
	}

	return s + strings.Repeat(" ", width-len(s))
}

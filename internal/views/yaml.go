package views

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/derailed/k9s/internal/config"
)

var (
	keyValRX = regexp.MustCompile(`\A(\s*)([\w|\-|\.|\s]+):\s(.+)\z`)
	keyRX    = regexp.MustCompile(`\A(\s*)([\w|\-|\.|\s]+):\s*\z`)
)

const (
	yamlFullFmt  = "%s[key::b]%s[colon::-]: [val::]%s"
	yamlKeyFmt   = "%s[key::b]%s[colon::-]:"
	yamlValueFmt = "[val::]%s"
)

func subStyle(s config.Yaml) (full, key, val string) {
	full = strings.Replace(yamlFullFmt, "[key", "["+s.KeyColor, 1)
	full = strings.Replace(full, "[colon", "["+s.ColonColor, 1)
	full = strings.Replace(full, "[val", "["+s.ValueColor, 1)

	key = strings.Replace(yamlKeyFmt, "[key", "["+s.KeyColor, 1)
	key = strings.Replace(key, "[colon", "["+s.ColonColor, 1)

	val = strings.Replace(yamlValueFmt, "[val", "["+s.ValueColor, 1)

	return
}

func colorizeYAML(style config.Yaml, raw string) string {
	lines := strings.Split(raw, "\n")
	full, key, val := subStyle(style)

	buff := make([]string, 0, len(lines))
	for _, l := range lines {
		res := keyValRX.FindStringSubmatch(l)
		if len(res) == 4 {
			buff = append(buff, fmt.Sprintf(full, res[1], res[2], res[3]))
			continue
		}
		res = keyRX.FindStringSubmatch(l)
		if len(res) == 3 {
			buff = append(buff, fmt.Sprintf(key, res[1], res[2]))
			continue
		}
		buff = append(buff, fmt.Sprintf(val, l))
	}

	return strings.Join(buff, "\n")
}

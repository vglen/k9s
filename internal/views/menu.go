package views

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/derailed/k9s/internal/config"
	"github.com/derailed/k9s/internal/resource"
	"github.com/derailed/tview"
	"github.com/gdamore/tcell"
	"github.com/rs/zerolog/log"
)

func init() {
	initKeys()
}

const (
	menuIndexFmt = " [key:bg:b]<%d> [fg:bg:d]%s "
	maxRows      = 7
)

var menuRX = regexp.MustCompile(`\d`)

type (
	hint struct {
		mnemonic, description string
	}
	hints []hint

	hinter interface {
		hints() hints
	}
)

func (h hints) Len() int {
	return len(h)
}

func (h hints) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
}

func (h hints) Less(i, j int) bool {
	n, err1 := strconv.Atoi(h[i].mnemonic)
	m, err2 := strconv.Atoi(h[j].mnemonic)
	if err1 == nil && err2 == nil {
		return n < m
	}
	if err1 == nil && err2 != nil {
		return true
	}
	if err1 != nil && err2 == nil {
		return false
	}
	return strings.Compare(h[i].description, h[j].description) < 0
}

// -----------------------------------------------------------------------------
type (
	actionHandler func(*tcell.EventKey) *tcell.EventKey

	keyAction struct {
		description string
		action      actionHandler
		visible     bool
	}
	keyActions map[tcell.Key]keyAction
)

func newKeyAction(d string, a actionHandler, display bool) keyAction {
	return keyAction{description: d, action: a, visible: display}
}

func (a keyActions) toHints() hints {
	kk := make([]int, 0, len(a))
	for k, v := range a {
		if v.visible {
			kk = append(kk, int(k))
		}
	}
	sort.Ints(kk)

	hh := make(hints, 0, len(kk))
	for _, k := range kk {
		if name, ok := tcell.KeyNames[tcell.Key(k)]; ok {
			hh = append(hh, hint{
				mnemonic:    name,
				description: a[tcell.Key(k)].description})
		} else {
			log.Error().Msgf("Unable to locate KeyName for %#v", string(k))
		}
	}
	return hh
}

// -----------------------------------------------------------------------------
type menuView struct {
	*tview.Table

	styles *config.Styles
}

func newMenuView(styles *config.Styles) *menuView {
	v := menuView{Table: tview.NewTable(), styles: styles}
	v.SetBackgroundColor(styles.BgColor())

	return &v
}

func (v *menuView) populateMenu(hh hints) {
	v.Clear()
	sort.Sort(hh)
	t := v.buildMenuTable(hh)
	for row := 0; row < len(t); row++ {
		for col := 0; col < len(t[row]); col++ {
			if len(t[row][col]) == 0 {
				continue
			}
			c := tview.NewTableCell(t[row][col])
			c.SetBackgroundColor(v.styles.BgColor())
			v.SetCell(row, col, c)
		}
	}
}

func (v *menuView) buildMenuTable(hh hints) [][]string {
	table := make([][]hint, maxRows+1)

	colCount := (len(hh) / maxRows) + 1
	for row := 0; row < maxRows; row++ {
		table[row] = make([]hint, colCount+1)
	}

	var row, col int
	firstCmd := true
	maxKeys := make([]int, colCount+1)
	for _, h := range hh {
		isDigit := menuRX.MatchString(h.mnemonic)
		if !isDigit && firstCmd {
			row, col, firstCmd = 0, col+1, false
		}
		if maxKeys[col] < len(h.mnemonic) {
			maxKeys[col] = len(h.mnemonic)
		}
		table[row][col] = h
		row++
		if row >= maxRows {
			col++
			row = 0
		}
	}

	strTable := make([][]string, maxRows+1)
	for r := 0; r < len(table); r++ {
		strTable[r] = make([]string, len(table[r]))
	}
	for row := range strTable {
		for col := range strTable[row] {
			strTable[row][col] = v.formatMenu(table[row][col], maxKeys[col])
		}
	}

	return strTable
}

// ----------------------------------------------------------------------------
// Helpers...

func toMnemonic(s string) string {
	if len(s) == 0 {
		return s
	}

	return "<" + strings.ToLower(s) + ">"
}

func (v *menuView) formatMenu(h hint, size int) string {
	i, err := strconv.Atoi(h.mnemonic)
	if err == nil {
		return formatNSMenu(i, h.description, v.styles.Frame())
	}

	return formatPlainMenu(h, size, v.styles.Frame())
}

func formatNSMenu(i int, name string, styles config.Frame) string {
	fmat := strings.Replace(menuIndexFmt, "[key", "["+styles.Menu.NumKeyColor, 1)
	fmat = strings.Replace(fmat, ":bg:", ":"+styles.Title.BgColor+":", -1)
	fmat = strings.Replace(fmat, "[fg", "["+styles.Menu.FgColor, 1)
	return fmt.Sprintf(fmat, i, resource.Truncate(name, 14))
}

func formatPlainMenu(h hint, size int, styles config.Frame) string {
	menuFmt := " [key:bg:b]%-" + strconv.Itoa(size+2) + "s [fg:bg:d]%s "
	fmat := strings.Replace(menuFmt, "[key", "["+styles.Menu.KeyColor, 1)
	fmat = strings.Replace(fmat, "[fg", "["+styles.Menu.FgColor, 1)
	fmat = strings.Replace(fmat, ":bg:", ":"+styles.Title.BgColor+":", -1)
	return fmt.Sprintf(fmat, toMnemonic(h.mnemonic), h.description)
}

// -----------------------------------------------------------------------------
// Key mapping Constants

// Defines numeric keys for container actions
const (
	Key0 int32 = iota + 48
	Key1
	Key2
	Key3
	Key4
	Key5
	Key6
	Key7
	Key8
	Key9
)

// Defines AltKeys
const (
	KeyAltA tcell.Key = 4 * (iota + 97)
	KeyAltB
	KeyAltC
	KeyAltD
	KeyAltE
	KeyAltF
	KeyAltG
	KeyAltH
	KeyAltI
	KeyAltJ
	KeyAltK
	KeyAltL
	KeyAltM
	KeyAltN
	KeyAltO
	KeyAltP
	KeyAltQ
	KeyAltR
	KeyAltS
	KeyAltT
	KeyAltU
	KeyAltV
	KeyAltW
	KeyAltX
	KeyAltY
	KeyAltZ
)

// Defines char keystrokes
const (
	KeyA tcell.Key = iota + 97
	KeyB
	KeyC
	KeyD
	KeyE
	KeyF
	KeyG
	KeyH
	KeyI
	KeyJ
	KeyK
	KeyL
	KeyM
	KeyN
	KeyO
	KeyP
	KeyQ
	KeyR
	KeyS
	KeyT
	KeyU
	KeyV
	KeyW
	KeyX
	KeyY
	KeyZ
	KeyHelp  = 63
	KeySlash = 47
	KeyColon = 58
)

// Define Shift Keys
const (
	KeyShiftA tcell.Key = iota + 65
	KeyShiftB
	KeyShiftC
	KeyShiftD
	KeyShiftE
	KeyShiftF
	KeyShiftG
	KeyShiftH
	KeyShiftI
	KeyShiftJ
	KeyShiftK
	KeyShiftL
	KeyShiftM
	KeyShiftN
	KeyShiftO
	KeyShiftP
	KeyShiftQ
	KeyShiftR
	KeyShiftS
	KeyShiftT
	KeyShiftU
	KeyShiftV
	KeyShiftW
	KeyShiftX
	KeyShiftY
	KeyShiftZ
)

var numKeys = map[int]int32{
	0: Key0,
	1: Key1,
	2: Key2,
	3: Key3,
	4: Key4,
	5: Key5,
	6: Key6,
	7: Key7,
	8: Key8,
	9: Key9,
}

func initKeys() {
	tcell.KeyNames[tcell.Key(KeyHelp)] = "?"
	tcell.KeyNames[tcell.Key(KeySlash)] = "/"

	initNumbKeys()
	initStdKeys()
	initShiftKeys()
	initAltKeys()
}

func initNumbKeys() {
	for i := 0; i < 10; i++ {
		r := int(Key0) + i
		tcell.KeyNames[tcell.Key(48+i)] = string(r)
	}
}

func initStdKeys() {
	for i := 0; i < 27; i++ {
		r := int(KeyA) + i
		tcell.KeyNames[tcell.Key(r)] = string(r)
	}
}

func initShiftKeys() {
	for i := 0; i < 27; i++ {
		r := int(KeyShiftA) + i
		tcell.KeyNames[tcell.Key(r)] = "SHIFT-" + string(int(KeyA)+i)
	}
}

func initAltKeys() {
	for i := 0; i < 27; i++ {
		r := 4 * (int(KeyA) + i)
		tcell.KeyNames[tcell.Key(r)] = "ALT-" + string(int(KeyA)+i)
	}
}

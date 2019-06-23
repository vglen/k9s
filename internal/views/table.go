package views

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/derailed/k9s/internal/config"
	"github.com/derailed/k9s/internal/resource"
	"github.com/derailed/tview"
	"github.com/gdamore/tcell"
	"github.com/rs/zerolog/log"
)

type tableView struct {
	*resTable

	cmdBuff   *cmdBuff
	sortCol   sortColumn
	sortFn    sortFn
	cleanseFn cleanseFn
	filterFn  func(string)
}

func newTableView(app *appView, title string) *tableView {
	v := tableView{
		resTable: newResTable(app, title),
		cmdBuff:  newCmdBuff('/'),
		sortCol:  sortColumn{0, 0, true},
	}
	v.cmdBuff.addListener(app.cmd())
	v.cmdBuff.reset()

	v.SetInputCapture(v.keyboard)
	v.bindKeys()

	return &v
}

func (v *tableView) bindKeys() {
	v.actions = keyActions{
		tcell.KeyCtrlS: newKeyAction("Save", v.saveCmd, true),

		KeySlash:            newKeyAction("Filter Mode", v.activateCmd, false),
		tcell.KeyEscape:     newKeyAction("Filter Reset", v.resetCmd, false),
		tcell.KeyEnter:      newKeyAction("Filter", v.filterCmd, false),
		tcell.KeyBackspace2: newKeyAction("Erase", v.eraseCmd, false),
		tcell.KeyBackspace:  newKeyAction("Erase", v.eraseCmd, false),
		tcell.KeyDelete:     newKeyAction("Erase", v.eraseCmd, false),

		KeyShiftI: newKeyAction("Invert", v.sortInvertCmd, false),
		KeyShiftN: newKeyAction("Sort Name", v.sortColCmd(0), true),
		KeyShiftA: newKeyAction("Sort Age", v.sortColCmd(-1), true),
	}
}

func (v *tableView) handleInput(evt *tcell.EventKey) tcell.Key {
	key := evt.Key()
	if key != tcell.KeyRune {
		return key
	}

	if !v.cmdBuff.isActive() {
		return asKey(evt)
	}
	v.cmdBuff.add(evt.Rune())
	v.clearSelection()
	v.doUpdate(v.filtered())
	v.selectFirstRow()

	return tcell.KeyNUL
}

func (v *tableView) keyboard(evt *tcell.EventKey) *tcell.EventKey {
	key := v.handleInput(evt)
	if key == tcell.KeyNUL {
		return nil
	}

	if a, ok := v.actions[key]; ok {
		log.Debug().Msgf(">> TableView handled %s", tcell.KeyNames[key])
		return a.action(evt)
	}

	return evt
}

func (v *tableView) filterCmd(evt *tcell.EventKey) *tcell.EventKey {
	if v.cmdBuff.isActive() {
		v.cmdBuff.setActive(false)
		cmd := v.cmdBuff.String()
		if isLabelSelector(cmd) && v.filterFn != nil {
			v.filterFn(trimLabelSelector(cmd))
			return nil
		}
		v.refresh()
		return nil
	}

	return evt
}

func (v *tableView) eraseCmd(evt *tcell.EventKey) *tcell.EventKey {
	if v.cmdBuff.isActive() {
		v.cmdBuff.del()
	}

	return nil
}

func (v *tableView) resetCmd(evt *tcell.EventKey) *tcell.EventKey {
	if !v.cmdBuff.empty() {
		v.app.flash().info("Clearing filter...")
	}
	if isLabelSelector(v.cmdBuff.String()) {
		v.filterFn("")
	}
	v.cmdBuff.reset()
	v.refresh()

	return nil
}

func (v *tableView) sortColCmd(col int) func(evt *tcell.EventKey) *tcell.EventKey {
	return func(evt *tcell.EventKey) *tcell.EventKey {
		switch col {
		case -2:
			v.sortCol.index, v.sortCol.asc = 0, true
		case -1:
			v.sortCol.index, v.sortCol.asc = v.GetColumnCount()-1, true
		default:
			v.sortCol.index, v.sortCol.asc = v.nameColIndex()+col, true
		}
		v.refresh()

		return nil
	}
}

func (v *tableView) sortInvertCmd(evt *tcell.EventKey) *tcell.EventKey {
	v.sortCol.asc = !v.sortCol.asc
	v.refresh()

	return nil
}

func (v *tableView) activateCmd(evt *tcell.EventKey) *tcell.EventKey {
	if v.app.inCmdMode() {
		return evt
	}

	v.app.flash().info("Filter mode activated.")
	if isLabelSelector(v.cmdBuff.String()) {
		return nil
	}
	v.cmdBuff.reset()
	v.cmdBuff.setActive(true)

	return nil
}

func (v *tableView) refresh() {
	v.update(v.data)
}

// Update table content
func (v *tableView) update(data resource.TableData) {
	v.currentNS, v.data = data.Namespace, data
	v.showNSBindings(v.sortColCmd(-2))
	v.Clear()
	v.resetTitle()

	if v.cmdBuff.empty() || isLabelSelector(v.cmdBuff.String()) {
		v.doUpdate(v.data)
		return
	}
	v.doUpdate(v.filtered())
}

func (v *tableView) filtered() resource.TableData {
	rx, err := regexp.Compile(`(?i)` + v.cmdBuff.String())
	if err != nil {
		v.app.flash().err(errors.New("Invalid filter expression"))
		v.cmdBuff.clear()
		return v.data
	}

	filtered := resource.TableData{
		Header:    v.data.Header,
		Rows:      resource.RowEvents{},
		Namespace: v.data.Namespace,
	}
	for k, row := range v.data.Rows {
		if rx.MatchString(strings.Join(row.Fields, " ")) {
			filtered.Rows[k] = row
		}
	}

	return filtered
}

func (v *tableView) adjustSorter(data resource.TableData) {
	// Going from namespace to non namespace or vice-versa?
	switch {
	case v.sortCol.colCount == 0:
	case len(data.Header) > v.sortCol.colCount:
		v.sortCol.index++
	case len(data.Header) < v.sortCol.colCount:
		v.sortCol.index--
	}
	v.sortCol.colCount = len(data.Header)
	if v.sortCol.index < 0 {
		v.sortCol.index = 0
	}
}

func (v *tableView) doUpdate(data resource.TableData) {
	v.adjustSorter(data)

	fg := config.AsColor(v.app.styles.Table().Header.FgColor)
	bg := config.AsColor(v.app.styles.Table().Header.BgColor)
	for col, h := range data.Header {
		v.addHeaderCell(data.NumCols[h], col, h)
		c := v.GetCell(0, col)
		c.SetBackgroundColor(bg)
		c.SetTextColor(fg)
	}
	v.sort(data)
}

func (v *tableView) sort(data resource.TableData) {
	pads := make(maxyPad, len(data.Header))
	computeMaxColumns(pads, v.sortCol.index, data)

	sortFn := defaultSort
	if v.sortFn != nil {
		sortFn = v.sortFn
	}
	prim, sec := sortAllRows(v.sortCol, data.Rows, sortFn)
	row := 1
	for _, pk := range prim {
		for _, sk := range sec[pk] {
			v.buildRow(row, data, sk, pads)
			row++
		}
	}
}

func (v *tableView) addHeaderCell(numerical bool, col int, name string) {
	c := tview.NewTableCell(sortIndicator(v.sortCol, v.app.styles.Table(), col, name))
	c.SetExpansion(1)
	if numerical || cpuRX.MatchString(name) || memRX.MatchString(name) {
		c.SetAlign(tview.AlignRight)
	}
	v.SetCell(0, col, c)
}

func (v *tableView) resetTitle() {
	rc := len(v.data.Rows)

	var title string
	switch v.currentNS {
	case resource.NotNamespaced, rbacNS:
		title = skinTitle(fmt.Sprintf(titleFmt, v.baseTitle, rc), v.app.styles.Frame())
	case resource.AllNamespace:
	case resource.AllNamespaces:
		title = skinTitle(fmt.Sprintf(nsTitleFmt, v.baseTitle, resource.AllNamespace, rc), v.app.styles.Frame())
	default:
		title = skinTitle(fmt.Sprintf(nsTitleFmt, v.baseTitle, v.currentNS, rc), v.app.styles.Frame())
	}

	if !v.cmdBuff.isActive() && !v.cmdBuff.empty() {
		title += skinTitle(fmt.Sprintf(searchFmt, v.filterAsStr()), v.app.styles.Frame())
	}
	v.SetTitle(title)
}

func (v *tableView) filterAsStr() string {
	cmd := v.cmdBuff.String()
	if isLabelSelector(cmd) {
		cmd = trimLabelSelector(cmd)
	}
	return cmd
}

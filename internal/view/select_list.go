package view

import (
	"context"

	"github.com/derailed/k9s/internal/model"
	"github.com/derailed/k9s/internal/ui"
	"github.com/derailed/tview"
	"github.com/gdamore/tcell"
)

type selectList struct {
	*tview.List

	parent  Loggable
	actions ui.KeyActions
}

func newSelectList(parent Loggable) *selectList {
	v := selectList{List: tview.NewList(), actions: ui.KeyActions{}}
	{
		v.parent = parent
		v.SetBorder(true)
		v.SetMainTextColor(tcell.ColorWhite)
		v.ShowSecondaryText(false)
		v.SetShortcutColor(tcell.ColorAqua)
		v.SetSelectedBackgroundColor(tcell.ColorAqua)
		v.SetTitle(" [aqua::b]Container Selector ")
		v.SetInputCapture(func(evt *tcell.EventKey) *tcell.EventKey {
			if a, ok := v.actions[evt.Key()]; ok {
				a.Action(evt)
				evt = nil
			}
			return evt
		})
	}

	return &v
}

func (v *selectList) Init(context.Context) {}
func (v *selectList) Start()               {}
func (v *selectList) Stop()                {}
func (v *selectList) Name() string         { return "picker" }

// Protocol...

func (v *selectList) Pop() {
	v.parent.Pop()
}

// SetActions to handle keyboard events.
func (v *selectList) setActions(aa ui.KeyActions) {
	v.actions = aa
}

func (v *selectList) Hints() model.MenuHints {
	if v.actions != nil {
		return v.actions.Hints()
	}

	return nil
}

func (v *selectList) populate(ss []string) {
	v.Clear()
	for i, s := range ss {
		v.AddItem(s, "Select a container", rune('a'+i), nil)
	}
}
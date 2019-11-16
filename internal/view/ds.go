package view

import (
	"github.com/derailed/k9s/internal/k8s"
	"github.com/derailed/k9s/internal/resource"
	"github.com/derailed/k9s/internal/ui"
	"github.com/rs/zerolog/log"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type DaemonSet struct {
	*LogResource

	restartableResource *RestartableResource
}

func NewDaemonSet(title, gvr string, list resource.List) ResourceViewer {
	l := NewLogResource(title, gvr, list)
	d := DaemonSet{
		LogResource:         l,
		restartableResource: newRestartableResourceForParent(l.Resource),
	}
	d.extraActionsFn = d.extraActions
	d.enterFn = d.showPods

	return &d
}

func (d *DaemonSet) extraActions(aa ui.KeyActions) {
	d.LogResource.extraActions(aa)
	d.restartableResource.extraActions(aa)
	aa[ui.KeyShiftD] = ui.NewKeyAction("Sort Desired", d.sortColCmd(1), false)
	aa[ui.KeyShiftC] = ui.NewKeyAction("Sort Current", d.sortColCmd(2), false)
}

func (d *DaemonSet) showPods(app *App, _, res, sel string) {
	ns, n := namespaced(sel)
	dset, err := k8s.NewDaemonSet(app.Conn()).Get(ns, n)
	if err != nil {
		d.app.Flash().Err(err)
		return
	}

	ds, ok := dset.(*appsv1.DaemonSet)
	if !ok {
		log.Fatal().Msg("Expecting a valid ds")
	}
	l, err := metav1.LabelSelectorAsSelector(ds.Spec.Selector)
	if err != nil {
		app.Flash().Err(err)
		return
	}

	showPods(app, ns, l.String(), "")
}
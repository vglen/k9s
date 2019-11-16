package view

import (
	"context"
	"fmt"
	"time"

	"github.com/derailed/k9s/internal/model"
	"github.com/derailed/k9s/internal/resource"
	"github.com/derailed/k9s/internal/ui"
	"github.com/gdamore/tcell"
	"github.com/rs/zerolog/log"
)

const (
	logBuffSize  = 100
	FlushTimeout = 200 * time.Millisecond

	logCoFmt = " Logs([fg:bg:]%s:[hilite:bg:b]%s[-:bg:-]) "
	logFmt   = " Logs([fg:bg:]%s) "
)

// Logs presents a collection of logs.
type Logs struct {
	*ui.Pages

	app        *App
	parent     Loggable
	cancelFunc context.CancelFunc
}

// NewLogs returns a new logs viewer.
func NewLogs(title string, parent Loggable) *Logs {
	return &Logs{
		Pages:  ui.NewPages(),
		parent: parent,
	}
}

func (l *Logs) Init(ctx context.Context) {
	l.app = mustExtractApp(ctx)
}

func (l *Logs) Start()       {}
func (l *Logs) Stop()        {}
func (l *Logs) Name() string { return "logs" }

// Protocol...

func (l *Logs) reload(co string, parent Loggable, prevLogs bool) {
	l.parent = parent
	l.deletePage()
	l.AddPage("logs", NewLog(co, l.app, l.backCmd), true, true)
	l.load(co, prevLogs)
}

func (l *Logs) mustLogViewer() *Log {
	v, ok := l.CurrentPage().Item.(*Log)
	if !ok {
		log.Fatal().Msg("Expecting a log viewer")
	}

	return v
}

// Hints show action hints
func (l *Logs) Hints() model.MenuHints {
	v := l.mustLogViewer()
	return v.actions.Hints()
}

func (l *Logs) deletePage() {
	l.RemovePage("logs")
}

func (l *Logs) stop() {
	if l.cancelFunc == nil {
		return
	}
	l.cancelFunc()
	log.Debug().Msgf("Canceling logs...")
	l.cancelFunc = nil
}

func (l *Logs) load(container string, prevLogs bool) {
	if err := l.doLoad(l.parent.getSelection(), container, prevLogs); err != nil {
		l.app.Flash().Err(err)
		v := l.mustLogViewer()
		v.log("😂 Doh! No logs are available at this time. Check again later on...")
		return
	}
	l.app.SetFocus(l)
}

func (l *Logs) doLoad(path, co string, prevLogs bool) error {
	l.stop()

	v := l.mustLogViewer()
	v.logs.Clear()
	v.setTitle(path, co)

	var ctx context.Context
	ctx = context.WithValue(context.Background(), resource.IKey("informer"), l.app.informer)
	ctx, l.cancelFunc = context.WithCancel(ctx)

	c := make(chan string, 10)
	go updateLogs(ctx, c, v, logBuffSize)

	res, ok := l.parent.getList().Resource().(resource.Tailable)
	if !ok {
		close(c)
		return fmt.Errorf("Resource %T is not tailable", l.parent.getList().Resource())
	}

	if err := res.Logs(ctx, c, l.logOpts(path, co, prevLogs)); err != nil {
		l.cancelFunc()
		close(c)
		return err
	}

	return nil
}

func (l *Logs) logOpts(path, co string, prevLogs bool) resource.LogOptions {
	ns, po := namespaced(path)
	return resource.LogOptions{
		Fqn: resource.Fqn{
			Namespace: ns,
			Name:      po,
			Container: co,
		},
		Lines:    int64(l.app.Config.K9s.LogRequestSize),
		Previous: prevLogs,
	}
}

func updateLogs(ctx context.Context, c <-chan string, l *Log, buffSize int) {
	defer func() {
		log.Debug().Msgf("updateLogs view bailing out!")
	}()
	buff, index := make([]string, buffSize), 0
	for {
		select {
		case line, ok := <-c:
			if !ok {
				log.Debug().Msgf("Closed channel detected. Bailing out...")
				l.Flush(index, buff)
				return
			}
			if index < buffSize {
				buff[index] = line
				index++
				continue
			}
			l.Flush(index, buff)
			index = 0
			buff[index] = line
			index++
		case <-time.After(FlushTimeout):
			l.Flush(index, buff)
			index = 0
		case <-ctx.Done():
			return
		}
	}
}

// ----------------------------------------------------------------------------
// Actions...

func (l *Logs) backCmd(evt *tcell.EventKey) *tcell.EventKey {
	l.stop()
	l.parent.Pop()

	return evt
}
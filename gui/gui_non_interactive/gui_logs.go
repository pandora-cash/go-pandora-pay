package gui_non_interactive

import (
	"fmt"
	"pandora-pay/gui/gui_interface"
)

func (g *GUINonInteractive) message(prefix string, color string, any ...interface{}) {
	text := gui_interface.ProcessArgument(any...)
	final := prefix + " " + color + " " + text

	g.writingMutex.Lock()
	fmt.Println(final)
	g.writingMutex.Unlock()
}

func (g *GUINonInteractive) Log(any ...any) {
	g.message("LOG", g.colorLog, any...)
}

func (g *GUINonInteractive) Info(any ...any) {
	g.message("INF", g.colorInfo, any...)
}

func (g *GUINonInteractive) Warning(any ...any) {
	g.message("WARN", g.colorWarning, any...)
}

func (g *GUINonInteractive) Fatal(any ...any) {
	g.message("FATAL", g.colorFatal, any...)
	panic(any)
}

func (g *GUINonInteractive) Error(any ...any) {
	g.message("ERR", g.colorError, any...)
}

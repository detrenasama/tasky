//go:build windows

package main

import (
	"github.com/getlantern/systray"
)

// Run запускает системный трей Windows и блокирует поток до выхода.
// systray на Windows — чистый Go (без cgo), собирается с CGO_ENABLED=0.
func Run() {
	systray.Run(onReady, onExit)
}

func onReady() {
	systray.SetTitle("Tasky")
	systray.SetTooltip("Tasky — сервер задач")

	mStart := systray.AddMenuItem("Запустить сервер", "Запустить сервер Tasky")
	mStop := systray.AddMenuItem("Остановить сервер", "Остановить сервер Tasky")
	mOpen := systray.AddMenuItem("Открыть в браузере", "Открыть веб-интерфейс")
	mQuit := systray.AddMenuItem("Выход", "Выйти из индикатора")

	mStop.Disable()

	go func() {
		for {
			select {
			case <-mStart.ClickedCh:
				if err := startServer(); err == nil {
					mStart.Disable()
					mStop.Enable()
				} else {
					systray.SetTooltip("Tasky: " + err.Error())
				}
			case <-mStop.ClickedCh:
				if err := stopServer(); err == nil {
					mStop.Disable()
					mStart.Enable()
				} else {
					systray.SetTooltip("Tasky: " + err.Error())
				}
			case <-mOpen.ClickedCh:
				if err := openBrowser(); err != nil {
					systray.SetTooltip("Tasky: " + err.Error())
				}
			case <-mQuit.ClickedCh:
				_ = stopServer()
				systray.Quit()
				return
			}
		}
	}()
}

func onExit() {
	_ = stopServer()
}

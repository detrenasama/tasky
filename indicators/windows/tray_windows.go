//go:build windows

package main

import (
	"time"

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

	// Синхронизация меню с реальным состоянием сервера (HTTP + owned).
	syncMenu := func() {
		reachable := serverReachable()
		owned := serverCmd != nil && serverCmd.Process != nil
		canStart := !reachable && !owned
		canStop := reachable && owned
		if canStart {
			mStart.Enable()
		} else {
			mStart.Disable()
		}
		if canStop {
			mStop.Enable()
		} else {
			mStop.Disable()
		}
	}

	// Первичная синхронизация — если сервер уже запущен извне (консоль),
	// сразу покажем правильное состояние.
	syncMenu()

	// Фондовый опрос — раз в секунду, как в GNOME extension.js:19.
	go func() {
		ticker := time.NewTicker(1 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			syncMenu()
		}
	}()

	go func() {
		for {
			select {
			case <-mStart.ClickedCh:
				if err := startServer(); err != nil {
					systray.SetTooltip("Tasky: " + err.Error())
				}
				// Состояние обновится на следующем тике; форсируем.
				reachable := serverReachable()
				owned := serverCmd != nil && serverCmd.Process != nil
				if !reachable && !owned {
					mStart.Enable()
				} else {
					mStart.Disable()
				}
				if reachable && owned {
					mStop.Enable()
				} else {
					mStop.Disable()
				}
			case <-mStop.ClickedCh:
				if err := stopServer(); err != nil {
					systray.SetTooltip("Tasky: " + err.Error())
				}
				reachable := serverReachable()
				owned := serverCmd != nil && serverCmd.Process != nil
				if !reachable && !owned {
					mStart.Enable()
				} else {
					mStart.Disable()
				}
				if reachable && owned {
					mStop.Enable()
				} else {
					mStop.Disable()
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

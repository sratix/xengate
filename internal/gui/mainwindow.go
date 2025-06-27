//go:build !(android || ios)
// +build !android,!ios

package gui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

func mainWindow(s *FyneScreen) fyne.CanvasObject {
	// w := s.Current

	// fynePE := &fyne.PointEvent{
	// 	AbsolutePosition: fyne.Position{
	// 		X: 10,
	// 		Y: 30,
	// 	},
	// 	Position: fyne.Position{
	// 		X: 10,
	// 		Y: 30,
	// 	},
	// }

	// w.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
	// 	if !s.Hotkeys {
	// 		return
	// 	}

	// 	if k.Name == "Space" || k.Name == "P" {
	// 		currentState := s.getScreenState()
	// 		switch currentState {
	// 		case "Playing":
	// 			go s.PlayPause.Tapped(fynePE)
	// 		case "Paused", "Stopped", "":
	// 			go s.PlayPause.Tapped(fynePE)
	// 		}
	// 	}

	// 	if k.Name == "S" {
	// 		go s.Stop.Tapped(fynePE)
	// 	}

	// 	if k.Name == "M" {
	// 		s.MuteUnmute.Tapped(fynePE)
	// 	}

	// 	if k.Name == "Prior" {
	// 		s.VolumeUp.Tapped(fynePE)
	// 	}

	// 	if k.Name == "Next" {
	// 		s.VolumeDown.Tapped(fynePE)
	// 	}
	// })

	// sfiletext := widget.NewEntry()
	// sfiletext.Disable()

	content := container.NewVBox(
		widget.NewLabelWithStyle("Xengate Media Player", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewLabel("Version: "+s.version),
		widget.NewLabel("State: "+s.State),
	)

	return content
}

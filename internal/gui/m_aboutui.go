package gui

import (
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// show info about app
func (a *App) aboutDialog() {
	logo := canvas.NewImageFromResource(theme.FyneLogo())
	logoRow := container.NewBorder(nil, nil, logo, nil)
	infoRow := widget.NewRichTextFromMarkdown(`
## xengate - stream media to UPnP/DLNA devices
---
Use xengate to stream media to UPnP/DLNA devices.

---`)
	noteRow := widget.NewRichTextFromMarkdown(`
---
*Created using* [Fyne](https://fyne.io) *GUI library*.

*App icon designed by* [Icon8](https://icon8.com).`)

	aboutDialog := dialog.NewCustom("About", "Ok", container.NewVBox(logoRow, infoRow, nil, noteRow), a.topWindow)
	aboutDialog.SetOnClosed(func() {})
	aboutDialog.Show()
}

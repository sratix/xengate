package dialogs

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
)

func About(app fyne.App, window fyne.Window) {
	logo := canvas.NewImageFromResource(fyne.NewStaticResource("icon", go2TVIcon512))

	logo.Resize(fyne.NewSize(128, 128))
	logo.SetMinSize(fyne.NewSize(128, 128))
	logo.Move(fyne.NewPos(0, 0))
	logoRow := container.NewCenter(logo)
	infoRow := widget.NewRichTextFromMarkdown(`
## XenGate - photo picker
---
Use [Pixyne](http://vinser.github.io/pixyne) to quickly review your photo folder and safely delete bad and similar shots.

You may also fix EXIF the shooting dates, crop and ajust photos.

---`)

	var buildVersion, buildFor, buildTime, goVersion, versionLine, buildLine string
	if buildVersion = app.Metadata().Version; buildVersion == "" {
		buildVersion = "selfcrafted"
	}
	versionLine = "Version: " + buildVersion

	if buildFor = app.Metadata().Custom["BuildForOS"]; buildFor != "" {
		buildLine = fmt.Sprintf("Build for: %s ", buildFor)
	}
	if buildTime = app.Metadata().Custom["BuildTime"]; buildTime != "" {
		buildLine = buildLine + fmt.Sprintf(" | Build time: %s ", buildTime)
	}
	if goVersion = app.Metadata().Custom["GoVersion"]; goVersion != "" {
		buildLine = buildLine + fmt.Sprintf(" | Go version: %s ", goVersion)
	}

	tecRow := widget.NewRichTextFromMarkdown(
		`Licence: MIT | [GitHub](https://github.com/vinser/pixyne) repo | ` + versionLine + `

` + buildLine)

	noteRow := widget.NewRichTextFromMarkdown(`
---
*Created using* [Fyne](https://fyne.io) *GUI library*.`)

	aboutDialog := dialog.NewCustom("About", "Ok", container.NewVBox(logoRow, infoRow, tecRow, noteRow), window)

	aboutDialog.Show()
}

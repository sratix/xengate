//go:build !(android || ios)
// +build !android,!ios

// Package gui provides graphical user interface functionality.
package gui

import (
	"errors"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

const (
	versionCheckerTitle     = "Version checker"
	versionInfoError        = "failed to get version info"
	internetConnectionError = "check your internet connection"
)

func aboutWindow(w *AppWindow) fyne.CanvasObject {
	iconResource := fyne.NewStaticResource("Go2TV Icon", go2TVIcon512)
	iconImage := canvas.NewImageFromResource(iconResource)
	description := widget.NewRichTextFromMarkdown(`
` + lang.L("Cast your media files to UPnP/DLNA Media Renderers and Smart TVs") + `

---

## ` + lang.L("Author") + `
Alex Ballas - alex@ballas.org

## ` + lang.L("License") + `
MIT

## ` + lang.L("Version") + `

` + w.version)

	for _, segment := range description.Segments {
		if textSegment, ok := segment.(*widget.TextSegment); ok {
			textSegment.Style.Alignment = fyne.TextAlignCenter
		}
		if hyperlinkSegment, ok := segment.(*widget.HyperlinkSegment); ok {
			hyperlinkSegment.Alignment = fyne.TextAlignCenter
		}
	}

	githubButton := widget.NewButton(lang.L("Github page"), func() {
		go func() {
			url, _ := url.Parse("https://github.com/alexballas/go2tv")
			_ = fyne.CurrentApp().OpenURL(url)
		}()
	})

	versionButton := widget.NewButton(lang.L("Check version"), func() {
		go checkVersion(w)
	})

	w.CheckVersion = versionButton

	iconImage.SetMinSize(fyne.Size{Width: 64, Height: 64})

	content := container.NewVBox(
		container.NewCenter(iconImage),
		container.NewCenter(description),
		container.NewCenter(container.NewHBox(githubButton, versionButton)),
	)

	return container.NewPadded(content)
}

func checkVersion(w *AppWindow) {
	w.CheckVersion.Disable()
	defer w.CheckVersion.Enable()

	client := &http.Client{
		Timeout: 3 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://github.com/alexballas/Go2TV/releases/latest", nil)
	if err != nil {
		dialog.ShowError(errors.New(lang.L(versionInfoError)+" - "+lang.L(internetConnectionError)), w.window)
		return
	}

	resp, err := client.Do(req)
	if err != nil {
		dialog.ShowError(errors.New(lang.L(versionInfoError)+" - "+lang.L(internetConnectionError)), w.window)
		return
	}

	defer resp.Body.Close()

	location, err := resp.Location()
	if err != nil {
		dialog.ShowError(errors.New(lang.L(versionInfoError)+" - "+lang.L("you're using a development or a non-compiled version")), w.window)
		return
	}

	currentVersionStr := strings.ReplaceAll(w.version, ".", "")
	currentVersion, err := strconv.Atoi(currentVersionStr)
	if err != nil {
		dialog.ShowError(errors.New(lang.L(versionInfoError)+" - "+lang.L("you're using a development or a non-compiled version")), w.window)
		return
	}

	latestVersionStr := filepath.Base(location.Path)
	latestVersionStr = strings.Trim(latestVersionStr, "v")
	latestVersion, err := strconv.Atoi(strings.ReplaceAll(latestVersionStr, ".", ""))
	if err != nil {
		dialog.ShowError(errors.New(lang.L(versionInfoError)+" - "+lang.L(internetConnectionError)), w.window)
		return
	}

	switch {
	case latestVersion > currentVersion:
		dialog.ShowInformation(lang.L(versionCheckerTitle), lang.L("New version")+": "+latestVersionStr, w.window)
	case latestVersion == currentVersion:
		dialog.ShowInformation(lang.L(versionCheckerTitle), lang.L("No new version"), w.window)
	default:
		dialog.ShowInformation(lang.L(versionCheckerTitle), lang.L("New version")+": "+latestVersionStr, w.window)
	}
}

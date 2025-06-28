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

func aboutWindow(s *FyneScreen) fyne.CanvasObject {
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

` + s.version)

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
		go checkVersion(s)
	})

	s.CheckVersion = versionButton

	iconImage.SetMinSize(fyne.Size{Width: 64, Height: 64})

	content := container.NewVBox(
		container.NewCenter(iconImage),
		container.NewCenter(description),
		container.NewCenter(container.NewHBox(githubButton, versionButton)),
	)

	return container.NewPadded(content)
}

func checkVersion(s *FyneScreen) {
	s.CheckVersion.Disable()
	defer s.CheckVersion.Enable()

	httpClient := &http.Client{
		Timeout: 3 * time.Second,
	}

	req, err := http.NewRequest("GET", "https://github.com/alexballas/Go2TV/releases/latest", nil)
	if err != nil {
		dialog.ShowError(errors.New(lang.L("failed to get version info")+" - "+lang.L("check your internet connection")), s.Current)
		return
	}

	resp, err := httpClient.Do(req)
	if err != nil {
		dialog.ShowError(errors.New(lang.L("failed to get version info")+" - "+lang.L("check your internet connection")), s.Current)
		return
	}

	defer resp.Body.Close()

	location, err := resp.Location()
	if err != nil {
		dialog.ShowError(errors.New(lang.L("failed to get version info")+" - "+lang.L("you're using a development or a non-compiled version")), s.Current)
		return
	}

	currentVersion, err := strconv.Atoi(strings.ReplaceAll(s.version, ".", ""))
	if err != nil {
		dialog.ShowError(errors.New(lang.L("failed to get version info")+" - "+lang.L("you're using a development or a non-compiled version")), s.Current)
		return
	}

	latestVersionStr := filepath.Base(location.Path)
	latestVersionStr = strings.Trim(latestVersionStr, "v")
	latestVersion, err := strconv.Atoi(strings.ReplaceAll(latestVersionStr, ".", ""))
	if err != nil {
		dialog.ShowError(errors.New(lang.L("failed to get version info")+" - "+lang.L("check your internet connection")), s.Current)
		return
	}

	switch {
	case latestVersion > currentVersion:
		dialog.ShowInformation(lang.L("Version checker"), lang.L("New version")+": "+latestVersionStr, s.Current)
	case latestVersion == currentVersion:
		dialog.ShowInformation(lang.L("Version checker"), lang.L("No new version"), s.Current)
	default:
		dialog.ShowInformation(lang.L("Version checker"), lang.L("New version")+": "+latestVersionStr, s.Current)
	}
}

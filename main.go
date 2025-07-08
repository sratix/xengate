package main

import (
	"log"
	"os"
	"slices"

	"xengate/backend"
	"xengate/res"
	"xengate/ui"
	"xengate/ui/util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/lang"
	_ "go.uber.org/automaxprocs"
)

func main() {
	myApp, err := backend.StartupApp(res.AppName, res.DisplayName, res.AppVersion, res.AppVersionTag, res.LatestReleaseURL)
	if err != nil {
		if err != backend.ErrAnotherInstance {
			log.Fatalf("fatal startup error: %v", err.Error())
		}
		return
	}

	switch myApp.Config.Application.UIScaleSize {
	case "Smaller":
		os.Setenv("FYNE_SCALE", "0.85")
	case "Larger":
		os.Setenv("FYNE_SCALE", "1.1")
	}

	if myApp.Config.Application.DisableDPIDetection {
		os.Setenv("FYNE_DISABLE_DPI_DETECTION", "true")
	}

	// load configured app language, or all otherwise
	lIdx := slices.IndexFunc(res.TranslationsInfo, func(t res.TranslationInfo) bool {
		return t.Name == myApp.Config.Application.Language
	})
	success := false
	if lIdx >= 0 {
		tr := res.TranslationsInfo[lIdx]
		content, err := res.Translations.ReadFile("translations/" + tr.TranslationFileName)
		if err == nil {
			// "trick" Fyne into loading translations for configured language
			// by pretending it's the translation for the system locale
			name := lang.SystemLocale().LanguageString()
			lang.AddTranslations(fyne.NewStaticResource(name+".json", content))
			success = true
		} else {
			log.Printf("Error loading translation file %s: %s\n", tr.TranslationFileName, err.Error())
		}
	}
	if !success {
		if err := lang.AddTranslationsFS(res.Translations, "translations"); err != nil {
			log.Printf("Error loading translations: %s", err.Error())
		}
	}

	fyneApp := app.New()
	fyneApp.SetIcon(fyne.NewStaticResource("icon", ui.Go2TVIcon512))

	mainWindow := ui.NewMainWindow(fyneApp, res.AppName, res.DisplayName, res.AppVersion, myApp)
	mainWindow.Window.SetMaster()
	myApp.OnReactivate = util.FyneDoFunc(mainWindow.Show)
	myApp.OnExit = util.FyneDoFunc(mainWindow.Quit)

	mainWindow.ShowAndRun()

	log.Println("Running shutdown tasks...")
	myApp.Shutdown()
}

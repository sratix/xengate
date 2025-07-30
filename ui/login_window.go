package ui

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/BurntSushi/toml"
)

type LoginConfig struct {
	ParentHash string `toml:"parent_hash"`
}

type LoginWindow struct {
	window     fyne.Window
	dialog     *widget.PopUp
	onComplete func(isParent bool)
	config     *LoginConfig
	configPath string
}

func NewLoginWindow(parent fyne.Window) *LoginWindow {
	w := &LoginWindow{
		window: parent,
		config: &LoginConfig{},
	}

	configDir, _ := os.UserConfigDir()
	w.configPath = filepath.Join(configDir, "xengate", "login.toml")
	w.loadConfig()
	w.createUI()

	return w
}

func (w *LoginWindow) loadConfig() {
	os.MkdirAll(filepath.Dir(w.configPath), 0o755)
	if _, err := os.Stat(w.configPath); os.IsNotExist(err) {
		w.saveConfig()
		return
	}
	_, err := toml.DecodeFile(w.configPath, w.config)
	if err != nil {
		fmt.Printf("Error loading config: %v\n", err)
	}
}

func (w *LoginWindow) saveConfig() {
	f, err := os.Create(w.configPath)
	if err != nil {
		fmt.Printf("Error creating config file: %v\n", err)
		return
	}
	defer f.Close()
	encoder := toml.NewEncoder(f)
	if err := encoder.Encode(w.config); err != nil {
		fmt.Printf("Error saving config: %v\n", err)
	}
}

func (w *LoginWindow) createWelcomeHeader() *fyne.Container {
	logo := canvas.NewText("X", theme.PrimaryColor())
	logo.TextSize = 48
	logo.TextStyle.Bold = true

	title := canvas.NewText("Welcome to Xengate", theme.ForegroundColor())
	title.TextSize = 24
	title.TextStyle.Bold = true

	subtitle := widget.NewLabel("Please select your access mode")
	subtitle.TextStyle = fyne.TextStyle{Italic: true}

	return container.NewVBox(
		container.NewCenter(logo),
		container.NewCenter(title),
		container.NewCenter(subtitle),
	)
}

func (w *LoginWindow) createModeCard(title, description string, icon fyne.Resource, isPrimary bool, onTap func()) fyne.CanvasObject {
	iconContainer := container.NewCenter(widget.NewIcon(icon))

	titleLabel := widget.NewLabel(title)
	titleLabel.TextStyle = fyne.TextStyle{Bold: true}
	titleLabel.Alignment = fyne.TextAlignCenter

	descLabel := widget.NewLabel(description)
	descLabel.Alignment = fyne.TextAlignCenter
	descLabel.Wrapping = fyne.TextWrapWord

	var btn *widget.Button
	if isPrimary {
		btn = widget.NewButtonWithIcon("Enter as Parent", theme.LoginIcon(), onTap)
		btn.Importance = widget.HighImportance
	} else {
		btn = widget.NewButtonWithIcon("Enter as Child", theme.AccountIcon(), onTap)
		btn.Importance = widget.MediumImportance
	}

	content := container.NewVBox(
		iconContainer,
		titleLabel,
		descLabel,
		container.NewCenter(btn),
	)

	card := widget.NewCard("", "", content)
	return container.NewPadded(card)
}

func (w *LoginWindow) createUI() {
	header := w.createWelcomeHeader()

	parentCard := w.createModeCard(
		"Parent Mode",
		"Full access and control\nManage settings and restrictions",
		theme.HomeIcon(),
		true,
		func() {
			if w.config.ParentHash == "" {
				w.showSetPasswordDialog()
			} else {
				w.showPasswordDialog()
			}
		},
	)

	childCard := w.createModeCard(
		"Child Mode",
		"Limited access\nSafe and controlled environment",
		theme.AccountIcon(),
		false,
		func() {
			if w.onComplete != nil {
				w.onComplete(false)
			}
			w.dialog.Hide()
		},
	)

	cardsContainer := container.NewHBox(
		parentCard,
		widget.NewSeparator(),
		childCard,
	)

	footer := widget.NewLabel("© 2025 Xengate. All rights reserved.")
	footer.Alignment = fyne.TextAlignCenter
	footer.TextStyle = fyne.TextStyle{Italic: true}

	content := container.NewVBox(
		header,
		widget.NewSeparator(),
		container.NewCenter(cardsContainer),
		widget.NewSeparator(),
		footer,
	)

	w.dialog = widget.NewModalPopUp(
		container.NewPadded(content),
		w.window.Canvas(),
	)
	w.dialog.Resize(fyne.NewSize(600, 400))
}

func (w *LoginWindow) createPasswordForm(isNewPassword bool) (*widget.Entry, *widget.Entry, *fyne.Container) {
	input := widget.NewPasswordEntry()
	input.PlaceHolder = "Enter password"

	var confirm *widget.Entry
	var formItems []*widget.FormItem

	if isNewPassword {
		confirm = widget.NewPasswordEntry()
		confirm.PlaceHolder = "Confirm password"
		formItems = []*widget.FormItem{
			{Text: "New Password", Widget: input},
			{Text: "Confirm Password", Widget: confirm},
		}
	} else {
		formItems = []*widget.FormItem{
			{Text: "Password", Widget: input},
		}
	}

	form := &widget.Form{
		Items:      formItems,
		SubmitText: "",
		OnSubmit:   nil,
	}

	return input, confirm, container.NewVBox(form)
}

func (w *LoginWindow) showSetPasswordDialog() {
	// Hide main dialog first
	w.dialog.Hide()

	content := container.NewVBox(
		widget.NewRichTextFromMarkdown("# Create Parent Password\n\nSet a secure password for parent access. This password will be required for future parent mode access."),
	)

	input, confirm, formContainer := w.createPasswordForm(true)

	submitBtn := widget.NewButtonWithIcon("Set Password", theme.ConfirmIcon(), nil)
	submitBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), nil)
	cancelBtn.Importance = widget.MediumImportance

	buttons := container.NewHBox(
		cancelBtn,
		submitBtn,
	)

	var customDialog dialog.Dialog

	submitBtn.OnTapped = func() {
		if input.Text != confirm.Text {
			dialog.ShowError(fmt.Errorf("passwords do not match"), w.window)
			return
		}
		if len(input.Text) < 6 {
			dialog.ShowError(fmt.Errorf("password must be at least 6 characters"), w.window)
			return
		}

		hash := sha256.Sum256([]byte(input.Text))
		w.config.ParentHash = hex.EncodeToString(hash[:])
		w.saveConfig()

		customDialog.Hide()
		if w.onComplete != nil {
			w.onComplete(true)
		}
	}

	cancelBtn.OnTapped = func() {
		customDialog.Hide()
		w.dialog.Show()
	}

	combinedContent := container.NewVBox(
		content,
		widget.NewSeparator(),
		formContainer,
		container.NewCenter(buttons),
	)

	customDialog = dialog.NewCustomWithoutButtons(
		"Set Parent Password",
		combinedContent,
		w.window,
	)
	customDialog.Resize(fyne.NewSize(400, 300))
	customDialog.Show()
}

func (w *LoginWindow) showPasswordDialog() {
	// Hide main dialog first
	w.dialog.Hide()

	content := container.NewVBox(
		widget.NewRichTextFromMarkdown("# Parent Access\n\nPlease enter your parent access password to continue."),
	)

	input, _, formContainer := w.createPasswordForm(false)

	loginBtn := widget.NewButtonWithIcon("Login", theme.LoginIcon(), nil)
	loginBtn.Importance = widget.HighImportance

	cancelBtn := widget.NewButtonWithIcon("Cancel", theme.CancelIcon(), nil)

	cancelBtn.Importance = widget.MediumImportance

	buttons := container.NewHBox(
		cancelBtn,
		loginBtn,
	)

	var customDialog dialog.Dialog

	validatePassword := func() {
		hash := sha256.Sum256([]byte(input.Text))
		inputHash := hex.EncodeToString(hash[:])

		if inputHash == w.config.ParentHash {
			customDialog.Hide()
			if w.onComplete != nil {
				w.onComplete(true)
			}
		} else {
			dialog.ShowError(fmt.Errorf("incorrect password"), w.window)
		}
	}

	loginBtn.OnTapped = validatePassword
	input.OnSubmitted = func(string) { validatePassword() }

	cancelBtn.OnTapped = func() {
		customDialog.Hide()
		w.dialog.Show()
	}

	combinedContent := container.NewVBox(
		content,
		widget.NewSeparator(),
		formContainer,
		container.NewCenter(buttons),
	)

	customDialog = dialog.NewCustomWithoutButtons(
		"Parent Access",
		combinedContent,
		w.window,
	)
	customDialog.Resize(fyne.NewSize(400, 300))
	customDialog.Show()
}

func (w *LoginWindow) SetOnComplete(callback func(isParent bool)) {
	w.onComplete = callback
}

func (w *LoginWindow) Show() {
	w.dialog.Show()
}

func (w *LoginWindow) Hide() {
	w.dialog.Hide()
}

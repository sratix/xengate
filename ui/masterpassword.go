package ui

import (
	"errors"

	"xengate/backend"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"github.com/zalando/go-keyring"
)

const (
	masterPassKey     = "xengate-master-password"
	masterPassService = "xengate-app"
)

type MasterPasswordManager struct {
	app     fyne.App
	window  fyne.Window
	storage *backend.AppStorage
}

func NewMasterPasswordManager(app fyne.App, window fyne.Window, storage *backend.AppStorage) *MasterPasswordManager {
	return &MasterPasswordManager{
		app:     app,
		window:  window,
		storage: storage,
	}
}

func (m *MasterPasswordManager) ShowMasterPasswordDialog(callback func(string)) {
	// چک کردن پسورد ذخیره شده
	savedPass, err := keyring.Get(masterPassService, masterPassKey)
	if err == nil && savedPass != "" {
		callback(savedPass)
		return
	}

	// content := container.NewVBox()
	passEntry := widget.NewPasswordEntry()
	confirmEntry := widget.NewPasswordEntry()
	saveCheckbox := widget.NewCheck("Remember master password", nil)

	form := &widget.Form{
		Items: []*widget.FormItem{
			{Text: "Master Password", Widget: passEntry},
			{Text: "Confirm Password", Widget: confirmEntry},
			{Text: "", Widget: saveCheckbox},
		},
		OnSubmit: func() {
			if passEntry.Text == "" {
				dialog.ShowError(errors.New("password cannot be empty"), m.window)
				return
			}
			if passEntry.Text != confirmEntry.Text {
				dialog.ShowError(errors.New("passwords do not match"), m.window)
				return
			}

			// ذخیره پسورد در keyring اگر کاربر خواسته باشد
			if saveCheckbox.Checked {
				err := keyring.Set(masterPassService, masterPassKey, passEntry.Text)
				if err != nil {
					dialog.ShowError(err, m.window)
					return
				}
			}

			callback(passEntry.Text)
		},
		OnCancel: func() {
			m.app.Quit()
		},
	}

	dialog := dialog.NewCustom("Set Master Password", "Cancel", form, m.window)
	dialog.Show()
}

func (m *MasterPasswordManager) ClearSavedPassword() error {
	return keyring.Delete(masterPassService, masterPassKey)
}

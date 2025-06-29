//go:build !windows
// +build !windows

package gui

// NewFyneUI .
func NewFyneUI(version string) *FyneUI {
	return newGUI(version)
}

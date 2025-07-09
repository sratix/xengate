package components

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// Declare conformity with Layout interface
var _ fyne.Layout = (*badgeLayout)(nil)

type badgeLayout struct{}

// Layout is called to pack all child objects into a specified size.
func (l badgeLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	padding := theme.Padding()
	pos := fyne.NewPos(padding, padding*2)
	siz := fyne.NewSize(size.Width-0*padding, size.Height-4*padding)
	for _, child := range objects {
		child.Resize(siz)
		if _, ok := child.(*canvas.Text); ok {
			child.Move(fyne.NewPos(pos.X, pos.Y-1.2))
			continue
		}
		child.Move(pos)
	}
}

// MinSize finds the smallest size that satisfies all the child objects.
func (l badgeLayout) MinSize(objects []fyne.CanvasObject) (min fyne.Size) {
	for _, child := range objects {
		if !child.Visible() {
			continue
		}

		min = min.Max(child.MinSize())
	}
	min = min.Add(fyne.NewSize(2*theme.Padding(), 4*theme.Padding()))
	return
}

// NewBadge creates a badge widget container. Badge is a small colored text container, usually used to represent tags.
func NewBadge(label string, clr color.Color) *fyne.Container {
	back := &canvas.Rectangle{
		StrokeColor:  clr,
		StrokeWidth:  .5,
		CornerRadius: 4,
	}
	text := &canvas.Text{
		Color:     clr,
		Text:      label,
		Alignment: fyne.TextAlignCenter,
		TextSize:  theme.TextSize() * 0.75,
	}

	return container.New(badgeLayout{}, text, back)
}

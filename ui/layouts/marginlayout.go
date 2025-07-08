package layouts

import "fyne.io/fyne/v2"

type MarginLayout struct {
	MarginTop    float32
	MarginBottom float32
	MarginLeft   float32
	MarginRight  float32
}

func (m *MarginLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	w, h := float32(0), float32(0)
	for _, o := range objects {
		childSize := o.MinSize()
		w = fyne.Max(w, childSize.Width)
		h = fyne.Max(h, childSize.Height)
	}
	return fyne.NewSize(w, h+m.MarginTop+m.MarginBottom)
}

func (m *MarginLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, o := range objects {
		o.Resize(fyne.NewSize(size.Width, size.Height-m.MarginTop-m.MarginBottom))
		o.Move(fyne.NewPos(0, m.MarginTop))
	}
}

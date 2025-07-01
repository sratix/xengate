package components

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type AccordionItem struct {
	title     string
	content   *fyne.Container
	isOpen    bool
	indicator *canvas.Rectangle
	fields    []string
	data      map[string]string
}

type CustomAccordion struct {
	widget.BaseWidget
	items     []*AccordionItem
	container *fyne.Container
}

func NewAccordionItem(title string, fields []string) *AccordionItem {
	indicator := canvas.NewRectangle(color.NRGBA{R: 255, A: 255})
	indicator.Resize(fyne.NewSize(4, 40))

	return &AccordionItem{
		title:     title,
		fields:    fields,
		isOpen:    false,
		indicator: indicator,
		data:      make(map[string]string),
	}
}

func NewCustomAccordion() *CustomAccordion {
	acc := &CustomAccordion{
		items: make([]*AccordionItem, 0),
	}
	acc.ExtendBaseWidget(acc)
	return acc
}

func (a *CustomAccordion) AddItem(title string, fields []string) {
	item := NewAccordionItem(title, fields)

	form := &widget.Form{
		SubmitText: "ثبت",
		CancelText: "انصراف",
	}

	var formItems []*widget.FormItem
	for _, field := range fields {
		entry := widget.NewEntry()
		entry.SetPlaceHolder(field)
		formItems = append(formItems, widget.NewFormItem(field, entry))
	}
	form.Items = formItems

	form.OnSubmit = func() {
		for _, formItem := range form.Items {
			entry := formItem.Widget.(*widget.Entry)
			item.data[formItem.Text] = entry.Text
		}
		fmt.Printf("Form submitted for %s: %v\n", title, item.data)
		item.isOpen = false
		a.Refresh()
	}

	form.OnCancel = func() {
		item.isOpen = false
		a.Refresh()
	}

	content := container.NewVBox(form)
	content.Hide()
	item.content = content

	a.items = append(a.items, item)
	a.Refresh()
}

func (a *CustomAccordion) createItemWidget(item *AccordionItem) fyne.CanvasObject {
	titleBtn := widget.NewButton(item.title, func() {
		for _, i := range a.items {
			if i != item {
				i.isOpen = false
				i.content.Hide()
				i.indicator.FillColor = color.NRGBA{R: 255, A: 255}
				i.indicator.Refresh()
			}
		}

		item.isOpen = !item.isOpen
		if item.isOpen {
			item.content.Show()
			item.indicator.FillColor = color.NRGBA{G: 255, A: 255}
		} else {
			item.content.Hide()
			item.indicator.FillColor = color.NRGBA{R: 255, A: 255}
		}
		item.indicator.Refresh()
		a.Refresh()
	})

	header := container.NewBorder(
		nil, nil,
		item.indicator,
		nil,
		titleBtn,
	)

	return container.NewVBox(
		header,
		item.content,
	)
}

// حذف MinSize
// func (a *CustomAccordion) MinSize() fyne.Size {
//     return fyne.NewSize(300, 400)
// }

func (a *CustomAccordion) CreateRenderer() fyne.WidgetRenderer {
	vbox := container.NewVBox()

	for _, item := range a.items {
		vbox.Add(a.createItemWidget(item))
	}

	return widget.NewSimpleRenderer(vbox)
}

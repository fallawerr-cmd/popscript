package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
)

type CallbackFunc func(args []string)

type UI struct {
	app     fyne.App
	window  fyne.Window
	widgets map[string]fyne.CanvasObject
	content []fyne.CanvasObject
}

var Instance *UI

func New() *UI {
	Instance = &UI{
		widgets: make(map[string]fyne.CanvasObject),
	}
	return Instance
}

func (u *UI) Window(title string, width, height float32) {
	u.app = app.New()
	u.window = u.app.NewWindow(title)
	u.window.Resize(fyne.NewSize(width, height))
}

func (u *UI) Label(text string) {
	l := widget.NewLabel(text)
	u.content = append(u.content, l)
}

func (u *UI) LabelNamed(name string, text string) {
	l := widget.NewLabel(text)
	u.widgets[name] = l
	u.content = append(u.content, l)
}

func (u *UI) Button(text string, action CallbackFunc) {
	b := widget.NewButton(text, func() {
		if action != nil {
			action(nil)
		}
	})
	u.content = append(u.content, b)
}

func (u *UI) ButtonNamed(name string, text string, action CallbackFunc) {
	b := widget.NewButton(text, func() {
		if action != nil {
			action(nil)
		}
	})
	u.widgets[name] = b
	u.content = append(u.content, b)
}

func (u *UI) Input(name string, placeholder string, onChange CallbackFunc) {
	e := widget.NewEntry()
	e.SetPlaceHolder(placeholder)
	if onChange != nil {
		e.OnChanged = func(s string) {
			onChange([]string{s})
		}
	}
	u.widgets[name] = e
	u.content = append(u.content, e)
}

func (u *UI) Checkbox(text string, action CallbackFunc) {
	c := widget.NewCheck(text, func(checked bool) {
		if action != nil {
			v := "false"
			if checked {
				v = "true"
			}
			action([]string{v})
		}
	})
	u.content = append(u.content, c)
}

func (u *UI) Get(name string) string {
	w, ok := u.widgets[name]
	if !ok {
		return ""
	}
	if e, ok := w.(*widget.Entry); ok {
		return e.Text
	}
	if l, ok := w.(*widget.Label); ok {
		return l.Text
	}
	if b, ok := w.(*widget.Button); ok {
		return b.Text
	}
	return ""
}

func (u *UI) Set(name string, value string) {
	w, ok := u.widgets[name]
	if !ok {
		return
	}
	fyne.Do(func() {
		if e, ok := w.(*widget.Entry); ok {
			e.SetText(value)
		} else if l, ok := w.(*widget.Label); ok {
			l.SetText(value)
		} else if b, ok := w.(*widget.Button); ok {
			b.SetText(value)
		}
	})
}

func (u *UI) Space() {
	u.content = append(u.content, widget.NewLabel(""))
}

func (u *UI) Row(items []fyne.CanvasObject) {
	row := container.NewHBox(items...)
	u.content = append(u.content, row)
}

func (u *UI) Refresh() {
	if u.window != nil {
		fyne.Do(func() {
			u.window.Content().Refresh()
		})
	}
}

func (u *UI) Run() {
	box := container.NewVBox(u.content...)
	u.window.SetContent(box)
	u.window.ShowAndRun()
}

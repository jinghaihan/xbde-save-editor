package app

import (
	"math"

	"github.com/egoist/mygo/ui"
)

// Use FETH's container spacing and standard MyGo widget sizes.
var editorAppearance = struct {
	CardRadius, CardPadding float32
	ControlGap, SectionGap  float32
}{20, 20, 8, 12}

func makeEditorTheme(dark bool) *ui.Theme {
	t := ui.LightTheme()
	if dark {
		t = ui.DarkTheme()
		t.Accent = ui.Hex("#f1a9ac")
		t.AccentHover = ui.Hex("#fac4c7")
		t.AccentPressed = ui.Hex("#db8e94")
		t.AccentText = ui.Hex("#38191d")
		t.Selection = ui.RGBA(241, 169, 172, 0.30)
		t.Focus = ui.RGBA(241, 169, 172, 0.75)
	} else {
		t.Background = ui.Hex("#e7e7e7")
		t.Surface = ui.Hex("#f9f9f9")
		t.SurfaceHover = ui.Hex("#ececec")
		t.SurfacePressed = ui.Hex("#dfdfdf")
		t.Border = ui.Hex("#e2e2e2")
		t.Text = ui.Hex("#222222")
		t.TextMuted = ui.Hex("#7f7f7f")
		t.Accent = ui.Hex("#af3b48")
		t.AccentHover = ui.Hex("#98303d")
		t.AccentPressed = ui.Hex("#802833")
		t.AccentText = ui.Hex("#fffaf1")
		t.Selection = ui.RGBA(175, 59, 72, 0.22)
		t.Focus = ui.RGBA(175, 59, 72, 0.65)
	}
	t.Radius = 8
	t.FontSize = 13
	return t
}

var lightEditorTheme = makeEditorTheme(false)
var darkEditorTheme = makeEditorTheme(true)

func (a *App) currentTheme() *ui.Theme {
	if a.dark {
		return darkEditorTheme
	}
	return lightEditorTheme
}
func cardColor(c *ui.Context) ui.Color {
	if c.Theme().Dark {
		return ui.Hex("#222225")
	}
	return ui.Hex("#f2f2f2")
}
func themedList(c *ui.Context, state *ui.ListState, count int, row func(int)) ui.Element {
	t := *c.Theme()
	if t.Dark {
		t.Accent = ui.Hex("#64333b")
		t.AccentText = ui.Hex("#fcebee")
	} else {
		t.Accent = ui.Hex("#f0d5d9")
		t.AccentText = ui.Hex("#792936")
	}
	var list ui.Element
	ui.Themed(c, &t).Grow(1).Basis(0).MinHeight(0).Children(func() { list = ui.List(c, state, count, row).Grow(1).MinHeight(0).Background(cardColor(c)).Gap(2) })
	return list
}

type formLayout struct {
	fields []ui.Element
	width  float32
}

func (a *App) editorForm(c *ui.Context, fields func()) ui.Element {
	previous := a.form
	form := &formLayout{}
	a.form = form
	defer func() { a.form = previous }()
	column := ui.Column(c).Gap(editorAppearance.ControlGap).MinWidth(0).Children(fields)
	for _, field := range form.fields {
		field.ColumnTracks(ui.Fixed(float32(math.Ceil(float64(form.width)))), ui.Fr(1))
	}
	return column
}

func (a *App) editorField(c *ui.Context, label string, control func()) ui.Element {
	if a.form == nil {
		return ui.Field(c, label, control).Gap(editorAppearance.ControlGap).MinWidth(0)
	}
	form := a.form
	field := ui.Field(c, label, control).Grid().AlignItems(ui.Center).
		Gap(editorAppearance.ControlGap).MinWidth(0)
	width, _ := c.MeasureText(0, ui.Span{Text: label, Font: c.Theme().Font, Size: c.Theme().FontSize, Weight: 500})
	form.width = max(form.width, width)
	form.fields = append(form.fields, field)
	return field
}

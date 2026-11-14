package app

import (
	"embed"
	"encoding/json"
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/i18n"
	_ "github.com/egoist/mygo/locales/ja"
	_ "github.com/egoist/mygo/locales/zh-Hans"
	_ "github.com/egoist/mygo/locales/zh-Hant"
	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/xbde-save-editor/internal/save"
	"os"
	"path/filepath"
	"strings"
)

//go:embed locales/*.json
var localeFiles embed.FS
var Version = "0.0.0"
var pages = []string{"Main", "Characters", "Items", "Affinity", "Achievements", "Collectopaedia"}
var languages = []struct{ Code, Name string }{{"en", "English"}, {"zh-Hans", "简体中文"}, {"zh-Hant", "繁體中文"}, {"ja", "日本語"}, {"ko", "한국어"}, {"de", "Deutsch"}, {"fr", "Français"}, {"es", "Español"}, {"it", "Italiano"}}

type edit func(*save.Document)
type row struct{ ID, Name, Value string }
type tableState struct {
	List     ui.ListState
	Query    string
	Selected int
	Rows     []row
}
type App struct {
	invalid                map[string]string
	Save                   *save.Document
	Session                *save.Session
	Path, Language, Status string
	Page                   int
	win                    *mygo.Window
	locale                 map[string]map[string]string
	tabs                   map[string]*int
	tables                 map[string]*tableState
	numbers                map[string]*float64
	texts, choices         map[string]*string
	flags                  map[string]*bool
	drafts                 map[string]any
	pending                map[string]edit
	pendingOrder           []string
	focusedInput           string
	revision               int
	dark                   bool
	preferencesPath        string
	languageChosen         bool
	confirm                bool
	confirmLabel           string
	confirmAction          func()
}

func New() *App {
	a := &App{invalid: map[string]string{}, Language: "en", locale: map[string]map[string]string{}, tabs: map[string]*int{}, tables: map[string]*tableState{}, numbers: map[string]*float64{}, texts: map[string]*string{}, choices: map[string]*string{}, flags: map[string]*bool{}, drafts: map[string]any{}, pending: map[string]edit{}}
	for _, l := range languages {
		b, e := localeFiles.ReadFile("locales/" + l.Code + ".json")
		if e != nil {
			panic(e)
		}
		var m map[string]string
		if e = json.Unmarshal(b, &m); e != nil {
			panic(e)
		}
		a.locale[l.Code] = m
	}
	return a
}
func (a *App) T(key string) string {
	if v := a.locale[a.Language][key]; v != "" {
		return v
	}
	if v := a.locale["en"][key]; v != "" {
		return v
	}
	return key
}
func (a *App) SetLanguage(code string) error {
	if a.locale[code] == nil {
		return fmt.Errorf("unsupported language: %s", code)
	}
	a.Language = code
	a.revision++
	if a.win != nil {
		mygo.App.SetLocale(code)
		a.menu()
		a.win.Invalidate()
	}
	return nil
}
func (a *App) Load(path string) error {
	s, e := save.OpenSession(path)
	if e != nil {
		return e
	}
	a.Session = s
	a.Save = s.Document
	a.Path = s.Path
	a.Status = ""
	a.invalid = map[string]string{}
	a.pending = map[string]edit{}
	a.pendingOrder = nil
	a.focusedInput = ""
	a.numbers = map[string]*float64{}
	a.choices = map[string]*string{}
	a.flags = map[string]*bool{}
	a.drafts = map[string]any{}
	a.tables = map[string]*tableState{}
	a.revision++
	if a.win != nil {
		a.menu()
	}
	return nil
}
func (a *App) stage(key string, fn edit) {
	if a.pending[key] == nil {
		a.pendingOrder = append(a.pendingOrder, key)
	}
	a.pending[key] = fn
}
func (a *App) dirty(key string) bool {
	if a.invalid[key] != "" {
		return true
	}
	for k := range a.pending {
		if key == k || strings.HasPrefix(key, k+"/") {
			return true
		}
	}
	return false
}
func (a *App) ApplyPending(prefix string) error {
	if a.Save == nil {
		return fmt.Errorf("No save loaded")
	}
	for k, message := range a.invalid {
		if prefix == "" || k == prefix || strings.HasPrefix(k, prefix+"/") {
			a.Status = message
			return fmt.Errorf("%s", message)
		}
	}
	keys := []string{}
	err := a.Save.Edit(func(s *save.Document) {
		for _, k := range a.pendingOrder {
			if prefix == "" || k == prefix || strings.HasPrefix(k, prefix+"/") {
				if fn := a.pending[k]; fn != nil {
					fn(s)
					keys = append(keys, k)
				}
			}
		}
	})
	if err != nil {
		a.Status = err.Error()
		return err
	}
	for _, k := range keys {
		delete(a.pending, k)
	}
	order := a.pendingOrder[:0]
	for _, k := range a.pendingOrder {
		if a.pending[k] != nil {
			order = append(order, k)
		}
	}
	a.pendingOrder = order
	a.Status = ""
	a.pruneDrafts()
	return nil
}
func (a *App) perform(fn edit) bool {
	if a.Save == nil {
		return false
	}
	for _, message := range a.invalid {
		a.Status = message
		return false
	}
	err := a.Save.Edit(func(s *save.Document) {
		for _, k := range a.pendingOrder {
			if f := a.pending[k]; f != nil {
				f(s)
			}
		}
		fn(s)
	})
	if err != nil {
		a.Status = err.Error()
		return false
	}
	a.invalid = map[string]string{}
	a.pending = map[string]edit{}
	a.pendingOrder = nil
	a.drafts = map[string]any{}
	a.Status = ""
	return true
}
func (a *App) SaveTo(path string) error {
	if e := a.ApplyPending(""); e != nil {
		return e
	}
	return a.Session.Save(path)
}
func (a *App) ask(key string, fn func()) {
	a.confirm = true
	a.confirmLabel = a.T(key)
	a.confirmAction = fn
}
func (a *App) post(fn func()) {
	if a.win != nil {
		a.win.Update(fn)
	} else {
		fn()
	}
}
func (a *App) openPath(path string) {
	if a.Session != nil && (a.Session.Changed() || len(a.pending) > 0) {
		a.ask("DiscardQuestion", func() { a.loadPath(path) })
		return
	}
	a.loadPath(path)
}
func (a *App) loadPath(path string) {
	go func() {
		s, e := save.OpenSession(path)
		a.post(func() {
			if e != nil {
				a.Status = e.Error()
				return
			}
			a.Session = s
			a.Save = s.Document
			a.Path = s.Path
			a.invalid = map[string]string{}
			a.pending = map[string]edit{}
			a.pendingOrder = nil
			a.focusedInput = ""
			a.numbers = map[string]*float64{}
			a.choices = map[string]*string{}
			a.flags = map[string]*bool{}
			a.drafts = map[string]any{}
			a.tables = map[string]*tableState{}
			a.revision++
			a.Status = ""
			a.menu()
		})
	}()
}
func (a *App) open() {
	go func() {
		paths, e := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: a.win, Title: a.T("OpenSave")})
		a.post(func() {
			if e != nil {
				a.Status = e.Error()
			} else if len(paths) > 0 {
				a.openPath(paths[0])
			}
		})
	}()
}
func (a *App) save(as bool) {
	if a.Save == nil {
		return
	}
	if !as {
		if e := a.SaveTo(a.Path); e != nil {
			a.Status = e.Error()
		} else {
			a.Status = a.T("Saved") + " " + a.Path
		}
		return
	}
	go func() {
		path, e := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: a.win, Title: a.T("SaveAs"), DefaultPath: filepath.Join(filepath.Dir(a.Path), "edited-"+filepath.Base(a.Path))})
		a.post(func() {
			if e != nil {
				a.Status = e.Error()
			} else if path != "" {
				if e = a.SaveTo(path); e != nil {
					a.Status = e.Error()
				} else {
					a.Status = a.T("Saved") + " " + path
				}
			}
		})
	}()
}
func (a *App) menu() {
	langs := []*mygo.MenuItem{}
	for _, l := range languages {
		code := l.Code
		langs = append(langs, &mygo.MenuItem{Label: l.Name, Click: func(*mygo.MenuItem, *mygo.Window) {
			a.post(func() {
				a.SetLanguage(code)
				a.languageChosen = true
				if a.preferencesPath != "" {
					os.MkdirAll(filepath.Dir(a.preferencesPath), 0700)
					if e := os.WriteFile(a.preferencesPath, []byte(code), 0600); e != nil {
						a.Status = e.Error()
					}
				}
			})
		}})
	}
	file := &mygo.MenuItem{Label: a.T("File"), Submenu: []*mygo.MenuItem{
		{Label: a.T("OpenSave"), Accelerator: "CmdOrCtrl+O", Click: func(*mygo.MenuItem, *mygo.Window) { a.open() }},
		{Label: a.T("Save"), Accelerator: "CmdOrCtrl+S", Disabled: a.Save == nil, Click: func(*mygo.MenuItem, *mygo.Window) { a.save(false) }},
		{Label: a.T("SaveAs"), Accelerator: "CmdOrCtrl+Shift+S", Disabled: a.Save == nil, Click: func(*mygo.MenuItem, *mygo.Window) { a.save(true) }},
		{Label: a.T("Exit"), Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
	}}
	about := &mygo.MenuItem{Label: a.T("About"), Click: func(*mygo.MenuItem, *mygo.Window) {
		go mygo.Dialog.Message(mygo.MessageOptions{Parent: a.win, Message: "XBDE Save Editor " + Version, Detail: "Save references: damysteryman/XCDESave. Data: Xenoblade DE data tables. UI: MyGo.\nhttps://github.com/jinghaihan/xbde-save-editor", Buttons: []string{"OK"}})
	}}
	mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{{Role: mygo.RoleAppMenu}, file, {Label: a.T("Language"), Submenu: langs}, {Role: mygo.RoleEditMenu}, {Label: a.T("Help"), Submenu: []*mygo.MenuItem{about}}}))
}
func Run(path string) error {
	a := New()
	if dir, e := os.UserConfigDir(); e == nil {
		a.preferencesPath = filepath.Join(dir, "XbdeEditor", "language.txt")
		if b, e := os.ReadFile(a.preferencesPath); e == nil {
			a.languageChosen = a.SetLanguage(strings.TrimSpace(string(b))) == nil
		}
	}
	if l := os.Getenv("XBDE_EDITOR_LANGUAGE"); l != "" {
		a.languageChosen = a.SetLanguage(l) == nil
	}
	mygo.App.WhenReady(func() {
		if !a.languageChosen {
			tags := []string{}
			for _, l := range languages {
				tags = append(tags, l.Code)
			}
			a.SetLanguage(i18n.Match(mygo.App.PreferredLanguages(), tags))
		}
		mygo.Theme.SetSource(mygo.ThemeSystem)
		a.dark = mygo.Theme.IsDark()
		a.win = mygo.NewWindow(mygo.WindowOptions{Title: "XBDE Save Editor", Width: 1120, Height: 780, MinWidth: 860, MinHeight: 600, StateKey: "main", Content: ui.View(a.View)})
		a.menu()
		mygo.Theme.OnUpdated(func() { a.post(func() { a.dark = mygo.Theme.IsDark() }) })
		if path != "" {
			a.openPath(path)
		}
	})
	return mygo.App.Run()
}

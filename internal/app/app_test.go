package app

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/xbde-save-editor/internal/save"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func fixture(t *testing.T) *App {
	t.Helper()
	root := os.Getenv("XBDE_SAVE_DIRECTORY")
	if root == "" {
		root = t.TempDir()
		raw := make([]byte, save.FileSize)
		binary.LittleEndian.PutUint32(raw, 7)
		raw[save.PartyOffset+24] = 2
		binary.LittleEndian.PutUint16(raw[save.PartyOffset:], 1)
		binary.LittleEndian.PutUint16(raw[save.PartyOffset+2:], 2)
		binary.LittleEndian.PutUint32(raw[save.CharacterOffset:], 20)
		binary.LittleEndian.PutUint32(raw[save.CharacterOffset+save.CharacterSize:], 20)
		raw[save.ArtsOffset] = 1
		if e := os.WriteFile(filepath.Join(root, "bfsgame00.sav"), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	a := New()
	if e := a.Load(filepath.Join(root, "bfsgame00.sav")); e != nil {
		t.Fatal(e)
	}
	return a
}
func modifier() ui.Modifiers {
	if runtime.GOOS == "darwin" {
		return ui.Cmd
	}
	return ui.Ctrl
}
func snapshot(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	dir := os.Getenv("XBDE_AUDIT_DIR")
	if dir == "" {
		return
	}
	if e := os.MkdirAll(dir, 0755); e != nil {
		t.Fatal(e)
	}
	f, e := os.Create(filepath.Join(dir, name+".png"))
	if e != nil {
		t.Fatal(e)
	}
	defer f.Close()
	if e = png.Encode(f, tt.Image()); e != nil {
		t.Fatal(e)
	}
}
func bounds(t *testing.T, tt *ui.Tester, width int) {
	t.Helper()
	nodes := tt.AccessAll()
	for _, n := range nodes {
		if n.Offscreen || n.Bounds.W <= 0 || n.Bounds.H <= 0 {
			continue
		}
		if n.Role == ui.RoleTextField || n.Role == ui.RoleButton || n.Role == ui.RoleComboBox {
			if n.Bounds.X < -0.5 || n.Bounds.X+n.Bounds.W > float32(width)+.5 {
				t.Errorf("%s overflows: %+v", n.Name, n.Bounds)
			}
		}
	}
	for _, search := range nodes {
		if !search.Search {
			continue
		}
		for _, list := range nodes {
			if list.Role != ui.RoleList {
				continue
			}
			gap := list.Bounds.Y - search.Bounds.Y - search.Bounds.H
			if gap >= 0 && gap < 20 && math.Abs(float64(search.Bounds.X-list.Bounds.X)) < 20 {
				if search.Bounds.X != list.Bounds.X || search.Bounds.W != list.Bounds.W {
					t.Errorf("search/list misaligned: %+v / %+v", search.Bounds, list.Bounds)
				}
			}
		}
	}
}
func TestAllPagesAndThemes(t *testing.T) {
	for _, dark := range []bool{false, true} {
		for _, loaded := range []bool{false, true} {
			for _, size := range [][2]int{{1120, 780}, {860, 600}} {
				a := New()
				if loaded {
					a = fixture(t)
				}
				a.dark = dark
				tt := ui.NewTesterWith(a.View, size[0], size[1], ui.TesterOptions{Dark: dark})
				for p := range pages {
					a.Page = p
					subcount := 1
					key := ""
					switch p {
					case 1:
						key = "characterTabs"
						subcount = 5
					case 2:
						key = "itemTabs"
						subcount = 7
					case 3:
						key = "affinityTabs"
						subcount = 2
					}
					for sub := 0; sub < subcount; sub++ {
						if key != "" {
							*a.tab(key) = sub
						}
						a.revision++
						tt.Frame()
						bounds(t, tt, size[0])
						if size[0] == 1120 && loaded {
							snapshot(t, tt, fmt.Sprintf("%s-%d-dark%v", pages[p], sub, dark))
						}
						if _, ok := tt.AccessByRole(ui.RoleButton, a.T("ApplyChanges")); ok {
							t.Fatal("redundant Apply button")
						}
					}
				}
			}
		}
	}
}
func editNumber(t *testing.T, tt *ui.Tester, label, text string) {
	t.Helper()
	group, ok := tt.AccessByRole(ui.RoleGroup, label)
	if !ok {
		t.Fatalf("missing numeric input %s", label)
	}
	found := false
	for _, n := range tt.AccessAll() {
		if n.Role == ui.RoleTextField && n.Bounds.X >= group.Bounds.X && n.Bounds.Y >= group.Bounds.Y && n.Bounds.X+n.Bounds.W <= group.Bounds.X+group.Bounds.W && n.Bounds.Y+n.Bounds.H <= group.Bounds.Y+group.Bounds.H {
			tt.ClickAt(n.Bounds.X+n.Bounds.W/2, n.Bounds.Y+n.Bounds.H/2)
			found = true
			break
		}
	}
	if !found {
		t.Fatal("numeric input not accessible")
	}
	tt.Key(modifier(), ui.KeyA)
	tt.Type(text)
}
func TestBlurSaveAndBulkAction(t *testing.T) {
	a := fixture(t)
	original, e := os.ReadFile(a.Path)
	if e != nil {
		t.Fatal(e)
	}
	tt := ui.NewTester(a.View, 1120, 780)
	editNumber(t, tt, a.T("Money"), "123456")
	if a.Save.Money() == 123456 {
		t.Fatal("focused draft applied before blur")
	}
	editNumber(t, tt, a.T("Noponstones"), "4567")
	if a.Save.Money() != 123456 {
		t.Fatalf("blur did not commit: %s", a.Status)
	}
	editNumber(t, tt, a.T("Noponstones"), "4567")
	path := filepath.Join(t.TempDir(), "copy.sav")
	if e = a.SaveTo(path); e != nil {
		t.Fatal(e)
	}
	d, e := save.Open(path)
	if e != nil || d.Money() != 123456 || d.Noponstones() != 4567 {
		t.Fatal("focused Save did not flush", e)
	}
	raw, _ := os.ReadFile(a.Path)
	if !bytes.Equal(raw, original) {
		t.Fatal("editing wrote original")
	}
	a.Page = 1
	a.revision++
	tt.Frame()
	editNumber(t, tt, a.T("AP"), "12")
	if e = tt.Click(a.T("MaxAP")); e != nil {
		t.Fatal(e)
	}
	if a.Save.Characters()[0].AP() != 99999999 {
		t.Fatal("stale draft overwrote bulk action", a.Status)
	}
}
func TestNineLanguagesAndCampaigns(t *testing.T) {
	a := fixture(t)
	root := os.Getenv("XBDE_SAVE_DIRECTORY")
	if root == "" {
		root = t.TempDir()
		raw := a.Save.Bytes()
		if e := os.WriteFile(filepath.Join(root, "bfsgame00.sav"), raw, 0600); e != nil {
			t.Fatal(e)
		}
		binary.LittleEndian.PutUint16(raw[save.PartyOffset+2:], 14)
		binary.LittleEndian.PutUint32(raw[save.CharacterOffset+13*save.CharacterSize:], 58)
		if e := os.WriteFile(filepath.Join(root, "bfsmeria00.sav"), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	for _, file := range []string{"bfsgame00.sav", "bfsmeria00.sav"} {
		if e := a.Load(filepath.Join(root, file)); e != nil {
			t.Fatal(e)
		}
		for _, lang := range languages {
			a.SetLanguage(lang.Code)
			for page := range pages {
				a.Page = page
				a.revision++
				tt := ui.NewTester(a.View, 1120, 780)
				bounds(t, tt, 1120)
			}
		}
	}
}

func TestInventoryControlsCommitToMemory(t *testing.T) {
	a := fixture(t)
	if e := a.Save.Edit(func(s *save.Document) { s.AddItem(1852, 1) }); e != nil {
		// The user's stack may already be at its cap. Choose the existing stack.
		if len(a.Save.Inventory(10)) == 0 {
			t.Fatal(e)
		}
	}
	a.Page = 2
	*a.tab("itemTabs") = 1
	before := a.Save.Bytes()
	source, _ := os.ReadFile(a.Path)
	tt := ui.NewTester(a.View, 1120, 780)
	editNumber(t, tt, a.T("Quantity"), "7")
	if e := tt.Click(a.T("Favorite")); e != nil {
		t.Fatal(e)
	}
	item := a.Save.Inventory(10)[0]
	if item.Quantity() != 7 {
		t.Fatalf("quantity did not commit: %d (%s)", item.Quantity(), a.Status)
	}
	if bytes.Equal(a.Save.Bytes(), before) {
		t.Fatal("checkbox/quantity made no edit")
	}
	path := filepath.Join(t.TempDir(), "inventory.sav")
	if e := a.SaveTo(path); e != nil {
		t.Fatal(e)
	}
	reloaded, e := save.Open(path)
	if e != nil || reloaded.Inventory(10)[0].Quantity() != 7 {
		t.Fatal("inventory write", e)
	}
	original, _ := os.ReadFile(a.Path)
	if !bytes.Equal(source, original) {
		t.Fatal("inventory UI changed source")
	}
}
func TestInvalidFocusedValuePreventsSave(t *testing.T) {
	a := fixture(t)
	before := a.Save.Bytes()
	tt := ui.NewTester(a.View, 1120, 780)
	editNumber(t, tt, a.T("Money"), "invalid")
	path := filepath.Join(t.TempDir(), "invalid.sav")
	if e := a.SaveTo(path); e == nil {
		t.Fatal("invalid focused input accepted")
	}
	if _, e := os.Stat(path); !os.IsNotExist(e) {
		t.Fatal("failed save wrote a file")
	}
	if !bytes.Equal(before, a.Save.Bytes()) {
		t.Fatal("invalid draft changed memory")
	}
}

func TestEmptyInventoryKeepsCreationControls(t *testing.T) {
	a := fixture(t)
	if err := a.Save.Edit(func(s *save.Document) {
		for _, item := range s.Inventory(13) {
			item.Delete()
		}
	}); err != nil {
		t.Fatal(err)
	}
	source, err := os.ReadFile(a.Path)
	if err != nil {
		t.Fatal(err)
	}
	a.Page = 2
	*a.tab("itemTabs") = 3
	tt := ui.NewTester(a.View, 1120, 780)
	add, ok := tt.AccessByRole(ui.RoleButton, a.T("Add"))
	if !ok || add.Disabled {
		t.Fatal("empty inventory cannot create its first item")
	}
	remove, ok := tt.AccessByRole(ui.RoleButton, a.T("Delete"))
	if !ok || !remove.Disabled {
		t.Fatal("empty inventory must retain disabled item controls")
	}
	if err := tt.Click(a.T("Add")); err != nil {
		t.Fatal(err)
	}
	if len(a.Save.Inventory(13)) != 1 {
		t.Fatalf("Add did not create one manual: %s", a.Status)
	}
	if a.confirm {
		t.Fatal("adding an item opened a redundant confirmation dialog")
	}
	original, err := os.ReadFile(a.Path)
	if err != nil || !bytes.Equal(source, original) {
		t.Fatal("creating an item wrote to the source file")
	}
}

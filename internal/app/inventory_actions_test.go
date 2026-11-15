package app

import (
	"bytes"
	"os"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/xbde-save-editor/internal/save"
)

func TestInventoryListDialogs(t *testing.T) {
	for _, mode := range []struct {
		name, key string
		tab       int
		count     func(*save.Document) int
	}{
		{"manual", "inventory/13", 3, func(s *save.Document) int { return len(s.Inventory(13)) }},
		{"weapon", "equipmentInventory/0", 5, func(s *save.Document) int { return len(s.EquipmentInventory(0)) }},
		{"armor", "equipmentInventory/2", 6, func(s *save.Document) int { return len(s.EquipmentInventory(2)) }},
		{"gem", "gems", 0, func(s *save.Document) int { return len(s.Gems()) }},
	} {
		t.Run(mode.name, func(t *testing.T) {
			a := fixture(t)
			if mode.name == "manual" {
				if err := a.Save.Edit(func(s *save.Document) {
					for _, x := range s.Inventory(13) {
						x.Delete()
					}
				}); err != nil {
					t.Fatal(err)
				}
			}
			source, _ := os.ReadFile(a.Path)
			a.Page = 2
			if mode.name == "armor" {
				*a.filter("armorSlot", "1") = "2"
			}
			*a.tab("itemTabs") = mode.tab
			tt := ui.NewTester(a.View, 860, 600)
			a.table(mode.key).Query = "__no_match__"
			tt.Frame()
			baseline := a.Save.Bytes()
			count := mode.count(a.Save)
			click := func(key string) {
				t.Helper()
				button, ok := tt.AccessByRole(ui.RoleButton, a.T(key))
				if !ok || button.Disabled {
					t.Fatalf("button %s is unavailable", key)
				}
				tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
			}
			click("Add")
			if a.inventoryDialog == nil || !bytes.Equal(baseline, a.Save.Bytes()) {
				t.Fatal("opening parameters edited the document")
			}
			bounds(t, tt, 860)
			snapshot(t, tt, "Dialog-add-"+mode.name)
			click("Cancel")
			if a.inventoryDialog != nil || !bytes.Equal(baseline, a.Save.Bytes()) {
				t.Fatal("cancel changed inventory")
			}
			click("Add")
			click("Add")
			if a.inventoryDialog != nil || mode.count(a.Save) != count+1 {
				t.Fatalf("Add did not create one record: %s", a.Status)
			}
			if mode.name == "armor" && *a.filter("armorSlot", "1") != "2" {
				t.Fatal("inventory action reset the armor category")
			}
			selected := a.selectedID(mode.key)
			if selected == "" || a.table(mode.key).Query != "" {
				t.Fatal("created record is not selected and visible in inventory")
			}
			list, ok := tt.AccessByRole(ui.RoleList, "")
			if !ok {
				t.Fatal("missing inventory list")
			}
			for _, key := range []string{"Add", "Delete"} {
				button, ok := tt.AccessByRole(ui.RoleButton, a.T(key))
				if !ok || button.Bounds.X < list.Bounds.X || button.Bounds.X+button.Bounds.W > list.Bounds.X+list.Bounds.W {
					t.Fatalf("%s is outside the inventory sidebar", key)
				}
			}
			click("Delete")
			if mode.count(a.Save) != count {
				t.Fatalf("Delete did not remove the selected record: %s", a.Status)
			}
			if a.selectedID(mode.key) == selected {
				t.Fatal("selection still refers to deleted record")
			}
			after, _ := os.ReadFile(a.Path)
			if !bytes.Equal(source, after) {
				t.Fatal("inventory action wrote the source file")
			}
		})
	}
}

func TestFillMissingUsesSidebarAndDoesNotEquip(t *testing.T) {
	a := fixture(t)
	a.Page = 2
	*a.tab("itemTabs") = 5
	source, _ := os.ReadFile(a.Path)
	tt := ui.NewTester(a.View, 860, 600)
	character := a.Save.Characters()[0]
	equipped := character.Equipment(0).Index()
	owned := map[int]bool{}
	for _, x := range a.Save.EquipmentInventory(0) {
		owned[x.ItemID()] = true
	}
	expected := 0
	for _, def := range a.Save.CreatableEquipment(0) {
		for _, id := range def.Characters {
			if id == character.ID && !owned[def.Id] {
				expected++
				break
			}
		}
	}
	count := len(a.Save.EquipmentInventory(0))
	baseline := a.Save.Bytes()
	click := func(key string) {
		t.Helper()
		button, ok := tt.AccessByRole(ui.RoleButton, a.T(key))
		if !ok || button.Disabled {
			t.Fatalf("button %s is unavailable", key)
		}
		tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
	}
	click("FillMissingEquipment")
	if a.inventoryDialog == nil || !bytes.Equal(baseline, a.Save.Bytes()) {
		t.Fatal("opening bulk parameters changed inventory")
	}
	bounds(t, tt, 860)
	snapshot(t, tt, "Dialog-fill-equipment")
	click("Cancel")
	if !bytes.Equal(baseline, a.Save.Bytes()) {
		t.Fatal("cancelling bulk parameters changed inventory")
	}
	click("FillMissingEquipment")
	click("FillMissingEquipment")
	if a.inventoryDialog != nil || len(a.Save.EquipmentInventory(0)) != count+expected {
		t.Fatalf("bulk inventory count is wrong: before=%d after=%d expected=%d dialog=%+v status=%s", count, len(a.Save.EquipmentInventory(0)), expected, a.inventoryDialog, a.Status)
	}
	if a.Save.Character(character.ID).Equipment(0).Index() != equipped {
		t.Fatal("bulk inventory creation changed character equipment")
	}
	completed := a.Save.Bytes()
	click("FillMissingEquipment")
	click("FillMissingEquipment")
	if !bytes.Equal(completed, a.Save.Bytes()) {
		t.Fatal("repeated Fill Missing created duplicates")
	}
	after, _ := os.ReadFile(a.Path)
	if !bytes.Equal(source, after) {
		t.Fatal("bulk inventory action wrote the source file")
	}
}

func TestInventoryCommandsRemainVisibleWhenUnavailable(t *testing.T) {
	for _, sub := range []int{0, 3, 4, 5, 6} {
		a := New()
		a.Page = 2
		*a.tab("itemTabs") = sub
		tt := ui.NewTester(a.View, 860, 600)
		for _, label := range []string{"Add", "Delete"} {
			command, ok := tt.AccessByRole(ui.RoleButton, a.T(label))
			if !ok || !command.Disabled {
				t.Fatalf("tab %d must retain its disabled %s command", sub, label)
			}
		}
	}
}

func TestInventoryDialogsInEveryLanguage(t *testing.T) {
	for _, language := range languages {
		for _, spec := range []struct {
			mode, key     string
			tab, category int
		}{{"item", "inventory/13", 3, 13}, {"equipment", "equipmentInventory/0", 5, 0}, {"gem", "gems", 0, 0}, {"fill", "equipmentInventory/0", 5, 0}} {
			a := fixture(t)
			a.SetLanguage(language.Code)
			a.Page = 2
			*a.tab("itemTabs") = spec.tab
			tt := ui.NewTester(a.View, 860, 600)
			command := "Add"
			if spec.mode == "fill" {
				command = "FillMissingEquipment"
			}
			button, ok := tt.AccessByRole(ui.RoleButton, a.T(command))
			if !ok || button.Disabled {
				t.Fatalf("missing %s in %s", command, language.Code)
			}
			list, ok := tt.AccessByRole(ui.RoleList, "")
			if !ok || button.Bounds.X+button.Bounds.W > list.Bounds.X+list.Bounds.W+.5 {
				t.Fatalf("%s overflows the sidebar in %s", command, language.Code)
			}
			tt.ClickAt(button.Bounds.X+button.Bounds.W/2, button.Bounds.Y+button.Bounds.H/2)
			bounds(t, tt, 860)
			cancel, ok := tt.AccessByRole(ui.RoleButton, a.T("Cancel"))
			if !ok || cancel.Disabled {
				t.Fatalf("missing Cancel in %s", language.Code)
			}
			tt.Key(0, ui.KeyEscape)
			if a.inventoryDialog != nil {
				t.Fatalf("Escape did not dismiss %s in %s", spec.mode, language.Code)
			}
		}
	}
}

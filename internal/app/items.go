package app

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/xbde-save-editor/internal/save"
	"strings"
)

func (a *App) namesOrNone(names []string) string {
	if len(names) == 0 {
		return a.T("None")
	}
	return strings.Join(names, ", ")
}

func (a *App) itemsPage(c *ui.Context) {
	t := a.tabsView(c, "itemTabs", "Gems", "Collectables", "Materials", "ArtManuals", "KeyItems", "Weapons", "Armor")
	switch {
	case t == 0:
		a.gemsPage(c)
	case t <= 4:
		kind := []int{0, 10, 11, 13, 12}[t]
		a.inventoryPage(c, kind)
	default:
		slot := 0
		if t == 6 {
			selected := a.filter("armorSlot", "1")
			slot = parseID(*selected)
		}
		a.equipmentInventoryPage(c, slot)
	}
}
func (a *App) inventoryPage(c *ui.Context, kind int) {
	rows := []row{}
	if a.Save != nil {
		for _, x := range a.Save.Inventory(kind) {
			rows = append(rows, row{rid(x.Index), a.itemName(x.ItemID(), x.Name()), rid(x.Quantity())})
		}
	}
	key := "inventory/" + rid(kind)
	a.split(c, key, save.KindNames[kind], rows, listHeader{}, func() {
		id := a.selectedID(key)
		editable := a.Save != nil && id != "" && a.Save.Item(kind, parseID(id)).CanEdit()
		canCreate := a.Save != nil && a.Save.Campaign() != "Unknown" && kind != 12
		ui.Row(c).Gap(8).Children(func() {
			a.inventoryCommand(c, "Add", canCreate, func() { a.openInventoryDialog("item", key, kind) })
			a.action(c, "Delete", editable, func(s *save.Document) { s.Item(kind, parseID(id)).Delete() })
		})
		a.action(c, "MaxQuantities", canCreate && len(rows) > 0, func(s *save.Document) { s.MaxItems(kind) })
	}, func(id string) {
		a.heading(c, "ItemEditor", "")
		quantity, favorite, editable := 0, false, false
		index := parseID(id)
		if a.Save != nil && id != "" {
			x := a.Save.Item(kind, index)
			quantity, favorite, editable = x.Quantity(), x.Favorite(), x.CanEdit()
		}
		a.disabled(c, !editable, func() {
			a.numberEdit(c, "Quantity", key+"/"+id+"/quantity", quantity, 1, 99, func(s *save.Document, v int) { s.Item(kind, index).SetQuantity(v) })
			v := a.flag(key+"/"+id+"/favorite", favorite)
			if ui.Checkbox(c, v, a.T("Favorite")).Changed() {
				n := *v
				a.stage(key+"/"+id+"/favorite", func(s *save.Document) { s.Item(kind, index).SetFavorite(n) })
			}
		})
	})
}
func (a *App) equipmentInventoryPage(c *ui.Context, slot int) {
	rows := []row{}
	if a.Save != nil {
		for _, x := range a.Save.EquipmentInventory(slot) {
			rows = append(rows, row{rid(x.Index), a.itemName(x.ItemID(), x.Name()), ""})
		}
	}
	key := "equipmentInventory/" + rid(slot)
	header := listHeader{}
	if slot > 0 {
		header.controls = func() {
			options := []option{}
			for i := 1; i <= 5; i++ {
				options = append(options, option{rid(i), a.T(save.SlotNames[i])})
			}
			a.combo(c, "Armor", "armorSlot", a.filter("armorSlot", "1"), options)
		}
	}
	a.split(c, key, save.SlotNames[slot], rows, header, func() {
		id := a.selectedID(key)
		canDelete := a.Save != nil && id != "" && a.Save.OwnedEquipment(slot, parseID(id)).CanDelete()
		canCreate := a.Save != nil && len(a.Save.CreatableEquipment(slot)) > 0
		ui.Row(c).Gap(8).Children(func() {
			a.inventoryCommand(c, "Add", canCreate, func() { a.openInventoryDialog("equipment", key, slot) })
			a.action(c, "Delete", canDelete, func(s *save.Document) { s.OwnedEquipment(slot, parseID(id)).Delete() })
		})
		a.inventoryCommand(c, "FillMissingEquipment", canCreate, func() { a.openInventoryDialog("fill", key, slot) })
	}, func(id string) {
		a.heading(c, "Equipment", "")
		var sockets any = ""
		names, fixed := []string{}, []string{}
		favorite, editable := false, false
		index := parseID(id)
		if a.Save != nil && id != "" {
			x := a.Save.OwnedEquipment(slot, index)
			sockets, favorite, editable = x.Sockets(), x.Favorite(), x.CanEdit()
			for _, cid := range x.EquippedBy() {
				names = append(names, save.CharacterName(cid, a.Language))
			}
			for _, gid := range x.FixedGems() {
				fixed = append(fixed, a.itemName(gid, save.ItemName(gid)))
			}
		}
		a.disabled(c, id == "", func() {
			a.staticField(c, "GemSocket", sockets)
			a.staticField(c, "EquippedBy", a.namesOrNone(names))
			a.staticField(c, "FixedGem", a.namesOrNone(fixed))
		})
		a.disabled(c, !editable, func() {
			v := a.flag(key+"/"+id+"/favorite", favorite)
			if ui.Checkbox(c, v, a.T("Favorite")).Changed() {
				n := *v
				a.stage(key+"/"+id+"/favorite", func(s *save.Document) { s.OwnedEquipment(slot, index).SetFavorite(n) })
			}
		})
	})
}
func (a *App) gemDefinitionPicker(c *ui.Context, key string, effect, rank *string, compatible func(save.GemDefinition) bool) *save.GemDefinition {
	effects := []option{}
	seen := map[int]bool{}
	for _, r := range save.GemDefinitions {
		if compatible != nil && !compatible(r) {
			continue
		}
		if !seen[r.EffectId] {
			effects = append(effects, option{rid(r.EffectId), a.name("gem", r.EffectId, save.GemName(r.EffectId))})
			seen[r.EffectId] = true
		}
	}
	if *effect == "" && len(effects) > 0 {
		*effect = effects[0].ID
	}
	a.combo(c, "GemEffect", key+"/effect", effect, effects)
	ranks := []option{}
	for _, r := range save.GemDefinitions {
		if r.EffectId == parseID(*effect) && (compatible == nil || compatible(r)) {
			ranks = append(ranks, option{rid(r.Rank), rid(r.Rank)})
		}
	}
	valid := false
	for _, o := range ranks {
		if o.ID == *rank {
			valid = true
		}
	}
	if !valid && len(ranks) > 0 {
		*rank = ranks[0].ID
	}
	a.combo(c, "GemRank", key+"/rank", rank, ranks)
	return save.GemDefinitionFor(parseID(*effect), parseID(*rank))
}
func (a *App) gemsPage(c *ui.Context) {
	rows := []row{}
	if a.Save != nil {
		for _, g := range a.Save.Gems() {
			if !g.IsCylinder() {
				rows = append(rows, row{rid(g.Index), a.gemLabel(g), ""})
			}
		}
	}
	a.split(c, "gems", "Gems", rows, listHeader{}, func() {
		id := a.selectedID("gems")
		canDelete := a.Save != nil && id != "" && a.Save.Gem(parseID(id)).CanDelete()
		ui.Row(c).Gap(8).Children(func() {
			a.inventoryCommand(c, "Add", a.Save != nil && a.Save.Campaign() != "Unknown", func() { a.openInventoryDialog("gem", "gems", 0) })
			a.action(c, "Delete", canDelete, func(s *save.Document) { s.Gem(parseID(id)).Delete() })
		})
	}, func(id string) {
		a.heading(c, "GemEditor", "")
		index := parseID(id)
		if a.Save != nil && id != "" {
			g := a.Save.Gem(index)
			a.disabled(c, !g.CanEdit(), func() {
				key := "gem/" + id
				type draftGem struct {
					Effect, Rank string
					Strength     float64
				}
				v := draft(a, key, func() draftGem { return draftGem{rid(g.EffectID()), rid(g.Rank()), float64(g.Strength())} })
				oldEffect, oldRank := v.Effect, v.Rank
				slots := g.EquippedSlots()
				rule := a.gemDefinitionPicker(c, key, &v.Effect, &v.Rank, func(r save.GemDefinition) bool {
					for _, slot := range slots {
						if !r.Allows(slot) {
							return false
						}
					}
					return true
				})
				if rule != nil {
					changed := oldEffect != v.Effect || oldRank != v.Rank
					if changed {
						v.Strength = float64(rule.Minimum)
					}
					if a.formNumber(c, "GemStrength", key+"/value", &v.Strength, float64(rule.Minimum), float64(rule.Maximum)) {
						changed = true
					}
					a.staticField(c, "GemChance", fmt.Sprintf("%d%%", rule.Chance))
					if changed {
						effect, rank, value := parseID(v.Effect), parseID(v.Rank), int(v.Strength)
						a.stage(key, func(s *save.Document) { s.Gem(index).Set(effect, rank, value) })
					}
					a.action(c, "MaxGem", g.CanEdit(), func(s *save.Document) { s.Gem(index).Maximize() })
				}
			})
		}
		if id == "" {
			a.disabled(c, true, func() {
				effect, rank := "", ""
				value := 0.0
				a.combo(c, "GemEffect", "emptyGem/effect", &effect, nil)
				a.combo(c, "GemRank", "emptyGem/rank", &rank, nil)
				a.formNumber(c, "GemStrength", "emptyGem/value", &value, 0, 0)
				a.staticField(c, "GemChance", "")
				a.action(c, "MaxGem", false, func(*save.Document) {})
			})
		}
	})
}

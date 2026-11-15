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
			opts := []option{}
			for i := 1; i <= 5; i++ {
				opts = append(opts, option{rid(i), a.T(save.SlotNames[i])})
			}
			selected := draft(a, "armorSlot", func() string { return "1" })
			a.combo(c, "Armor", "armorSlot", selected, opts)
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
	a.split(c, key, save.KindNames[kind], rows, listHeader{}, func() { a.action(c, "MaxQuantities", kind != 12, func(s *save.Document) { s.MaxItems(kind) }) }, func(id string) {
		a.disabled(c, kind == 12, func() {
			a.heading(c, "ItemEditor", "")
			quantity := 0
			favorite := false
			editable := false
			index := parseID(id)
			if a.Save != nil && id != "" {
				x := a.Save.Item(kind, index)
				quantity = x.Quantity()
				favorite = x.Favorite()
				editable = x.CanEdit()
			}
			a.disabled(c, !editable, func() {
				a.numberEdit(c, "Quantity", key+"/"+id+"/quantity", quantity, 1, 99, func(s *save.Document, v int) { s.Item(kind, index).SetQuantity(v) })
				v := a.flag(key+"/"+id+"/favorite", favorite)
				if ui.Checkbox(c, v, a.T("Favorite")).Changed() {
					n := *v
					a.stage(key+"/"+id+"/favorite", func(s *save.Document) { s.Item(kind, index).SetFavorite(n) })
				}
				a.action(c, "Delete", editable, func(s *save.Document) { s.Item(kind, index).Delete() })
			})
			if kind == 12 {
				return
			}
			opts := []option{}
			for _, def := range save.InventoryDefinitions {
				if def.Kind == kind {
					opts = append(opts, option{rid(def.Id), a.itemName(def.Id, def.Name)})
				}
			}
			selected := draft(a, key+"/newID", func() string {
				if len(opts) > 0 {
					return opts[0].ID
				}
				return ""
			})
			quantityDraft := draft(a, key+"/newQty", func() float64 { return 1 })
			a.combo(c, "AddItem", key+"/newID", selected, opts)
			a.formNumber(c, "Quantity", key+"/newQty", quantityDraft, 1, 99)
			a.action(c, "Add", *selected != "", func(s *save.Document) { s.AddItem(parseID(*selected), int(*quantityDraft)) })
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
	a.split(c, key, save.SlotNames[slot], rows, listHeader{}, nil, func(id string) {
		a.heading(c, "Equipment", "")
		favorite := false
		editable, canDelete := false, false
		index := parseID(id)
		if a.Save != nil && id != "" {
			x := a.Save.OwnedEquipment(slot, index)
			favorite = x.Favorite()
			editable = x.CanEdit()
			canDelete = x.CanDelete()
			a.staticField(c, "GemSocket", x.Sockets())
			names := []string{}
			for _, cid := range x.EquippedBy() {
				names = append(names, save.CharacterName(cid, a.Language))
			}
			a.staticField(c, "EquippedBy", a.namesOrNone(names))
			fixed := []string{}
			for _, gid := range x.FixedGems() {
				fixed = append(fixed, a.itemName(gid, save.ItemName(gid)))
			}
			a.staticField(c, "FixedGem", a.namesOrNone(fixed))
		}
		a.disabled(c, !editable, func() {
			v := a.flag(key+"/"+id+"/favorite", favorite)
			if ui.Checkbox(c, v, a.T("Favorite")).Changed() {
				n := *v
				a.stage(key+"/"+id+"/favorite", func(s *save.Document) { s.OwnedEquipment(slot, index).SetFavorite(n) })
			}
		})
		a.action(c, "Delete", canDelete, func(s *save.Document) { s.OwnedEquipment(slot, index).Delete() })
		opts := []option{}
		chars := []option{}
		if a.Save != nil {
			for _, def := range a.Save.CreatableEquipment(slot) {
				opts = append(opts, option{rid(def.Id), a.itemName(def.Id, def.Name)})
			}
			for _, ch := range a.Save.Characters() {
				chars = append(chars, option{rid(ch.ID), save.CharacterName(ch.ID, a.Language)})
			}
		}
		selected := draft(a, key+"/newID", func() string {
			if len(opts) > 0 {
				return opts[0].ID
			}
			return ""
		})
		target := draft(a, key+"/target", func() string {
			if len(chars) > 0 {
				return chars[0].ID
			}
			return ""
		})
		a.combo(c, "AddEquipment", key+"/newID", selected, opts)
		a.action(c, "Add", *selected != "", func(s *save.Document) { s.AddEquipment(parseID(*selected)) })
		a.combo(c, "EquipmentTarget", key+"/target", target, chars)
		a.action(c, "FillMissingEquipment", *target != "", func(s *save.Document) { s.FillEquipment(slot, parseID(*target)) })
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
	a.split(c, "gems", "Gems", rows, listHeader{}, nil, func(id string) {
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
				a.action(c, "Delete", g.CanDelete(), func(s *save.Document) { s.Gem(index).Delete() })
			})
		}
		a.heading(c, "AddGem", "")
		effect := draft(a, "newGem/effect", func() string { return "" })
		rank := draft(a, "newGem/rank", func() string { return "1" })
		rule := a.gemDefinitionPicker(c, "newGem", effect, rank, nil)
		value := draft(a, "newGem/value/"+*effect+"/"+*rank, func() float64 {
			if rule != nil {
				return float64(rule.Minimum)
			}
			return 0
		})
		if rule != nil {
			a.formNumber(c, "GemStrength", "newGem/value", value, float64(rule.Minimum), float64(rule.Maximum))
			a.action(c, "Add", true, func(s *save.Document) { s.AddGem(parseID(*effect), parseID(*rank), int(*value)) })
		}
	})
}

package app

import (
	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/xbde-save-editor/internal/save"
)

type inventoryDialog struct {
	Open                 bool
	Mode, ListKey        string
	Category             int
	Choice, Effect, Rank string
	Quantity, Strength   float64
	Error                string
}

func (a *App) selectedID(key string) string {
	s := a.table(key)
	if s.Selected >= 0 && s.Selected < len(s.Rows) {
		return s.Rows[s.Selected].ID
	}
	return ""
}
func (a *App) selectInventoryRow(key string, index int) {
	s := a.table(key)
	s.Query = ""
	s.RequestedID = rid(index)
}
func (a *App) inventoryCommand(c *ui.Context, label string, enabled bool, run func()) {
	ui.Row(c.Key(label)).Children(func() {
		if ui.Button(c, a.T(label)).Disabled(!enabled || a.Save == nil).Clicked() {
			if a.ApplyPending("") == nil {
				run()
			}
		}
	})
}
func (a *App) openInventoryDialog(mode, key string, category int) {
	a.inventoryDialog = &inventoryDialog{Open: true, Mode: mode, ListKey: key, Category: category, Quantity: 1, Rank: "1"}
}
func (a *App) inventoryDialogView(c *ui.Context) {
	d := a.inventoryDialog
	if d == nil {
		return
	}
	title := "AddItem"
	switch d.Mode {
	case "equipment":
		title = "AddEquipment"
	case "gem":
		title = "AddGem"
	case "fill":
		title = "FillMissingEquipment"
	}
	ui.Modal(c, &d.Open, func() {
		width, _ := c.Size()
		ui.Column(c.Key("inventoryDialog")).Width(460).MaxWidth(width - 80).Gap(12).Children(func() {
			ui.Text(c, a.T(title)).Bold()
			ready := true
			a.editorForm(c, func() {
				switch d.Mode {
				case "item":
					options := []option{}
					for _, def := range save.InventoryDefinitions {
						if def.Kind == d.Category {
							options = append(options, option{rid(def.Id), a.itemName(def.Id, def.Name)})
						}
					}
					if d.Choice == "" && len(options) > 0 {
						d.Choice = options[0].ID
					}
					a.combo(c, "Items", "newItem/definition", &d.Choice, options)
					a.formNumber(c, "Quantity", "newItem/quantity", &d.Quantity, 1, 99)
					ready = d.Choice != ""
				case "equipment":
					options := []option{}
					for _, def := range a.Save.CreatableEquipment(d.Category) {
						options = append(options, option{rid(def.Id), a.itemName(def.Id, def.Name)})
					}
					if d.Choice == "" && len(options) > 0 {
						d.Choice = options[0].ID
					}
					a.combo(c, "Equipment", "newEquipment/definition", &d.Choice, options)
					ready = d.Choice != ""
				case "fill":
					options := []option{}
					for _, ch := range a.Save.Characters() {
						options = append(options, option{rid(ch.ID), save.CharacterName(ch.ID, a.Language)})
					}
					if d.Choice == "" && len(options) > 0 {
						d.Choice = options[0].ID
					}
					a.combo(c, "EquipmentTarget", "fillEquipment/character", &d.Choice, options)
					ready = d.Choice != ""
				case "gem":
					oldEffect, oldRank := d.Effect, d.Rank
					rule := a.gemDefinitionPicker(c, "newGem", &d.Effect, &d.Rank, nil)
					ready = rule != nil
					if rule != nil {
						if oldEffect != d.Effect || oldRank != d.Rank {
							d.Strength = float64(rule.Minimum)
						}
						a.formNumber(c, "GemStrength", "newGem/value", &d.Strength, float64(rule.Minimum), float64(rule.Maximum))
					}
				}
			})
			if d.Error != "" {
				ui.Text(c, d.Error).TextColor(c.Theme().Danger)
			}
			ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
				if ui.Button(c, a.T("Cancel")).Clicked() {
					d.Open = false
				}
				submit := "Add"
				if d.Mode == "fill" {
					submit = "FillMissingEquipment"
				}
				if ui.PrimaryButton(c, a.T(submit)).Disabled(!ready).Clicked() {
					index := -1
					if a.perform(func(s *save.Document) {
						switch d.Mode {
						case "item":
							index = s.AddItem(parseID(d.Choice), int(d.Quantity)).Index
						case "equipment":
							index = s.AddEquipment(parseID(d.Choice)).Index
						case "gem":
							index = s.AddGem(parseID(d.Effect), parseID(d.Rank), int(d.Strength)).Index
						case "fill":
							created := s.FillEquipment(d.Category, parseID(d.Choice))
							if len(created) > 0 {
								index = created[0].Index
							}
						}
					}) {
						if index >= 0 {
							a.selectInventoryRow(d.ListKey, index)
						}
						d.Open = false
					} else {
						d.Error = a.Status
					}
				}
			})
		})
	})
	if !d.Open {
		a.inventoryDialog = nil
	}
}

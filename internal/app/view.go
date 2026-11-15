package app

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/xbde-save-editor/internal/catalog"
	"github.com/jinghaihan/xbde-save-editor/internal/save"
	"path/filepath"
	"strconv"
)

func (a *App) View(c *ui.Context) {
	prev := a.focusedInput
	a.focusedInput = ""
	c.SetTheme(a.currentTheme())
	c = c.Key(a.revision)
	root := ui.Column(c).Fill().Background(c.Theme().Background)
	root.Children(func() {
		labels := []string{}
		for _, p := range pages {
			labels = append(labels, a.T(p))
		}
		ui.ScrollHorizontal(c).PaddingX(16 - c.Theme().Space(1.5)).Children(func() {
			old := a.Page
			if ui.Tabs(c, &a.Page, labels...).Changed() && a.Save != nil && a.ApplyPending("") != nil {
				a.Page = old
			}
		})
		ui.Column(c.Key(pages[a.Page])).Grow(1).Basis(0).MinHeight(0).Padding(16).Gap(12).Disabled(a.Save == nil).Children(func() {
			switch a.Page {
			case 0:
				a.mainPage(c)
			case 1:
				a.charactersPage(c)
			case 2:
				a.itemsPage(c)
			case 3:
				a.affinityPage(c)
			case 4:
				a.achievementsPage(c)
			case 5:
				a.collectionPage(c)
			}
		})
		if a.Status != "" {
			ui.Text(c, a.Status).Padding(0, 16, 12)
		}
	})
	if a.Save != nil && len(a.pending) > 0 && (a.focusedInput == "" || prev != "" && prev != a.focusedInput) {
		a.ApplyPending("")
	}
	if files := root.DroppedFiles(); len(files) > 0 {
		a.openPath(files[0])
	}
	if ui.AlertDialog(c, &a.confirm, a.confirmLabel, "", a.T("Cancel"), a.T("Discard")) == 1 && a.confirmAction != nil {
		f := a.confirmAction
		a.confirmAction = nil
		f()
	}
}
func (a *App) scroll(c *ui.Context, fn func()) {
	ui.Scroll(c).Grow(1).MinHeight(0).Children(func() { ui.Column(c).Gap(12).MinWidth(0).Children(fn) })
}
func (a *App) action(c *ui.Context, label string, enabled bool, fn edit) {
	ui.Row(c.Key(label)).Disabled(!enabled || a.Save == nil).Children(func() { a.button(c, label, func() { a.perform(fn) }) })
}
func (a *App) mainPage(c *ui.Context) {
	a.scroll(c, func() {
		a.sectionColumns(c, func() {
			a.section(c, "Resources", func() {
				money, nopon := 0, 0
				if a.Save != nil {
					money = a.Save.Money()
					nopon = a.Save.Noponstones()
				}
				editorForm(c, func() {
					a.numberEdit(c, "Money", "main/money", money, 0, max(money, save.MaximumCurrency), func(s *save.Document, v int) { s.SetResources(&v, nil) })
					a.numberEdit(c, "Noponstones", "main/nopon", nopon, 0, max(nopon, save.MaximumCurrency), func(s *save.Document, v int) { s.SetResources(nil, &v) })
				})
			})
			a.section(c, "Colony6Title", func() {
				state := &save.ColonyState{}
				if a.Save != nil && a.Save.Colony6() != nil {
					state = a.Save.Colony6()
				}
				editorForm(c, func() {
					for _, f := range []struct {
						k string
						v int
					}{{"Housing", state.Housing}, {"Commerce", state.Commerce}, {"Nature", state.Nature}, {"Special", state.Special}, {"Colony6OverallLevel", state.OverallLevel}, {"Colony6Development", state.Development}, {"Colony6Population", state.Population}} {
						a.staticField(c, f.k, f.v)
					}
				})
				if state.UnavailableReason != "" {
					ui.Text(c, state.UnavailableReason).TextColor(c.Theme().TextMuted)
				}
				a.action(c, "MaxColony6", state.CanMaximize, func(s *save.Document) { s.MaxColony6() })
			})
		}, func() {
			a.section(c, "SaveFile", func() {
				file, campaign, count := "", "", 0
				if a.Save != nil {
					file = filepath.Base(a.Path)
					campaign = a.T(a.Save.Campaign())
					count = len(a.Save.Characters())
				}
				editorForm(c, func() {
					a.staticField(c, "FileName", file)
					a.staticField(c, "Campaign", campaign)
					a.staticField(c, "Characters", count)
				})
			})
		})
	})
}
func rid(n int) string     { return strconv.Itoa(n) }
func parseID(s string) int { v, _ := strconv.Atoi(s); return v }
func (a *App) disabled(c *ui.Context, disabled bool, fn func()) {
	ui.Column(c).MinWidth(0).Gap(8).Disabled(disabled).Children(fn)
}
func (a *App) intChoice(c *ui.Context, label, key string, value int, options []option, fn func(int)) {
	v := a.choice(key, rid(value))
	if a.combo(c, label, key, v, options) {
		fn(parseID(*v))
	}
}
func numberedOptions(lo, hi int) []option {
	out := []option{}
	for i := lo; i <= hi; i++ {
		out = append(out, option{rid(i), rid(i)})
	}
	return out
}
func (a *App) selectedList(c *ui.Context, key, title string, rows []row, footer func()) int {
	return parseID(a.list(c, key, title, rows, listHeader{}, footer))
}
func (a *App) note(c *ui.Context, k string) { ui.Text(c, a.T(k)).TextColor(c.Theme().TextMuted) }
func (a *App) charactersPage(c *ui.Context) {
	rows := []row{}
	if a.Save != nil {
		for _, x := range a.Save.Characters() {
			rows = append(rows, row{rid(x.ID), save.CharacterName(x.ID, a.Language), fmt.Sprintf("Lv. %d", x.Level())})
		}
	}
	a.split(c, "characters", "Characters", rows, listHeader{}, func() {
		if *a.tab("characterTabs") == 0 {
			a.action(c, "MaxAllAP", true, func(s *save.Document) {
				for _, x := range s.Characters() {
					v := 99999999
					x.SetResources(&v, nil, nil)
				}
			})
		}
		if *a.tab("characterTabs") == 1 {
			a.action(c, "LearnMaxAllArts", a.Save != nil, func(s *save.Document) {
				for _, x := range s.Characters() {
					x.MaxArts(true)
				}
			})
		}
		if *a.tab("characterTabs") == 2 {
			a.action(c, "UnlockMaxAllSkills", a.Save != nil && a.Save.MainSupported(), func(s *save.Document) {
				for _, x := range s.Characters() {
					x.MaxSkills(true)
				}
			})
		}
	}, func(id string) {
		tab := a.tabsView(c, "characterTabs", "General", "Arts", "Skills", "Equipment", "SkillLinks")
		cid := parseID(id)
		if a.Save == nil || id == "" {
			a.emptyCharacter(c, tab)
			return
		}
		switch tab {
		case 0:
			a.characterGeneral(c, cid)
		case 1:
			a.characterArts(c, cid)
		case 2:
			a.characterSkills(c, cid)
		case 3:
			a.characterEquipment(c, cid)
		case 4:
			a.characterLinks(c, cid)
		}
	})
}
func (a *App) characterGeneral(c *ui.Context, id int) {
	x := a.Save.Character(id)
	key := "character/" + rid(id)
	a.sectionColumns(c, func() {
		ui.Text(c, a.T("General")).Bold()
		editorForm(c, func() {
			a.disabled(c, !x.CanEditProgression(), func() {
				a.intChoice(c, "Level", key+"/level", x.Level(), numberedOptions(x.MinimumLevel(), 99), func(v int) { a.stage(key+"/level", func(s *save.Document) { s.Character(id).SetProgression(&v, nil) }) })
				a.numberEdit(c, "Experience", key+"/exp", x.Experience(), 0, max(99999999, x.Experience()), func(s *save.Document, v int) { s.Character(id).SetProgression(nil, &v) })
			})
			a.numberEdit(c, "AP", key+"/ap", x.AP(), 0, max(99999999, x.AP()), func(s *save.Document, v int) { s.Character(id).SetResources(&v, nil, nil) })

			a.disabled(c, a.Save.Campaign() != "MainStory", func() {
				a.numberEdit(c, "AffinityCoins", key+"/coins", x.AffinityCoins(), 0, max(999, x.AffinityCoins()), func(s *save.Document, v int) { s.Character(id).SetResources(nil, &v, nil) })
			})
		})
		a.action(c, "MaxAP", true, func(s *save.Document) { v := 99999999; s.Character(id).SetResources(&v, nil, nil) })
	}, func() {
		ui.Text(c, a.T("ExpertMode")).Bold()
		editorForm(c, func() {
			a.staticField(c, "HighestLevel", x.ExpertLevel())
			a.staticField(c, "RecordedExperience", x.ExpertExperience())
			a.numberEdit(c, "ReserveExperience", key+"/reserve", x.ReserveExperience(), 0, max(199999998, x.ReserveExperience()), func(s *save.Document, v int) { s.Character(id).SetResources(nil, nil, &v) })
		})
		a.action(c, "MaxReserveExperience", true, func(s *save.Document) { v := 199999998; s.Character(id).SetResources(nil, nil, &v) })
	})
}
func (a *App) characterArts(c *ui.Context, id int) {
	rows := []row{}
	for _, x := range a.Save.Character(id).Arts() {
		value := a.T("NotLearned")
		if x.Learned() {
			value = fmt.Sprintf("Lv. %d", x.Level())
		}
		rows = append(rows, row{rid(x.Def.Id), a.artName(x), value})
	}
	key := "arts/" + rid(id)
	_, height := c.Size()
	ui.Row(c).Height(max(340, height-180)).Gap(12).AlignItems(ui.Stretch).Children(func() {
		selected := ""
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Children(func() {
			selected = a.list(c, key, "", rows, listHeader{}, func() {
				a.action(c, "LearnMaxCharacterArts", true, func(s *save.Document) { s.Character(id).MaxArts(true) })
			})
		})
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(8).Children(func() {
			if selected == "" {
				return
			}
			artID := parseID(selected)
			x := a.Save.Character(id).Art(artID)
			status := "Upgradeable"
			if !x.Learned() {
				status = "NotLearned"
			} else if x.Def.IsTalent {
				status = "FixedTalent"
			}
			a.staticField(c, "ArtStatus", a.T(status))
			a.staticField(c, "MaximumLevel", x.MaximumLevel())
			a.disabled(c, !x.CanEdit(), func() {
				a.intChoice(c, "Level", key+"/"+rid(artID), x.Level(), numberedOptions(1, x.MaximumLevel()), func(v int) {
					a.stage(key+"/"+rid(artID), func(s *save.Document) { s.Character(id).Art(artID).SetLevel(v) })
				})
			})
			a.action(c, "LearnArt", x.CanLearn(), func(s *save.Document) { s.Character(id).Art(artID).Learn(1) })
			a.action(c, "MaxArt", x.CanEdit(), func(s *save.Document) { z := s.Character(id).Art(artID); z.SetLevel(z.MaximumLevel()) })
		})
	})
}
func (a *App) artName(x save.Art) string {
	name := a.name("art", x.Def.Id, x.Def.Name)
	if x.Def.IsMonado() {
		name = a.itemName(1, save.ItemName(1)) + " " + name
	}
	return name
}
func (a *App) characterSkills(c *ui.Context, id int) {
	a.disabled(c, !a.Save.MainSupported(), func() {
		trees := a.Save.Character(id).SkillTrees()
		a.tiles(c, len(trees), func(i int) {
			t := trees[i]
			index := t.Def.Index
			key := fmt.Sprintf("skills/%d/%d", id, index)
			a.section(c, a.treeName(t), func() {
				for i, s := range t.Def.Skills {
					mark := ""
					if i < t.LearnedCount() {
						mark = "✓ "
					}
					ui.Text(c, mark+a.name("skill", s.Id, s.Name))
				}
				editorForm(c, func() {
					a.disabled(c, !t.CanEdit(), func() {
						a.numberEdit(c, "LearnedSkills", key+"/count", t.LearnedCount(), t.MinimumCount(), 5, func(s *save.Document, v int) { s.Character(id).SkillTree(index).SetLearnedCount(v) })
					})
					a.disabled(c, !t.CanEdit() || t.LearnedCount() >= 5, func() {
						a.numberEdit(c, "SkillProgress", key+"/sp", t.Progress(), 0, max(t.Progress(), t.MaximumProgress()), func(s *save.Document, v int) { s.Character(id).SkillTree(index).SetProgress(v) })
					})
				})
				ui.Row(c).Gap(8).Children(func() {
					a.action(c, "UnlockSkillBranch", !t.Unlocked(), func(s *save.Document) { s.Character(id).SkillTree(index).Unlock() })
					a.action(c, "MaxTree", t.CanEdit(), func(s *save.Document) { s.Character(id).SkillTree(index).SetLearnedCount(5) })
				})
			})
		})
		a.action(c, "UnlockMaxCharacterSkills", true, func(s *save.Document) { s.Character(id).MaxSkills(true) })
	})
}
func (a *App) characterLinks(c *ui.Context, id int) {
	cost, ok := a.Save.Character(id).LinkedCost()
	if ok {
		ui.Text(c, replaceFormat(a.T("SkillLinkCoins"), rid(cost), rid(a.Save.Character(id).AffinityCoins())))
	}
	links := a.Save.Character(id).SkillLinks()
	if len(links) == 0 {
		a.note(c, "SkillLinksUnavailable")
		return
	}
	sources := []int{}
	for _, l := range links {
		if len(sources) == 0 || sources[len(sources)-1] != l.SourceID {
			sources = append(sources, l.SourceID)
		}
	}
	a.tiles(c, len(sources), func(i int) {
		source := sources[i]
		a.section(c, save.CharacterName(source, a.Language), func() {
			for _, l := range links {
				if l.SourceID != source {
					continue
				}
				index := l.Index
				opts := []option{{"0", a.T("None")}}
				for _, s := range l.Choices() {
					opts = append(opts, option{rid(s.Id), fmt.Sprintf("%s · %d", a.name("skill", s.Id, s.Name), s.AffinityCoins)})
				}
				if l.SkillID() != 0 && !l.CanSet(l.SkillID()) {
					opts = append(opts, option{rid(l.SkillID()), fmt.Sprintf("#%d", l.SkillID())})
				}
				a.disabled(c, !l.CanEdit(), func() {
					key := fmt.Sprintf("links/%d/%d/%d", id, source, index)
					a.intChoice(c, fmt.Sprintf("%d · %s", index, a.T("SkillShape"+save.Shapes[l.Shape()])), key, l.SkillID(), opts, func(v int) {
						a.stage(key, func(s *save.Document) { s.Character(id).SkillLink(source, index).SetSkill(v) })
					})
				})
			}
		})
	})
}
func (a *App) characterEquipment(c *ui.Context, id int) {
	a.tiles(c, len(save.SlotNames), func(slot int) {
		name := save.SlotNames[slot]
		x := a.Save.Character(id).Equipment(slot)
		key := fmt.Sprintf("equip/%d/%d", id, slot)
		a.section(c, name, func() {
			opts := []option{}
			for _, v := range x.AvailableItems() {
				opts = append(opts, option{rid(v.Index), a.itemName(v.ItemID(), v.Name()) + " [" + rid(v.Index) + "]"})
			}
			if len(opts) == 0 {
				opts = append(opts, option{rid(x.Index()), x.Name()})
			}
			a.disabled(c, !x.CanSwitch(), func() {
				a.intChoice(c, "Equipment", key, x.Index(), opts, func(v int) { a.stage(key, func(s *save.Document) { s.Character(id).Equipment(slot).Equip(v) }) })
			})
			for _, socket := range x.Sockets() {
				index := socket.Index
				opts := []option{{"-1", a.T("None")}}
				for _, g := range socket.AvailableGems() {
					opts = append(opts, option{rid(g.Index), a.gemLabel(g) + " [" + rid(g.Index) + "]"})
				}
				if socket.FixedID() != 0 {
					a.staticField(c, a.T("FixedGem")+" "+rid(index), a.itemName(socket.FixedID(), socket.Name()))
					continue
				}
				a.disabled(c, !socket.CanEdit(), func() {
					k := key + "/gem/" + rid(index)
					a.intChoice(c, a.T("GemSocket")+" "+rid(index), k, socket.GemIndex(), opts, func(v int) {
						a.stage(k, func(s *save.Document) {
							e := s.Character(id).Equipment(slot)
							indices := []int{}
							for _, ss := range e.Sockets() {
								n := ss.GemIndex()
								if ss.Index == index {
									n = v
								}
								indices = append(indices, n)
							}
							e.SetGems(indices)
						})
					})
				})
			}
		})
	})
}

func (a *App) tiles(c *ui.Context, n int, fn func(int)) {
	for start := 0; start < n; start += 2 {
		ui.Row(c.Key(start)).Gap(12).AlignItems(ui.Start).Children(func() {
			for i := start; i < min(start+2, n); i++ {
				ui.Column(c.Key(i)).Grow(1).Basis(0).MinWidth(0).Gap(8).Children(func() { fn(i) })
			}
			if start+1 >= n {
				ui.Column(c).Grow(1).Basis(0)
			}
		})
	}
}

func (a *App) emptyCharacter(c *ui.Context, tab int) {
	emptyNumber := func(label string) { a.numberField(c, label, "empty/"+label, 0, 0, 0, func(int) {}) }
	switch tab {
	case 0:
		a.sectionColumns(c, func() {
			ui.Text(c, a.T("General")).Bold()
			editorForm(c, func() {
				for _, key := range []string{"Level", "Experience", "AP", "AffinityCoins"} {
					emptyNumber(key)
				}
			})
		}, func() {
			ui.Text(c, a.T("ExpertMode")).Bold()
			editorForm(c, func() {
				a.staticField(c, "HighestLevel", "")
				a.staticField(c, "RecordedExperience", "")
				emptyNumber("ReserveExperience")
			})
		})
	case 1:
		ui.Row(c).Gap(12).Children(func() {
			ui.Column(c).Grow(1).Basis(0).Height(340).Children(func() { a.list(c, "emptyArts", "", nil, listHeader{}, nil) })
			ui.Column(c).Grow(1).Basis(0).Gap(8).Children(func() { a.staticField(c, "ArtStatus", ""); a.staticField(c, "MaximumLevel", ""); emptyNumber("Level") })
		})
	case 2:
		a.tiles(c, 5, func(i int) {
			a.section(c, "Skills", func() { emptyNumber("LearnedSkills"); emptyNumber("SkillProgress") })
		})
	case 3:
		a.tiles(c, 6, func(i int) {
			a.section(c, save.SlotNames[i], func() { s := ""; a.combo(c, "Equipment", "emptyEquipment/"+rid(i), &s, nil) })
		})
	case 4:
		a.note(c, "SkillLinksUnavailable")
	}
}

func (a *App) name(domain string, id int, fallback string) string {
	return catalog.Name(a.Language, domain, rid(id), fallback)
}
func (a *App) itemName(id int, fallback string) string { return catalog.Item(a.Language, id, fallback) }
func (a *App) treeName(t save.SkillTree) string {
	return catalog.Name(a.Language, "tree", rid(t.Def.CharacterId)+"/"+rid(t.Def.Index), t.Def.Name)
}
func (a *App) gemLabel(g save.Gem) string {
	return fmt.Sprintf("%s %d · %d", a.name("gem", g.EffectID(), save.GemName(g.EffectID())), g.Rank(), g.Strength())
}

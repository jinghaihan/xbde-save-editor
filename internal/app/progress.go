package app

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"github.com/jinghaihan/xbde-save-editor/internal/save"
	"strings"
)

func (a *App) affinityPage(c *ui.Context) {
	if a.tabsView(c, "affinityTabs", "Characters", "Regions") == 1 {
		a.scroll(c, func() {
			for id, name := range save.RegionNames {
				id := id + 1
				a.section(c, a.name("region", id, name), func() {
					points, stars := 0, 1
					if a.Save != nil && a.Save.Campaign() == "MainStory" {
						r := a.Save.Region(id)
						points = r.Points()
						stars = r.Stars()
					}
					a.disabled(c, a.Save == nil || !a.Save.MainSupported(), func() {
						a.editorForm(c, func() {
							key := "region/" + rid(id)
							a.numberEdit(c, "RegionPoints", key+"/points", points, 0, max(10000, points), func(s *save.Document, v int) { s.Region(id).SetPoints(v) })
							a.intChoice(c, "RegionStars", key+"/stars", stars, numberedOptions(1, 5), func(v int) { a.stage(key+"/stars", func(s *save.Document) { s.Region(id).SetStars(v) }) })
						})
					})
				})
			}
			a.action(c, "MaxAllRegions", a.Save != nil && a.Save.MainSupported(), func(s *save.Document) {
				for _, r := range s.Regions() {
					r.SetPoints(10000)
				}
			})
		})
		return
	}
	rows := []row{}
	if a.Save != nil {
		for _, p := range a.Save.Affinities() {
			rows = append(rows, row{rid(p.FirstID) + "/" + rid(p.SecondID), save.CharacterName(p.FirstID, a.Language) + " / " + save.CharacterName(p.SecondID, a.Language), rid(p.Points())})
		}
	}
	a.split(c, "affinity", "Affinity", rows, listHeader{}, func() {
		a.action(c, "MaxAllAffinity", a.Save != nil && a.Save.MainSupported(), func(s *save.Document) {
			for _, p := range s.Affinities() {
				p.SetPoints(5000)
			}
		})
	}, func(id string) {
		first, second, points, slots1, slots2 := 0, 0, 0, 0, 0
		if a.Save != nil && id != "" {
			ids := strings.Split(id, "/")
			first, second = parseID(ids[0]), parseID(ids[1])
			p := a.Save.Affinity(first, second)
			points = p.Points()
			slots1, slots2 = p.Slots(first, second), p.Slots(second, first)
		}
		a.disabled(c, id == "" || a.Save == nil || !a.Save.MainSupported(), func() {
			a.editorForm(c, func() {
				a.numberEdit(c, "AffinityPoints", "affinity/"+id, points, 0, max(5000, points), func(s *save.Document, v int) { s.Affinity(first, second).SetPoints(v) })
				a.staticField(c, "AffinitySlots", fmt.Sprintf("%d / %d", slots1, slots2))
			})
			a.note(c, "AffinitySlotHint")
			a.action(c, "MaxAffinity", id != "", func(s *save.Document) { s.Affinity(first, second).SetPoints(5000) })
		})
	})
}
func (a *App) achievementsPage(c *ui.Context) {
	category := a.filter("achievementCategory", "0")
	state := a.filter("achievementState", "0")
	rows := []row{}
	if a.Save != nil && a.Save.MainSupported() {
		for _, x := range a.Save.Achievements() {
			cat := parseID(*category)
			st := parseID(*state)
			if cat != 0 && x.Def.Category != cat-1 || st == 1 && !x.Completed() || st == 2 && x.Completed() || st == 3 && !x.Unmet() {
				continue
			}
			status := a.T("AchievementIncomplete")
			if x.Completed() {
				status = a.T("AchievementComplete")
			}
			if x.Unmet() {
				status = a.T("AchievementUnmetCounter")
			}
			rows = append(rows, row{rid(x.Def.Id), a.name("achievement", x.Def.Id, x.Def.Name), status})
		}
	}
	a.split(c, "achievements", "Achievements", rows, listHeader{controls: func() {
		a.combo(c, "AchievementCategory", "achCategory", category, []option{{"0", a.T("AllCategories")}, {"1", a.T("Trials")}, {"2", a.T("Records")}})
		a.combo(c, "ArtStatus", "achState", state, []option{{"0", a.T("AllStatuses")}, {"1", a.T("AchievementComplete")}, {"2", a.T("AchievementIncomplete")}, {"3", a.T("AchievementUnmetCounterFilter")}})
	}}, func() {
		a.action(c, "UnlockAllAchievements", a.Save != nil && a.Save.MainSupported(), func(s *save.Document) {
			for _, x := range s.Achievements() {
				x.Unlock()
			}
		})
		a.action(c, "RepairAllAchievementCounters", a.Save != nil && a.Save.MainSupported(), func(s *save.Document) {
			for _, x := range s.Achievements() {
				x.Repair()
			}
		})
	}, func(id string) {
		if a.Save != nil && !a.Save.MainSupported() {
			a.note(c, "AchievementsUnavailable")
		}
		condition, status, progress, reward := "", "", "", 0
		unlock, repair := false, false
		aid := parseID(id)
		if a.Save != nil && id != "" {
			x := a.Save.Achievement(aid)
			condition = a.name("achievement-condition", x.Def.Id, x.Def.Condition)
			reward = x.Def.RewardExperience
			status = a.T("AchievementIncomplete")
			if x.Completed() {
				status = a.T("AchievementComplete")
			}
			if x.Def.HasCounter() {
				progress = fmt.Sprintf("%d / %d", x.Progress(), x.Def.Required)
			}
			unlock = a.Save.MainSupported() && !x.Completed()
			repair = a.Save.MainSupported() && x.Unmet()
		}
		if condition != "" {
			ui.Text(c, condition)
		}
		a.disabled(c, id == "", func() {
			a.editorForm(c, func() {
				a.staticField(c, "ArtStatus", status)
				var rewardValue any = ""
				if id != "" {
					rewardValue = reward
				}
				a.staticField(c, "AchievementReward", rewardValue)
				a.staticField(c, "AchievementProgress", progress)
			})
		})
		a.action(c, "UnlockAchievement", unlock, func(s *save.Document) { s.Achievement(aid).Unlock() })
		a.action(c, "RepairAchievementCounter", repair, func(s *save.Document) { s.Achievement(aid).Repair() })
	})
}
func (a *App) collectionPage(c *ui.Context) {
	page := a.filter("collectionPage", "0")
	state := a.filter("collectionState", "0")
	rows := []row{}
	opts := []option{{"0", a.T("AllCategories")}}
	seen := map[int]bool{}
	if a.Save != nil {
		for _, x := range a.Save.Collection() {
			if !seen[x.Def.MapId] {
				opts = append(opts, option{rid(x.Def.MapId), a.name("map", x.Def.MapId, x.Def.MapName)})
				seen[x.Def.MapId] = true
			}
			if *page != "0" && x.Def.MapId != parseID(*page) || *state == "1" && !x.Registered() || *state == "2" && x.Registered() {
				continue
			}
			status := a.T("Unregistered")
			if x.Registered() {
				status = a.T("Registered")
			}
			rows = append(rows, row{rid(x.Def.Id), a.itemName(x.Def.ItemId, x.Def.Name()), status})
		}
	}
	a.split(c, "collection", "Collectopaedia", rows, listHeader{controls: func() {
		a.combo(c, "Collectopaedia", "collectionPage", page, opts)
		a.combo(c, "Registration", "collectionState", state, []option{{"0", a.T("AllStatuses")}, {"1", a.T("Registered")}, {"2", a.T("Unregistered")}})
	}}, func() {
		a.action(c, "CompletePage", *page != "0" && a.Save != nil && a.Save.CollectionSupported(), func(s *save.Document) { s.CompleteCollection(s.CollectionIDs(parseID(*page), 0)) })
		a.action(c, "CompleteCollection", a.Save != nil && a.Save.CollectionSupported(), func(s *save.Document) { s.CompleteCollection(s.CollectionIDs(0, 0)) })
	}, func(id string) {
		aid := parseID(id)
		registered := false
		mapID := parseID(*page)
		if a.Save != nil && id != "" {
			for _, x := range a.Save.Collection() {
				if x.Def.Id == aid {
					registered = x.Registered()
					mapID = x.Def.MapId
				}
			}
		}
		status := "Unregistered"
		if registered {
			status = "Registered"
		}
		a.heading(c, "Registration", "")
		a.staticField(c, "ArtStatus", a.T(status))
		a.action(c, "RegisterEntry", id != "" && !registered, func(s *save.Document) { s.CompleteCollection(s.CollectionIDs(0, aid)) })
		a.heading(c, "CollectionRewards", "")
		if a.Save != nil && a.Save.CollectionSupported() && mapID != 0 {
			plan := a.Save.PlanCollection(a.Save.CollectionIDs(mapID, 0))
			if len(plan.Rewards) == 0 {
				a.note(c, "None")
			}
			for _, r := range plan.Rewards {
				ui.Text(c, a.itemName(r.ItemId, r.Name))
			}
		}
	})
}

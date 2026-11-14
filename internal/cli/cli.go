package cli

import (
	"encoding/json"
	"fmt"
	"github.com/jinghaihan/xbde-save-editor/internal/save"
	"io"
	"path/filepath"
	"strconv"
	"strings"
)

var Version = "0.0.0"

type object = map[string]any

func num(s string) int {
	v, e := strconv.Atoi(s)
	if e != nil || v < 0 {
		panic("Expected a nonnegative integer.")
	}
	return v
}
func fail(s string) { panic(s) }
func need(a []string, n int) {
	if len(a) != n {
		fail("Invalid arguments; use --help.")
	}
}
func slot(s string) int {
	for i, v := range save.SlotNames {
		if strings.EqualFold(v, s) {
			return i
		}
	}
	fail("Unknown equipment slot.")
	return 0
}
func kind(s string) int {
	for i, v := range save.KindNames {
		if strings.EqualFold(v, s) {
			return i
		}
	}
	fail("Unknown inventory category.")
	return 0
}
func options(a []string, allowed ...string) map[string]int {
	m := map[string]int{}
	if len(a) == 0 || len(a)%2 != 0 {
		fail("Expected option/value pairs.")
	}
	for i := 0; i < len(a); i += 2 {
		ok := false
		for _, k := range allowed {
			if a[i] == k {
				ok = true
			}
		}
		if !ok {
			fail("Unknown option: " + a[i])
		}
		if _, exists := m[a[i]]; exists {
			fail("Duplicate option.")
		}
		m[a[i]] = num(a[i+1])
	}
	return m
}
func ptr(m map[string]int, k string) *int {
	if v, ok := m[k]; ok {
		return &v
	}
	return nil
}
func pick(m map[string]int, k string, fallback int) int {
	if v, ok := m[k]; ok {
		return v
	}
	return fallback
}
func Run(a []string, out, stderr io.Writer) (code int) {
	defer func() {
		if p := recover(); p != nil {
			fmt.Fprintln(stderr, p)
			code = 1
		}
	}()
	if len(a) == 0 || a[0] == "--help" {
		fmt.Fprintln(out, Help)
		return 0
	}
	if a[0] == "--version" {
		need(a, 1)
		fmt.Fprintln(out, Version)
		return 0
	}
	if len(a) < 2 {
		fail("Specify a save file.")
	}
	session, e := save.OpenSession(a[1])
	if e != nil {
		fail(e.Error())
	}
	d := session.Document
	emit := func(v any) {
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		if e := enc.Encode(v); e != nil {
			fail(e.Error())
		}
	}
	read := true
	switch a[0] {
	case "inspect":
		need(a, 2)
		chars := []object{}
		for _, c := range d.Characters() {
			chars = append(chars, object{"Id": c.ID, "Name": save.CharacterName(c.ID, "en"), "Level": c.Level(), "Experience": c.Experience(), "AP": c.AP(), "AffinityCoins": c.AffinityCoins(), "ExpertLevel": c.ExpertLevel(), "ExpertExperience": c.ExpertExperience(), "ReserveExperience": c.ReserveExperience()})
		}
		campaign := 0
		if d.Campaign() == "FutureConnected" {
			campaign = 1
		} else if d.Campaign() == "Unknown" {
			campaign = 2
		}
		emit(object{"File": filepath.Base(a[1]), "Campaign": campaign, "Size": save.FileSize, "Sha256": d.SHA256(), "PartyIds": d.Party(), "Money": d.Money(), "Noponstones": d.Noponstones(), "Characters": chars})
	case "arts":
		need(a, 3)
		items := []object{}
		for _, x := range d.Character(num(a[2])).Arts() {
			items = append(items, object{"Id": x.Def.Id, "Name": x.Name(), "Level": x.Level(), "Learned": x.Learned(), "IsTalent": x.Def.IsTalent, "CanEdit": x.CanEdit(), "CanLearn": x.CanLearn(), "IsLevelLearned": x.Def.IsLevelLearned(), "RequiresEvent": !x.Def.IsLevelLearned() && !x.Def.IsTalent, "LearnLevel": x.Def.LearnLevel, "MaximumLevel": x.MaximumLevel(), "UnlockedMaximum": x.UnlockedMaximum()})
		}
		emit(items)
	case "skills":
		need(a, 3)
		items := []object{}
		for _, t := range d.Character(num(a[2])).SkillTrees() {
			items = append(items, object{"Index": t.Def.Index, "Name": t.Def.Name, "IsUnlocked": t.Unlocked(), "CanEdit": t.CanEdit(), "CanUnlock": d.MainSupported() && !t.Unlocked(), "LearnedCount": t.LearnedCount(), "Progress": t.Progress(), "MaximumProgress": t.MaximumProgress(), "Skills": t.Def.Skills})
		}
		emit(items)
	case "skill-links":
		need(a, 3)
		c := d.Character(num(a[2]))
		items := []object{}
		for _, l := range c.SkillLinks() {
			var skill any
			if s, ok := save.SkillDefinitions[l.SkillID()]; ok {
				skill = s
			}
			items = append(items, object{"SourceCharacterId": l.SourceID, "Index": l.Index, "Shape": save.Shapes[l.Shape()], "HighestUnlockedSlot": l.HighestSlot(), "IsUnlocked": l.Unlocked(), "SkillId": l.SkillID(), "Skill": skill, "CanEdit": l.CanEdit(), "Choices": l.Choices()})
		}
		cost, ok := c.LinkedCost()
		var value any
		if ok {
			value = cost
		}
		emit(object{"Supported": d.MainSupported(), "AffinityCoins": c.AffinityCoins(), "LinkedSkillCoinCost": value, "Slots": items})
	case "gems":
		need(a, 2)
		items := []object{}
		for _, g := range d.Gems() {
			if g.IsCylinder() {
				continue
			}
			var lo, hi any
			valid := false
			if rule := g.Definition(); rule != nil {
				lo = rule.Minimum
				hi = rule.Maximum
				valid = g.Strength() >= rule.Minimum && g.Strength() <= rule.Maximum && g.Chance() == rule.Chance
			}
			items = append(items, object{"Index": g.Index, "Name": save.GemName(g.EffectID()), "EffectId": g.EffectID(), "Rank": g.Rank(), "Strength": g.Strength(), "Chance": g.Chance(), "CanEdit": g.CanEdit(), "HasValidValue": valid, "Minimum": lo, "Maximum": hi, "Label": g.Label()})
		}
		emit(items)
	case "inventory":
		need(a, 3)
		k := kind(a[2])
		items, catalog := []object{}, []save.InventoryDefinition{}
		for _, x := range d.Inventory(k) {
			items = append(items, object{"Index": x.Index, "ItemId": x.ItemID(), "Name": x.Name(), "Quantity": x.Quantity(), "Favorite": x.Favorite(), "CanEdit": x.CanEdit()})
		}
		for _, def := range save.InventoryDefinitions {
			if def.Kind == k {
				catalog = append(catalog, def)
			}
		}
		emit(object{"Kind": save.KindNames[k], "Capacity": save.Capacity, "Items": items, "Catalog": catalog})
	case "equipment-inventory":
		need(a, 3)
		s := slot(a[2])
		items := []object{}
		for _, x := range d.EquipmentInventory(s) {
			items = append(items, object{"Index": x.Index, "ItemId": x.ItemID(), "Name": x.Name(), "GemSlotCount": x.Sockets(), "ArmorClass": x.ArmorClass(), "Favorite": x.Favorite(), "IsValid": x.Valid(), "FixedGemIds": x.FixedGems(), "EquippedBy": x.EquippedBy(), "CanDelete": x.CanDelete(), "CanEditFavorite": x.CanEdit()})
		}
		emit(object{"Slot": save.SlotNames[s], "Capacity": save.Capacity, "Count": len(items), "Supported": d.Campaign() != "Unknown" && d.Version() == 7, "Items": items, "Catalog": d.CreatableEquipment(s)})
	case "equipment":
		need(a, 3)
		items := []object{}
		c := d.Character(num(a[2]))
		for s := 0; s <= 5; s++ {
			x := c.Equipment(s)
			sockets, available := []object{}, []object{}
			for _, v := range x.AvailableItems() {
				available = append(available, object{"Index": v.Index, "ItemId": v.ItemID(), "Name": v.Name(), "GemSlotCount": v.Sockets()})
			}
			for _, v := range x.Sockets() {
				gems := []object{}
				if v.CanEdit() {
					for _, g := range v.AvailableGems() {
						gems = append(gems, object{"Index": g.Index, "Name": save.GemName(g.EffectID()), "Rank": g.Rank(), "Value": g.Value(), "Strength": g.Strength(), "Chance": g.Chance(), "Label": g.Label()})
					}
				}
				var gi, fi any
				if v.GemIndex() >= 0 {
					gi = v.GemIndex()
				}
				if v.FixedID() != 0 {
					fi = v.FixedID()
				}
				sockets = append(sockets, object{"Index": v.Index, "Name": v.Name(), "GemIndex": gi, "FixedItemId": fi, "CanEdit": v.CanEdit(), "AvailableGems": gems})
			}
			var id any
			n := 0
			if x.Exists() {
				id = x.Item().ItemID()
				n = x.Item().Sockets()
			}
			items = append(items, object{"Slot": save.SlotNames[s], "Index": x.Index(), "ItemId": id, "Name": x.Name(), "Exists": x.Exists(), "CanEdit": x.CanEdit(), "CanSwitch": x.CanSwitch(), "GemSlotCount": n, "AvailableItems": available, "Sockets": sockets})
		}
		emit(items)
	case "affinities":
		need(a, 2)
		items := []object{}
		for _, p := range d.Affinities() {
			items = append(items, object{"Index": p.Index, "FirstCharacterId": p.FirstID, "SecondCharacterId": p.SecondID, "Points": p.Points(), "FirstUnlockedSlots": p.Slots(p.FirstID, p.SecondID), "SecondUnlockedSlots": p.Slots(p.SecondID, p.FirstID), "CanEdit": d.MainSupported(), "IsMaximum": p.Points() == 5000 && p.Slots(p.FirstID, p.SecondID) == 5 && p.Slots(p.SecondID, p.FirstID) == 5})
		}
		emit(object{"Supported": d.MainSupported(), "MaximumPoints": 5000, "Pairs": items})
	case "region-affinities":
		need(a, 2)
		items := []object{}
		for _, r := range d.Regions() {
			items = append(items, object{"Id": r.ID, "Name": save.RegionNames[r.ID-1], "Points": r.Points(), "Stars": r.Stars(), "CanEdit": d.MainSupported(), "IsMaximum": r.Points() == 10000})
		}
		emit(object{"Supported": d.MainSupported(), "MaximumPoints": 10000, "FiveStarPoints": 8000, "Regions": items})
	case "achievements":
		need(a, 2)
		items := []object{}
		completed := 0
		for _, x := range d.Achievements() {
			var progress, meets any
			if x.Def.HasCounter() {
				progress = x.Progress()
				meets = x.Def.ConditionType == 3 && x.Progress() == x.Def.Required || x.Def.ConditionType == 4 && x.Progress() >= x.Def.Required
			}
			if x.Completed() {
				completed++
			}
			category := "Trial"
			if x.Def.Category == 1 {
				category = "Record"
			}
			items = append(items, object{"Id": x.Def.Id, "Name": x.Def.Name, "Category": category, "Condition": x.Def.Condition, "RewardExperience": x.Def.RewardExperience, "Required": x.Def.Required, "Progress": progress, "Completed": x.Completed(), "CanUnlock": d.MainSupported() && !x.Completed(), "CounterMeetsRequirement": meets, "HasUnmetCompletedCounter": x.Unmet(), "CanRepairCounter": d.MainSupported() && x.Unmet()})
		}
		emit(object{"Supported": d.MainSupported(), "Completed": completed, "Total": len(items), "Items": items})
	case "colony6":
		need(a, 2)
		v := d.Colony6()
		if v == nil {
			emit(nil)
		} else {
			emit(object{"Development": v.Development, "Population": v.Population, "Housing": v.Housing, "Commerce": v.Commerce, "Nature": v.Nature, "Special": v.Special, "OverallLevel": v.OverallLevel, "CanMaximize": v.CanMaximize, "UnavailableReason": v.UnavailableReason, "IsMaximum": v.Housing == 5 && v.Commerce == 5 && v.Nature == 5 && v.Special == 5 && v.OverallLevel == 5})
		}
	case "collectopaedia":
		need(a, 2)
		pages := []object{}
		maps := []int{}
		entries := map[int][]object{}
		counts := map[int]int{}
		names := map[int]string{}
		registered := 0
		for _, x := range d.Collection() {
			id := x.Def.MapId
			if entries[id] == nil {
				maps = append(maps, id)
			}
			entries[id] = append(entries[id], object{"Id": x.Def.Id, "ItemId": x.Def.ItemId, "Name": x.Def.Name(), "Category": x.Def.Category, "IsRegistered": x.Registered()})
			names[id] = x.Def.MapName
			if x.Registered() {
				counts[id]++
				registered++
			}
		}
		for _, id := range maps {
			pages = append(pages, object{"MapId": id, "MapName": names[id], "Count": len(entries[id]), "RegisteredCount": counts[id], "Entries": entries[id]})
		}
		emit(object{"Supported": d.CollectionSupported(), "Campaign": d.Campaign(), "Count": len(d.Collection()), "RegisteredCount": registered, "Pages": pages})
	case "collectopaedia-plan":
		ids := selection(d, a[2:])
		p := d.PlanCollection(ids)
		emit(object{"PreviewOnly": true, "NewEntryCount": len(p.NewEntries), "NewEntries": p.NewEntries, "Rewards": p.Rewards, "NewAchievementIds": p.NewAchievementIds})
	default:
		read = false
	}
	if read {
		return 0
	}
	if len(a) < 3 {
		fail("Specify an output file.")
	}
	err := d.Edit(func(s *save.Document) {
		b := a[3:]
		switch a[0] {
		case "copy":
			need(b, 0)
			src, _ := filepath.Abs(a[1])
			dst, _ := filepath.Abs(a[2])
			if src == dst {
				fail("Copy requires a different destination.")
			}
		case "resources":
			m := options(b, "--money", "--noponstones")
			s.SetResources(ptr(m, "--money"), ptr(m, "--noponstones"))
		case "character", "progression":
			if len(b) < 1 {
				fail("Specify character ID.")
			}
			c := s.Character(num(b[0]))
			if a[0] == "character" {
				m := options(b[1:], "--ap", "--coins", "--reserve-exp")
				c.SetResources(ptr(m, "--ap"), ptr(m, "--coins"), ptr(m, "--reserve-exp"))
			} else {
				m := options(b[1:], "--level", "--exp")
				c.SetProgression(ptr(m, "--level"), ptr(m, "--exp"))
			}
		case "max-ap":
			need(b, 0)
			for _, c := range s.Characters() {
				v := 99999999
				c.SetResources(&v, nil, nil)
			}
		case "art":
			if len(b) < 2 {
				fail("Specify character and art.")
			}
			m := options(b[2:], "--level")
			s.Character(num(b[0])).Art(num(b[1])).SetLevel(m["--level"])
		case "max-art", "learn-art":
			need(b, 2)
			x := s.Character(num(b[0])).Art(num(b[1]))
			if a[0] == "learn-art" {
				x.Learn(1)
			} else {
				x.SetLevel(x.MaximumLevel())
			}
		case "max-arts", "learn-max-arts":
			need(b, 1)
			s.Character(num(b[0])).MaxArts(a[0] == "learn-max-arts")
		case "learn-max-all-arts":
			need(b, 0)
			if len(s.Characters()) != len(s.Party()) {
				fail("Unsupported party.")
			}
			for _, c := range s.Characters() {
				c.MaxArts(true)
			}
		case "skill-tree":
			need(b, 4)
			m := options(b[2:], "--learned", "--sp")
			t := s.Character(num(b[0])).SkillTree(num(b[1]))
			if p := ptr(m, "--learned"); p != nil {
				t.SetLearnedCount(*p)
			} else {
				t.SetProgress(m["--sp"])
			}
		case "max-skill-tree", "unlock-skill-tree":
			need(b, 2)
			t := s.Character(num(b[0])).SkillTree(num(b[1]))
			if a[0] == "max-skill-tree" {
				t.SetLearnedCount(5)
			} else {
				t.Unlock()
			}
		case "max-skills", "unlock-max-skills":
			need(b, 1)
			s.Character(num(b[0])).MaxSkills(a[0] == "unlock-max-skills")
		case "max-all-skills", "unlock-max-all-skills":
			need(b, 0)
			if !s.MainSupported() {
				fail("Unsupported campaign.")
			}
			for _, id := range s.Party() {
				if id > 8 {
					fail("Unsupported party.")
				}
			}
			for _, c := range s.Characters() {
				c.MaxSkills(a[0] == "unlock-max-all-skills")
			}
		case "skill-link":
			need(b, 4)
			id := 0
			if b[3] != "none" {
				id = num(b[3])
			}
			s.Character(num(b[0])).SkillLink(num(b[1]), num(b[2])).SetSkill(id)
		case "equip":
			need(b, 3)
			s.Character(num(b[0])).Equipment(slot(b[1])).Equip(num(b[2]))
		case "equipment-gem":
			need(b, 4)
			x := s.Character(num(b[0])).Equipment(slot(b[1]))
			index := num(b[2])
			if index < 1 || index > len(x.Sockets()) {
				fail("Invalid socket.")
			}
			g := -1
			if b[3] != "none" {
				g = num(b[3])
			}
			indices := []int{}
			for _, socket := range x.Sockets() {
				n := socket.GemIndex()
				if socket.Index == index {
					n = g
				}
				indices = append(indices, n)
			}
			x.SetGems(indices)
		case "gem":
			if len(b) < 1 {
				fail("Specify gem index.")
			}
			m := options(b[1:], "--effect", "--rank", "--value")
			g := s.Gem(num(b[0]))
			g.Set(pick(m, "--effect", g.EffectID()), pick(m, "--rank", g.Rank()), pick(m, "--value", g.Strength()))
		case "max-gem", "delete-gem":
			need(b, 1)
			g := s.Gem(num(b[0]))
			if a[0] == "max-gem" {
				g.Maximize()
			} else {
				g.Delete()
			}
		case "add-gem":
			m := options(b, "--effect", "--rank", "--value")
			if len(m) != 3 {
				fail("Specify effect, rank and value.")
			}
			s.AddGem(m["--effect"], m["--rank"], m["--value"])
		case "add-item":
			need(b, 3)
			m := options(b[1:], "--quantity")
			s.AddItem(num(b[0]), m["--quantity"])
		case "item":
			need(b, 4)
			m := options(b[2:], "--quantity")
			s.Item(kind(b[0]), num(b[1])).SetQuantity(m["--quantity"])
		case "delete-item":
			need(b, 2)
			s.Item(kind(b[0]), num(b[1])).Delete()
		case "max-items":
			need(b, 1)
			s.MaxItems(kind(b[0]))
		case "add-equipment":
			need(b, 1)
			s.AddEquipment(num(b[0]))
		case "fill-missing-equipment":
			need(b, 2)
			s.FillEquipment(slot(b[0]), num(b[1]))
		case "equipment-favorite":
			need(b, 3)
			v, e := strconv.ParseBool(b[2])
			if e != nil {
				fail("Choose true or false.")
			}
			s.OwnedEquipment(slot(b[0]), num(b[1])).SetFavorite(v)
		case "delete-equipment":
			need(b, 2)
			s.OwnedEquipment(slot(b[0]), num(b[1])).Delete()
		case "affinity":
			need(b, 4)
			m := options(b[2:], "--points")
			s.Affinity(num(b[0]), num(b[1])).SetPoints(m["--points"])
		case "max-affinity":
			need(b, 2)
			s.Affinity(num(b[0]), num(b[1])).SetPoints(5000)
		case "max-all-affinity":
			need(b, 0)
			if !s.MainSupported() {
				fail("Unsupported campaign.")
			}
			for _, p := range s.Affinities() {
				p.SetPoints(5000)
			}
		case "region-affinity":
			need(b, 3)
			m := options(b[1:], "--points", "--stars")
			r := s.Region(num(b[0]))
			if p := ptr(m, "--points"); p != nil {
				r.SetPoints(*p)
			} else {
				r.SetStars(m["--stars"])
			}
		case "max-region-affinity":
			need(b, 1)
			s.Region(num(b[0])).SetPoints(10000)
		case "max-all-region-affinity":
			need(b, 0)
			if !s.MainSupported() {
				fail("Unsupported campaign.")
			}
			for _, r := range s.Regions() {
				r.SetPoints(10000)
			}
		case "unlock-achievement", "repair-achievement-counter":
			need(b, 1)
			x := s.Achievement(num(b[0]))
			if a[0] == "unlock-achievement" {
				x.Unlock()
			} else {
				x.Repair()
			}
		case "unlock-all-achievements", "repair-all-achievement-counters":
			need(b, 0)
			if !s.MainSupported() {
				fail("Unsupported campaign.")
			}
			for _, x := range s.Achievements() {
				if a[0] == "unlock-all-achievements" {
					x.Unlock()
				} else {
					x.Repair()
				}
			}
		case "max-colony6":
			need(b, 0)
			s.MaxColony6()
		case "complete-collectopaedia":
			s.CompleteCollection(selection(s, b))
		default:
			fail("Unknown command; use --help.")
		}
	})
	if err != nil {
		fail(err.Error())
	}
	if err = session.Save(a[2]); err != nil {
		fail(err.Error())
	}
	return 0
}
func selection(d *save.Document, b []string) []int {
	if len(b) == 0 {
		return d.CollectionIDs(0, 0)
	}
	need(b, 2)
	if num(b[1]) == 0 {
		fail("Choose a positive entry or page ID.")
	}
	switch b[0] {
	case "--page":
		return d.CollectionIDs(num(b[1]), 0)
	case "--entry":
		return d.CollectionIDs(0, num(b[1]))
	}
	fail("Choose --page or --entry.")
	return nil
}

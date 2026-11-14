package save

import "math/rand/v2"

type CollectionDefinition struct {
	Id, ItemId, MapId int
	MapName           string
	Category          int
}

func (c CollectionDefinition) Campaign() string {
	if c.Id <= 300 {
		return "MainStory"
	}
	return "FutureConnected"
}
func (c CollectionDefinition) Name() string { return InventoryDefinitionByID(c.ItemId).Name }

var CollectionDefinitions = func() []CollectionDefinition {
	out := []CollectionDefinition{}
	for _, r := range rows("collectopaedia") {
		out = append(out, CollectionDefinition{number(r[0]), number(r[1]), number(r[2]), r[3], number(r[4])})
	}
	return out
}()

type CollectionReward struct {
	Id, ItemId                              int
	Name                                    string
	ItemType, EffectId, Rank, FixedStrength int
}

var CollectionRewards = func() []CollectionReward {
	out := []CollectionReward{}
	for _, r := range rows("collectopaedia-rewards") {
		out = append(out, CollectionReward{number(r[0]), number(r[1]), r[2], number(r[3]), number(r[4]), number(r[5]), number(r[6])})
	}
	return out
}()

type CollectionEntry struct {
	d   *Document
	Def CollectionDefinition
}

func (e CollectionEntry) Registered() bool    { return e.d.bit(0x148d6c, 0x19e9+e.Def.Id) }
func (d *Document) CollectionSupported() bool { return d.Campaign() != "Unknown" && d.Version() == 7 }
func (d *Document) Collection() []CollectionEntry {
	out := []CollectionEntry{}
	if d.CollectionSupported() {
		for _, def := range CollectionDefinitions {
			if def.Campaign() == d.Campaign() {
				out = append(out, CollectionEntry{d, def})
			}
		}
	}
	return out
}
func (d *Document) CollectionIDs(page, entry int) []int {
	require(d.CollectionSupported(), "Collectopaedia requires an identified format-7 campaign.")
	ids := []int{}
	for _, e := range d.Collection() {
		if (page == 0 || e.Def.MapId == page) && (entry == 0 || e.Def.Id == entry) {
			ids = append(ids, e.Def.Id)
		}
	}
	require(len(ids) > 0, "Choose Collectopaedia entries from this campaign.")
	return ids
}

type CollectionPlan struct {
	NewEntries        []CollectionDefinition
	Rewards           []CollectionReward
	NewAchievementIds []int
}

func (d *Document) PlanCollection(ids []int) CollectionPlan {
	require(d.CollectionSupported(), "Collectopaedia requires an identified format-7 campaign.")
	plan := CollectionPlan{[]CollectionDefinition{}, []CollectionReward{}, []int{}}
	before, after := map[int]bool{}, map[int]bool{}
	selected := map[int]bool{}
	for _, id := range ids {
		selected[id] = true
	}
	pages := map[int][]CollectionDefinition{}
	for _, e := range d.Collection() {
		if e.Registered() {
			before[e.Def.Id] = true
			after[e.Def.Id] = true
		}
		if selected[e.Def.Id] && !e.Registered() {
			plan.NewEntries = append(plan.NewEntries, e.Def)
			after[e.Def.Id] = true
		}
		delete(selected, e.Def.Id)
		pages[e.Def.MapId] = append(pages[e.Def.MapId], e.Def)
	}
	require(len(selected) == 0, "Choose Collectopaedia entries from this campaign.")
	first := []int{1, 8, 15, 22, 26, 32, 39, 46, 1, 50, 54, 60, 1, 64, 71, 78, 84, 88, 1, 95, 1, 102, 109, 116, 120, 1, 1, 124, 129}
	allDone, anyDone := true, false
	for _, page := range sortedKeys(pages) {
		defs := pages[page]
		groups := map[int][]CollectionDefinition{}
		for _, e := range defs {
			groups[e.Category] = append(groups[e.Category], e)
		}
		complete := func(es []CollectionDefinition, m map[int]bool) bool {
			for _, e := range es {
				if !m[e.Id] {
					return false
				}
			}
			return true
		}
		for i, cat := range sortedKeys(groups) {
			if !complete(groups[cat], before) && complete(groups[cat], after) {
				plan.Rewards = append(plan.Rewards, CollectionRewards[first[page-2]+i])
			}
		}
		done := complete(defs, after)
		if !complete(defs, before) && done {
			plan.Rewards = append(plan.Rewards, CollectionRewards[first[page-2]-1])
		}
		allDone = allDone && done
		anyDone = anyDone || done
	}
	if d.Campaign() == "MainStory" && len(plan.NewEntries) > 0 {
		for _, id := range []int{132, 133, 134} {
			if (id == 132 || id == 133 && anyDone || id == 134 && allDone) && !d.Achievement(id).Completed() {
				plan.NewAchievementIds = append(plan.NewAchievementIds, id)
			}
		}
	}
	return plan
}
func (d *Document) CompleteCollection(ids []int) CollectionPlan {
	plan := d.PlanCollection(ids)
	if len(plan.NewEntries) == 0 {
		return plan
	}
	staged := &Document{d.Bytes()}
	defs := []EquipmentDefinition{}
	for _, r := range plan.Rewards {
		if r.ItemType != 3 {
			defs = append(defs, *EquipmentDefinitionByID(r.ItemId))
		}
	}
	staged.createEquipmentBatch(defs)
	for _, r := range plan.Rewards {
		if r.ItemType == 3 {
			rule := GemDefinitionFor(r.EffectId, r.Rank)
			strength := r.FixedStrength
			if strength == 0 {
				strength = rule.Minimum + rand.IntN(rule.Maximum-rule.Minimum+1)
			}
			g := staged.AddGem(r.EffectId, r.Rank, strength)
			staged.w16(g.offset()+4, r.ItemId)
		}
	}
	for _, e := range plan.NewEntries {
		staged.setBit(0x148d6c, 0x19e9+e.Id)
	}
	for _, id := range plan.NewAchievementIds {
		staged.Achievement(id).Unlock()
	}
	d.data = staged.data
	return plan
}

package save

import "sort"

type SkillTree struct {
	d   *Document
	Def SkillTreeDefinition
}

func (c Character) SkillTrees() []SkillTree {
	out := []SkillTree{}
	for _, t := range SkillTrees {
		if t.CharacterId == c.ID {
			out = append(out, SkillTree{c.d, t})
		}
	}
	return out
}
func (c Character) SkillTree(index int) SkillTree {
	for _, t := range c.SkillTrees() {
		if t.Def.Index == index {
			return t
		}
	}
	panic(editError("This skill tree does not belong to the selected character."))
}
func (t SkillTree) offset() int       { return CharacterOffset + (t.Def.CharacterId-1)*CharacterSize }
func (t SkillTree) Progress() int     { return t.d.u32(t.offset() + 0x7c + (t.Def.Index-1)*4) }
func (t SkillTree) LearnedCount() int { return t.d.u32(t.offset() + 0x90 + (t.Def.Index-1)*4) }
func (t SkillTree) MinimumCount() int {
	n := 0
	for _, s := range t.Def.Skills {
		if s.RequiredSP != 0 {
			break
		}
		n++
	}
	return n
}
func (t SkillTree) Unlocked() bool {
	return t.Def.UnlockFlag() == 0 || t.d.bit(0x50, 0x2cdd+t.Def.UnlockFlag())
}
func (t SkillTree) CanEdit() bool { return t.d.MainSupported() && t.Unlocked() }
func (t SkillTree) MaximumProgress() int {
	n := t.LearnedCount()
	if n >= 0 && n < 5 && t.Def.Skills[n].RequiredSP > 0 {
		return t.Def.Skills[n].RequiredSP - 1
	}
	return 0
}
func (t SkillTree) SetProgress(v int) {
	require(t.CanEdit() && t.LearnedCount() >= t.MinimumCount() && t.LearnedCount() < 5, "Only an incomplete, unlocked skill tree can receive SP.")
	require(v >= 0 && v <= t.MaximumProgress(), "SP exceeds the next skill requirement.")
	t.d.w32(t.offset()+0x7c+(t.Def.Index-1)*4, v)
}
func (t SkillTree) SetLearnedCount(v int) {
	require(t.CanEdit(), "Only unlocked skill trees can be edited.")
	require(v >= t.MinimumCount() && v <= 5, "Invalid learned skill count.")
	t.d.w32(t.offset()+0x90+(t.Def.Index-1)*4, v)
	t.d.w32(t.offset()+0x7c+(t.Def.Index-1)*4, 0)
}
func (t SkillTree) Unlock() {
	require(t.d.MainSupported(), "Skill branches require main-story format 7.")
	if f := t.Def.UnlockFlag(); f != 0 {
		t.d.setBit(0x50, 0x2cdd+f)
	}
}
func (c Character) MaxSkills(unlock bool) {
	require(c.d.MainSupported() && c.ID >= 1 && c.ID <= 8, "Skills require a supported main-story character.")
	for _, t := range c.SkillTrees() {
		if unlock {
			t.Unlock()
		}
		if t.CanEdit() {
			t.SetLearnedCount(5)
		}
	}
}
func (d *Document) sourceAvailable(id int) bool {
	if id < 1 || id > 8 {
		return false
	}
	if d.u8(0x5eb)&0x10 != 0 {
		return id != 3
	}
	story := d.u16(0xdf0)
	switch id {
	case 1, 2:
		return true
	case 3:
		return story >= 11 && story < 42
	default:
		return story >= map[int]int{4: 100, 5: 69, 6: 137, 7: 128, 8: 273}[id]
	}
}
func alternate(a, b int) bool { return (a == 3 || a == 8) && (b == 3 || b == 8) }

type SkillLink struct {
	d                            *Document
	CharacterID, SourceID, Index int
}

func (c Character) SkillLinks() []SkillLink {
	out := []SkillLink{}
	if c.d.Campaign() != "MainStory" || c.ID < 1 || c.ID > 8 {
		return out
	}
	ids := append([]int(nil), c.d.Party()...)
	sort.Ints(ids)
	for _, id := range ids {
		if c.d.sourceAvailable(id) && id != c.ID && !alternate(id, c.ID) {
			for i := 1; i <= 5; i++ {
				out = append(out, SkillLink{c.d, c.ID, id, i})
			}
		}
	}
	return out
}
func (c Character) SkillLink(source, index int) SkillLink {
	for _, l := range c.SkillLinks() {
		if l.SourceID == source && l.Index == index {
			return l
		}
	}
	panic(editError("This source or skill-link slot is unavailable."))
}
func (l SkillLink) offset() int {
	return CharacterOffset + (l.CharacterID-1)*CharacterSize + 0xc4 + (l.SourceID-1)*5 + l.Index - 1
}
func (l SkillLink) SkillID() int { return l.d.u8(l.offset()) }
func (l SkillLink) Shape() int   { return LinkShapes[[2]int{l.CharacterID, l.SourceID}][l.Index-1] }
func (l SkillLink) HighestSlot() int {
	return l.d.u32(CharacterOffset + (l.CharacterID-1)*CharacterSize + 0xa4 + (l.SourceID-1)*4)
}
func (l SkillLink) Unlocked() bool { return l.HighestSlot() <= 4 && l.Index-1 <= l.HighestSlot() }
func (l SkillLink) CanEdit() bool  { return l.d.MainSupported() && l.Unlocked() }
func (c Character) LinkedCost() (int, bool) {
	if c.d.Campaign() != "MainStory" || c.ID < 1 || c.ID > 8 {
		return 0, false
	}
	cost := 0
	for src := 1; src <= 8; src++ {
		if !c.d.sourceAvailable(src) || src == c.ID || alternate(src, c.ID) {
			continue
		}
		for i := 1; i <= 5; i++ {
			l := SkillLink{c.d, c.ID, src, i}
			if l.HighestSlot() > 4 {
				return 0, false
			}
			if !l.Unlocked() || l.SkillID() == 0 {
				continue
			}
			def, ok := SkillDefinitions[l.SkillID()]
			if !ok {
				return 0, false
			}
			cost += def.AffinityCoins
		}
	}
	return cost, true
}
func (l SkillLink) validateSkill(id int) {
	require(l.CanEdit(), "Only unlocked skill links can be edited.")
	if id == 0 {
		return
	}
	s, ok := SkillDefinitions[id]
	require(ok && s.CharacterId() == l.SourceID && s.Shape == l.Shape(), "The skill must match the source and slot shape.")
	t := l.d.Character(l.SourceID).SkillTree(s.TreeIndex())
	require(t.Unlocked() && t.LearnedCount() >= 0 && t.LearnedCount() <= 5 && t.LearnedCount() > s.NodeIndex(), "The source must have learned this skill.")
	c := l.d.Character(l.CharacterID)
	for _, other := range c.SkillLinks() {
		require(other.SourceID != l.SourceID || other.Index == l.Index || other.SkillID() != id, "Duplicate skill link.")
	}
	used, valid := c.LinkedCost()
	require(valid, "Remove unrecognized skill links first.")
	prev := SkillDefinitions[l.SkillID()].AffinityCoins
	require(used-prev+s.AffinityCoins <= max(c.AffinityCoins(), used), "Not enough Affinity Coins.")
}
func (l SkillLink) SetSkill(id int) { l.validateSkill(id); l.d.w8(l.offset(), id) }
func (l SkillLink) CanSet(id int) (ok bool) {
	defer func() {
		if p := recover(); p != nil {
			if _, expected := p.(editError); expected {
				ok = false
			} else {
				panic(p)
			}
		}
	}()
	l.validateSkill(id)
	return true
}
func (l SkillLink) Choices() []SkillDefinition {
	out := []SkillDefinition{}
	if !l.CanEdit() {
		return out
	}
	for _, t := range l.d.Character(l.SourceID).SkillTrees() {
		if !t.Unlocked() || t.LearnedCount() > 5 {
			continue
		}
		for i, s := range t.Def.Skills {
			if i < t.LearnedCount() && s.Shape == l.Shape() && l.CanSet(s.Id) {
				out = append(out, s)
			}
		}
	}
	return out
}

package save

import "fmt"

type Equipment struct {
	d                 *Document
	CharacterID, Slot int
}

func (c Character) Equipment(slot int) Equipment {
	equipmentStart(slot)
	return Equipment{c.d, c.ID, slot}
}
func (e Equipment) reference() int {
	return CharacterOffset + (e.CharacterID-1)*CharacterSize + characterField(e.Slot)
}
func (e Equipment) Index() int { return e.d.u16(e.reference()) }
func (e Equipment) Exists() bool {
	if e.Index() >= Capacity || e.d.u16(e.reference()+2) != equipmentType(e.Slot) {
		return false
	}
	x := e.d.OwnedEquipment(e.Slot, e.Index())
	p := x.offset()
	return e.d.u8(p+0x10) == 1 && e.d.u16(p) == e.Index() && e.d.u16(p+2) == equipmentType(e.Slot) && e.d.u16(p+6) == equipmentType(e.Slot) && e.d.u16(p+8) == 1
}
func (e Equipment) Item() OwnedEquipment {
	require(e.Exists(), "Invalid equipped item reference.")
	return e.d.OwnedEquipment(e.Slot, e.Index())
}
func (e Equipment) Name() string {
	if e.Exists() {
		return e.Item().Name()
	}
	return fmt.Sprintf("Unknown reference (%d)", e.Index())
}
func (e Equipment) CanEdit() bool {
	if !e.Exists() || e.Item().Sockets() > 3 || e.d.Campaign() == "Unknown" {
		return false
	}
	n := 0
	for _, c := range e.d.Characters() {
		x := c.Equipment(e.Slot)
		if x.Index() == e.Index() && x.Exists() {
			n++
		}
	}
	return n == 1
}
func (d *Document) hasSkill(character, skill int) bool {
	def, ok := SkillDefinitions[skill]
	if !ok || !contains(d.Party(), def.CharacterId()) {
		return false
	}
	t := d.Character(def.CharacterId()).SkillTree(def.TreeIndex())
	if !t.Unlocked() || t.LearnedCount() < def.NodeIndex()+1 || t.LearnedCount() > 5 {
		return false
	}
	if def.CharacterId() == character {
		return true
	}
	p := CharacterOffset + (character-1)*CharacterSize + 0xc4 + (def.CharacterId()-1)*5
	for i := 0; i < 5; i++ {
		if d.u8(p+i) == skill {
			return true
		}
	}
	return false
}
func (x OwnedEquipment) CanUse(id int) bool {
	if !x.Valid() || x.d.Campaign() == "Unknown" {
		return false
	}
	def := EquipmentDefinitionByID(x.ItemID())
	if !contains(def.Characters, id) {
		return false
	}
	if x.Slot == 0 {
		return def.WeaponFlags == 0 || id == 8 && def.WeaponFlags == 3
	}
	ac := def.ArmorClass
	if ac <= 1 {
		return true
	}
	if id == 8 {
		return ac >= 4 && ac <= 13
	}
	if x.d.Campaign() == "FutureConnected" {
		return ac == 2 || ac == 3
	}
	if id == 3 && ac == 3 {
		return false
	}
	skills := map[int][]int{2: {1, 26, 51, 76, 101}, 3: {31, 98, 115, 141}}[ac]
	for _, s := range skills {
		if x.d.hasSkill(id, s) {
			return true
		}
	}
	return false
}
func (e Equipment) CanSwitch() bool {
	if !e.CanEdit() {
		return false
	}
	n := 0
	for _, c := range e.d.Characters() {
		if c.Equipment(e.Slot).Index() == e.Index() {
			n++
		}
	}
	return n == 1 && e.Item().CanUse(e.CharacterID)
}
func (e Equipment) AvailableItems() []OwnedEquipment {
	out := []OwnedEquipment{}
	if !e.CanSwitch() {
		return out
	}
	occupied := map[int]bool{}
	for _, c := range e.d.Characters() {
		if c.ID != e.CharacterID {
			occupied[c.Equipment(e.Slot).Index()] = true
		}
	}
	for _, x := range e.d.EquipmentInventory(e.Slot) {
		if !occupied[x.Index] && x.CanUse(e.CharacterID) {
			out = append(out, x)
		}
	}
	return out
}
func (e Equipment) Equip(index int) {
	require(e.CanSwitch(), "This equipped item cannot be switched.")
	ok := false
	for _, x := range e.AvailableItems() {
		if x.Index == index {
			ok = true
		}
	}
	require(ok, "Choose an owned, compatible item not equipped by another character.")
	e.d.w32(e.reference(), index|equipmentType(e.Slot)<<16)
}

type GemSocket struct {
	e     Equipment
	Index int
}

func (e Equipment) Sockets() []GemSocket {
	out := []GemSocket{}
	if e.Exists() {
		for i := 1; i <= min(e.Item().Sockets(), 3); i++ {
			out = append(out, GemSocket{e, i})
		}
	}
	return out
}
func (s GemSocket) offset() int { return s.e.Item().offset() + 0x18 + (s.Index-1)*8 }
func (s GemSocket) GemIndex() int {
	if s.e.d.u16(s.offset()+2) == 3 {
		return s.e.d.u16(s.offset())
	}
	return -1
}
func (s GemSocket) FixedID() int {
	if s.e.d.u32(s.offset()+4) != 0 {
		return s.e.d.u16(s.offset() + 4)
	}
	return 0
}
func (s GemSocket) Empty() bool { return s.e.d.u32(s.offset()) == 0 && s.FixedID() == 0 }
func (s GemSocket) CanEdit() bool {
	i := s.GemIndex()
	return s.e.CanEdit() && s.FixedID() == 0 && (s.Empty() || i >= 0 && i < Capacity && s.e.d.Gem(i).CanEquip())
}
func (s GemSocket) Name() string {
	if id := s.FixedID(); id > 0 {
		return ItemName(id)
	}
	i := s.GemIndex()
	if i >= 0 && i < Capacity && s.e.d.Gem(i).Exists() {
		return s.e.d.Gem(i).Label()
	}
	if s.Empty() {
		return "None"
	}
	return "Unknown gem"
}
func (s GemSocket) AvailableGems() []Gem {
	out := []Gem{}
	used := s.e.d.usedGems(nil)
	for _, g := range s.e.d.Gems() {
		if g.CanEquip() && (g.Index == s.GemIndex() || g.CanFit(s.e.Slot) && !used[g.Index]) {
			out = append(out, g)
		}
	}
	return out
}
func (e Equipment) SetGems(indices []int) {
	require(e.CanEdit() && len(indices) == e.Item().Sockets(), "Only valid equipment and its existing sockets can be edited.")
	changed := false
	selected := map[int]bool{}
	for i, s := range e.Sockets() {
		index := indices[i]
		if index >= 0 {
			require(!selected[index], "One gem cannot occupy multiple sockets.")
			selected[index] = true
		}
		if index == s.GemIndex() {
			continue
		}
		changed = true
		require(s.CanEdit(), "Fixed or unrecognized gems cannot be replaced.")
		if index >= 0 {
			require(index < Capacity && e.d.Gem(index).CanFit(e.Slot) && !e.d.usedGems(&e)[index], "Choose an unused compatible gem.")
		}
	}
	if changed {
		for i, s := range e.Sockets() {
			if indices[i] == s.GemIndex() {
				continue
			}
			v := 0
			if indices[i] >= 0 {
				v = indices[i] | 3<<16
			}
			e.d.w32(s.offset(), v)
		}
	}
}
func (d *Document) usedGems(excluded *Equipment) map[int]bool {
	m := map[int]bool{}
	for slot := 0; slot <= 5; slot++ {
		for _, x := range d.EquipmentInventory(slot) {
			if excluded != nil && excluded.Slot == slot && excluded.Index() == x.Index {
				continue
			}
			for i := 0; i < 3; i++ {
				p := x.offset() + 0x18 + i*8
				if d.u16(p+2) == 3 {
					m[d.u16(p)] = true
				}
			}
		}
	}
	return m
}
func (d *Document) gemSlots(index int) []int {
	seen := map[int]bool{}
	for slot := 0; slot <= 5; slot++ {
		for _, x := range d.EquipmentInventory(slot) {
			for i := 0; i < 3; i++ {
				p := x.offset() + 0x18 + i*8
				if d.u16(p+2) == 3 && d.u16(p) == index {
					seen[slot] = true
				}
			}
		}
	}
	return sortedKeys(seen)
}

type Gem struct {
	d     *Document
	Index int
}

func (d *Document) Gem(index int) Gem {
	require(index >= 0 && index < Capacity, "Invalid gem index.")
	return Gem{d, index}
}
func (d *Document) Gems() []Gem {
	out := []Gem{}
	for i := 0; i < Capacity; i++ {
		g := d.Gem(i)
		if g.Exists() {
			out = append(out, g)
		}
	}
	return out
}
func (g Gem) offset() int                { return 0x2c380 + g.Index*0x2c }
func (g Gem) Exists() bool               { return g.d.u8(g.offset()+0x10) == 1 }
func (g Gem) Rank() int                  { return g.d.u8(g.offset() + 0x16) }
func (g Gem) EffectID() int              { return g.d.u16(g.offset() + 0x1c) }
func (g Gem) Value() int                 { return g.d.u16(g.offset() + 0x1e) }
func (g Gem) Strength() int              { return g.Value() & 255 }
func (g Gem) Chance() int                { return g.Value() >> 8 }
func (g Gem) Definition() *GemDefinition { return GemDefinitionFor(g.EffectID(), g.Rank()) }
func (g Gem) Label() string {
	return fmt.Sprintf("%s %d · %d", GemName(g.EffectID()), g.Rank(), g.Strength())
}
func (g Gem) CanEquip() bool {
	p := g.offset()
	return g.Exists() && g.Rank() >= 1 && g.Rank() <= 6 && GemNames[g.EffectID()] != "" && g.d.u16(p) == g.Index && g.d.u16(p+2) == 3 && g.d.u16(p+6) == 3 && g.d.u16(p+8) == 1 && g.d.u8(p+0x19) == 0 && g.d.u16(p+0x1a) == 1
}
func (g Gem) CanEdit() bool {
	return g.CanEquip() && g.Definition() != nil && g.d.Campaign() != "Unknown"
}
func (g Gem) CanFit(slot int) bool {
	return slot >= 0 && slot <= 5 && g.CanEquip() && g.Definition() != nil && g.Definition().Allows(slot)
}
func (g Gem) Set(effect, rank, strength int) {
	require(g.CanEdit(), "This is not an editable normal gem.")
	rule := GemDefinitionFor(effect, rank)
	require(rule != nil && strength >= rule.Minimum && strength <= rule.Maximum, "Invalid gem effect, rank or strength.")
	encoded := strength | rule.Chance<<8
	if effect == g.EffectID() && rank == g.Rank() && encoded == g.Value() {
		return
	}
	for _, slot := range g.d.gemSlots(g.Index) {
		require(rule.Allows(slot), "Remove this gem before choosing an incompatible effect.")
	}
	if effect != g.EffectID() || rank != g.Rank() {
		g.d.w16(g.offset()+4, 0)
	}
	g.d.w8(g.offset()+0x16, rank)
	g.d.w8(g.offset()+0x17, rule.Attribute)
	g.d.w16(g.offset()+0x1c, effect)
	g.d.w16(g.offset()+0x1e, encoded)
}
func (g Gem) Maximize() {
	rule := g.Definition()
	require(rule != nil, "Unknown gem definition.")
	g.Set(g.EffectID(), g.Rank(), rule.Maximum)
}
func (g Gem) CanDelete() bool { return g.CanEdit() && !g.d.usedGems(nil)[g.Index] }
func (g Gem) Delete() {
	require(g.CanDelete(), "Remove this gem from equipment before deleting it.")
	g.d.w8(g.offset()+0x10, 0)
}
func (d *Document) AddGem(effect, rank, strength int) Gem {
	require(d.Campaign() != "Unknown", "Unconfirmed campaign.")
	rule := GemDefinitionFor(effect, rank)
	require(rule != nil && strength >= rule.Minimum && strength <= rule.Maximum, "Invalid gem definition or value.")
	index := d.initialize(0x2c380, 0x2c, -1, 3, 0, 1, d.usedGems(nil))
	g := d.Gem(index)
	p := g.offset()
	d.w8(p+0x16, rank)
	d.w8(p+0x17, rule.Attribute)
	d.w16(p+0x1a, 1)
	d.w16(p+0x1c, effect)
	d.w16(p+0x1e, strength|rule.Chance<<8)
	return g
}

func (g Gem) IsCylinder() bool { return g.d.u8(g.offset()+0x19) != 0 }

func (g Gem) EquippedSlots() []int { return g.d.gemSlots(g.Index) }

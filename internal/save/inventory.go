package save

import "fmt"

var SlotNames = []string{"Weapon", "Head", "Torso", "Arms", "Legs", "Feet"}
var KindNames = map[int]string{10: "Collectables", 11: "Materials", 12: "KeyItems", 13: "ArtManuals"}

func equipmentStart(slot int) int {
	require(slot >= 0 && slot <= 5, "Invalid equipment slot.")
	return 0x3b10 + slot*0x5dc0
}
func equipmentType(slot int) int {
	if slot == 0 {
		return 2
	}
	return slot + 3
}
func characterField(slot int) int {
	if slot == 0 {
		return 0x28
	}
	return 0x10 + slot*4
}
func inventoryStart(kind int) int {
	require(kind >= 10 && kind <= 13, "Invalid inventory category.")
	return 0x31970 + (kind-10)*0x2710
}
func (d *Document) nextSerial(kind, start, size int) int {
	n := d.u32(0x46900 + kind*4)
	for i := 0; i < Capacity; i++ {
		p := start + i*size
		if d.u8(p+0x10) != 0 {
			n = max(n, d.u32(p+0xc))
		}
	}
	require(n < 0xffffffff, "Inventory serial counter cannot be incremented.")
	return n + 1
}
func (d *Document) freeSlot(start, size int, reserved map[int]bool) int {
	for i := 0; i < Capacity; i++ {
		if d.u8(start+i*size+0x10) == 0 && !reserved[i] {
			return i
		}
	}
	panic(editError("This inventory is full."))
}
func (d *Document) initialize(start, size, index, kind, id, quantity int, reserved map[int]bool) int {
	if index < 0 {
		index = d.freeSlot(start, size, reserved)
	}
	serial := d.nextSerial(kind, start, size)
	p := start + index*size
	clear(d.data[p : p+size])
	d.w16(p, index)
	d.w16(p+2, kind)
	d.w16(p+4, id)
	d.w16(p+6, kind)
	d.w16(p+8, quantity)
	d.w32(p+0xc, serial)
	d.w8(p+0x10, 1)
	d.w32(0x46900+kind*4, serial)
	return index
}

type InventoryItem struct {
	d           *Document
	Kind, Index int
}

func (d *Document) Item(kind, index int) InventoryItem {
	inventoryStart(kind)
	require(index >= 0 && index < Capacity, "Invalid inventory index.")
	return InventoryItem{d, kind, index}
}
func (d *Document) Inventory(kind int) []InventoryItem {
	out := []InventoryItem{}
	for i := 0; i < Capacity; i++ {
		x := d.Item(kind, i)
		if x.Exists() {
			out = append(out, x)
		}
	}
	return out
}
func (x InventoryItem) offset() int    { return inventoryStart(x.Kind) + x.Index*0x14 }
func (x InventoryItem) Exists() bool   { return x.d.u8(x.offset()+0x10) == 1 }
func (x InventoryItem) ItemID() int    { return x.d.u16(x.offset() + 4) }
func (x InventoryItem) Quantity() int  { return x.d.u16(x.offset() + 8) }
func (x InventoryItem) Favorite() bool { return x.d.u8(x.offset()+0x11) == 1 }
func (x InventoryItem) Name() string {
	if def := InventoryDefinitionByID(x.ItemID()); def != nil {
		return def.Name
	}
	return fmt.Sprintf("Item #%d", x.ItemID())
}
func (x InventoryItem) CanEdit() bool {
	def := InventoryDefinitionByID(x.ItemID())
	p := x.offset()
	return x.d.Campaign() != "Unknown" && x.Kind != 12 && x.Exists() && def != nil && def.Kind == x.Kind && x.d.u16(p) == x.Index && x.d.u16(p+2) == x.Kind && x.d.u16(p+6) == x.Kind
}
func (x InventoryItem) SetQuantity(n int) {
	require(x.CanEdit() && n >= 1 && n <= 99, "Quantity must be 1–99 on an editable stack.")
	x.d.w16(x.offset()+8, n)
}
func (x InventoryItem) SetFavorite(v bool) {
	require(x.CanEdit(), "Protected inventory record.")
	x.d.w8(x.offset()+0x11, boolInt(v))
}
func (x InventoryItem) Delete() {
	require(x.CanEdit(), "Protected inventory record.")
	x.d.w8(x.offset()+0x10, 0)
}
func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
func (d *Document) AddItem(id, quantity int) InventoryItem {
	def := InventoryDefinitionByID(id)
	require(def != nil && def.Kind != 12 && d.Campaign() != "Unknown", "This item cannot be added.")
	require(quantity >= 1 && quantity <= 99, "Quantity must be 1–99.")
	matches := []InventoryItem{}
	for _, x := range d.Inventory(def.Kind) {
		if x.ItemID() == id {
			matches = append(matches, x)
		}
	}
	require(len(matches) <= 1, "Duplicate stacks must be resolved first.")
	if len(matches) == 1 {
		x := matches[0]
		x.SetQuantity(x.Quantity() + quantity)
		return x
	}
	i := d.initialize(inventoryStart(def.Kind), 0x14, -1, def.Kind, id, quantity, nil)
	return d.Item(def.Kind, i)
}
func (d *Document) MaxItems(kind int) {
	require(kind != 12 && d.Campaign() != "Unknown", "Protected inventory category.")
	for _, x := range d.Inventory(kind) {
		require(x.CanEdit(), "Unrecognized inventory record.")
	}
	for _, x := range d.Inventory(kind) {
		x.SetQuantity(99)
	}
}

type OwnedEquipment struct {
	d           *Document
	Slot, Index int
}

func (d *Document) OwnedEquipment(slot, index int) OwnedEquipment {
	equipmentStart(slot)
	require(index >= 0 && index < Capacity, "Invalid equipment index.")
	return OwnedEquipment{d, slot, index}
}
func (x OwnedEquipment) offset() int     { return equipmentStart(x.Slot) + x.Index*0x30 }
func (x OwnedEquipment) Exists() bool    { return x.d.u8(x.offset()+0x10) != 0 }
func (x OwnedEquipment) ItemID() int     { return x.d.u16(x.offset() + 4) }
func (x OwnedEquipment) Name() string    { return ItemName(x.ItemID()) }
func (x OwnedEquipment) Sockets() int    { return x.d.u8(x.offset() + 0x15) }
func (x OwnedEquipment) ArmorClass() int { return x.d.u8(x.offset() + 0x14) }
func (x OwnedEquipment) Favorite() bool  { return x.d.u8(x.offset()+0x11) == 1 }
func (x OwnedEquipment) Valid() bool {
	p := x.offset()
	def := EquipmentDefinitionByID(x.ItemID())
	return x.d.u8(p+0x10) == 1 && x.d.u16(p) == x.Index && x.d.u16(p+2) == equipmentType(x.Slot) && x.d.u16(p+6) == equipmentType(x.Slot) && x.d.u16(p+8) == 1 && x.Sockets() <= 3 && def != nil && def.Slot == x.Slot && (x.Slot == 0 || x.ArmorClass() == def.ArmorClass)
}
func (x OwnedEquipment) CanEdit() bool {
	return x.d.Campaign() != "Unknown" && x.d.Version() == 7 && x.Valid()
}
func (x OwnedEquipment) CanDelete() bool {
	def := EquipmentDefinitionByID(x.ItemID())
	return x.CanEdit() && def != nil && def.Ordinary() && !x.d.referencedEquipment(x.Slot)[x.Index]
}
func (x OwnedEquipment) SetFavorite(v bool) {
	require(x.CanEdit(), "Protected equipment record.")
	x.d.w8(x.offset()+0x11, boolInt(v))
}
func (x OwnedEquipment) Delete() {
	require(x.CanDelete(), "Referenced or protected equipment cannot be deleted.")
	x.d.w8(x.offset()+0x10, 0)
}
func (x OwnedEquipment) FixedGems() []int {
	out := []int{}
	for i := 0; i < min(x.Sockets(), 3); i++ {
		p := x.offset() + 0x1c + i*8
		if x.d.u32(p) != 0 {
			out = append(out, x.d.u16(p))
		}
	}
	return out
}
func (x OwnedEquipment) EquippedBy() []int {
	out := []int{}
	for _, c := range x.d.Characters() {
		if c.Equipment(x.Slot).Index() == x.Index {
			out = append(out, c.ID)
		}
	}
	return out
}
func (d *Document) EquipmentInventory(slot int) []OwnedEquipment {
	out := []OwnedEquipment{}
	for i := 0; i < Capacity; i++ {
		x := d.OwnedEquipment(slot, i)
		if x.Exists() {
			out = append(out, x)
		}
	}
	return out
}
func (d *Document) referencedEquipment(slot int) map[int]bool {
	m := map[int]bool{}
	for id := 1; id <= 15; id++ {
		p := CharacterOffset + (id-1)*CharacterSize + characterField(slot)
		i := d.u16(p)
		if i < Capacity && (d.u16(p+2) == equipmentType(slot) || contains(d.Party(), id)) {
			m[i] = true
		}
	}
	return m
}
func (d *Document) CreatableEquipment(slot int) []EquipmentDefinition {
	equipmentStart(slot)
	out := []EquipmentDefinition{}
	if d.Campaign() == "Unknown" || d.Version() != 7 {
		return out
	}
	for _, def := range EquipmentDefinitions {
		if def.Slot != slot || !def.Ordinary() {
			continue
		}
		for _, id := range def.Characters {
			if d.Campaign() == "MainStory" && id >= 1 && id <= 8 || d.Campaign() == "FutureConnected" && contains([]int{1, 7, 14, 15}, id) {
				out = append(out, def)
				break
			}
		}
	}
	return out
}
func (d *Document) createEquipmentBatch(defs []EquipmentDefinition) []OwnedEquipment {
	counts := map[int]int{}
	for _, def := range defs {
		counts[def.Slot]++
	}
	for slot, count := range counts {
		free := 0
		reserved := d.referencedEquipment(slot)
		for i := 0; i < Capacity; i++ {
			if d.u8(equipmentStart(slot)+i*0x30+0x10) == 0 && !reserved[i] {
				free++
			}
		}
		require(free >= count, "Not enough unreferenced equipment slots.")
		first := d.nextSerial(equipmentType(slot), equipmentStart(slot), 0x30)
		require(first+count-1 <= 0xffffffff, "Equipment serial counter would overflow.")
	}
	out := []OwnedEquipment{}
	for _, def := range defs {
		i := d.initialize(equipmentStart(def.Slot), 0x30, -1, equipmentType(def.Slot), def.Id, 1, d.referencedEquipment(def.Slot))
		x := d.OwnedEquipment(def.Slot, i)
		p := x.offset()
		d.w8(p+0x14, def.ArmorClass)
		d.w8(p+0x15, def.GemSlotCount)
		for n := 1; n <= 3; n++ {
			if id := def.FixedGem(n); id > 0 {
				d.w32(p+0x1c+(n-1)*8, id|1<<16)
			}
		}
		out = append(out, x)
	}
	return out
}
func (d *Document) AddEquipment(id int) OwnedEquipment {
	def := EquipmentDefinitionByID(id)
	require(def != nil, "Unknown equipment.")
	ok := false
	for _, v := range d.CreatableEquipment(def.Slot) {
		if v.Id == id {
			ok = true
		}
	}
	require(ok, "Equipment cannot be created in this campaign.")
	return d.createEquipmentBatch([]EquipmentDefinition{*def})[0]
}
func (d *Document) FillEquipment(slot, character int) []OwnedEquipment {
	d.Character(character)
	owned := map[int]bool{}
	for _, x := range d.EquipmentInventory(slot) {
		owned[x.ItemID()] = true
	}
	defs := []EquipmentDefinition{}
	for _, def := range d.CreatableEquipment(slot) {
		if contains(def.Characters, character) && !owned[def.Id] {
			defs = append(defs, def)
		}
	}
	require(d.Campaign() != "Unknown" && d.Version() == 7, "Equipment inventory requires an identified format-7 save.")
	return d.createEquipmentBatch(defs)
}

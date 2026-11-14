package save

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestInventoryAndEquipmentRoundTrip(t *testing.T) {
	d := fixture(t)
	original := d.Bytes()
	var item InventoryItem
	if err := d.Edit(func(s *Document) { item = s.AddItem(1852, 3); s.Item(item.Kind, item.Index).SetFavorite(true) }); err != nil {
		t.Fatal(err)
	}
	x := d.Inventory(10)[0]
	if x.Quantity() != 3 || !x.Favorite() {
		t.Fatal("stack edit")
	}
	if byte(d.u8(0x120000)) != original[0x120000] {
		t.Fatal("unrelated bytes")
	}
	before := d.Bytes()
	if err := d.Edit(func(s *Document) { s.AddItem(1852, 99) }); err == nil || !bytes.Equal(before, d.Bytes()) {
		t.Fatal("invalid stack was not atomic")
	}
	if err := d.Edit(func(s *Document) { s.Item(10, x.Index).Delete() }); err != nil || len(d.Inventory(10)) != 0 {
		t.Fatal(err)
	}
}
func TestCollectionRewardsAndRollback(t *testing.T) {
	d := fixture(t)
	ids := d.CollectionIDs(2, 0)
	plan := d.PlanCollection(ids)
	if len(plan.Rewards) != 7 || len(plan.NewEntries) == 0 || plan.Rewards[0].EffectId != 77 {
		t.Fatal("page plan")
	}
	if err := d.Edit(func(s *Document) { s.CompleteCollection(ids) }); err != nil {
		t.Fatal(err)
	}
	first := d.Bytes()
	if len(d.PlanCollection(ids).Rewards) != 0 {
		t.Fatal("duplicate rewards")
	}
	if err := d.Edit(func(s *Document) { s.CompleteCollection(ids) }); err != nil || !bytes.Equal(first, d.Bytes()) {
		t.Fatal("repeat changed bytes")
	}
	full := fixture(t)
	for i := 0; i < Capacity; i++ {
		full.w8(0x2c380+i*0x2c+0x10, 2)
	}
	before := full.Bytes()
	if err := full.Edit(func(s *Document) { s.CompleteCollection(s.CollectionIDs(0, 0)) }); err == nil || !bytes.Equal(before, full.Bytes()) {
		t.Fatal("full inventory must roll back entire reward batch")
	}
}
func TestAffinityMonotonicAndSkills(t *testing.T) {
	d := fixture(t)
	if err := d.Edit(func(s *Document) { s.Affinity(1, 2).SetPoints(5000); s.Character(1).MaxSkills(true) }); err != nil {
		t.Fatal(err)
	}
	if err := d.Edit(func(s *Document) { s.Affinity(1, 2).SetPoints(0) }); err != nil {
		t.Fatal(err)
	}
	if d.Affinity(1, 2).Slots(1, 2) != 5 {
		t.Fatal("lowering affinity relocked slots")
	}
	for _, tree := range d.Character(1).SkillTrees() {
		if !tree.Unlocked() || tree.LearnedCount() != 5 || tree.Progress() != 0 {
			t.Fatal("skill branches")
		}
	}
}
func TestUserSaves(t *testing.T) {
	root := os.Getenv("XBDE_SAVE_DIRECTORY")
	if root == "" {
		t.Skip("set XBDE_SAVE_DIRECTORY for read-only user save validation")
	}
	files, _ := filepath.Glob(filepath.Join(root, "*.sav"))
	n := 0
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if len(raw) != FileSize {
			continue
		}
		d, err := Parse(raw)
		if err != nil {
			t.Fatal(path, err)
		}
		if !bytes.Equal(raw, d.Bytes()) {
			t.Fatal("lossless", path)
		}
		before := d.Bytes()
		if err = d.Edit(func(s *Document) {
			s.SetResources(pointer(123456), nil)
			for _, c := range s.Characters() {
				c.SetResources(pointer(1234), nil, nil)
			}
		}); err != nil {
			t.Fatal(err)
		}
		allowed := func(p int) bool {
			if p >= 0x151b40 && p < 0x151b44 {
				return true
			}
			for _, c := range d.Characters() {
				o := c.offset() + 8
				if p >= o && p < o+4 {
					return true
				}
			}
			return false
		}
		for p, b := range d.data {
			if b != before[p] && !allowed(p) {
				t.Fatalf("%s changed unrelated offset %#x", path, p)
			}
		}
		n++
	}
	if n == 0 {
		t.Fatal("no game saves tested")
	}
	t.Logf("verified %d original saves without writing them", n)
}

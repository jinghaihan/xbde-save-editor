package save

import (
	"bytes"
	"testing"
)

func fixture(t *testing.T) *Document {
	t.Helper()
	d := &Document{make([]byte, FileSize)}
	d.w32(0, 7)
	d.w8(PartyOffset+24, 2)
	d.w16(PartyOffset, 1)
	d.w16(PartyOffset+2, 2)
	d.w32(CharacterOffset, 20)
	d.w32(CharacterOffset+CharacterSize, 10)
	d.w8(0x120000, 0xa7)
	b, e := Parse(d.data)
	if e != nil {
		t.Fatal(e)
	}
	return b
}
func pointer(i int) *int { return &i }
func TestLosslessResourcesAndRejectedBatch(t *testing.T) {
	d := fixture(t)
	original := d.Bytes()
	if !bytes.Equal(original, d.Bytes()) {
		t.Fatal("copy differs")
	}
	if d.Campaign() != "MainStory" {
		t.Fatal("campaign")
	}
	if e := d.Edit(func(s *Document) { s.SetResources(pointer(812345), nil) }); e != nil {
		t.Fatal(e)
	}
	if d.Money() != 812345 {
		t.Fatal("money")
	}
	for i, b := range d.data {
		if i < 0x151b40 || i >= 0x151b44 {
			if b != original[i] {
				t.Fatalf("unrelated byte changed at %#x", i)
			}
		}
	}
	before := d.Bytes()
	if e := d.Edit(func(s *Document) { s.SetResources(pointer(123), pointer(1000000000)) }); e == nil {
		t.Fatal("invalid batch accepted")
	}
	if !bytes.Equal(before, d.Bytes()) {
		t.Fatal("rejected batch changed document")
	}
}
func TestProgressionPreservesExpertHistory(t *testing.T) {
	d := fixture(t)
	d.w32(CharacterOffset+0xec, 30)
	d.w32(CharacterOffset+0xf0, 51)
	d.w32(CharacterOffset+0xf4, 789)
	if e := d.Edit(func(s *Document) { s.Character(1).SetProgression(pointer(12), nil) }); e != nil {
		t.Fatal(e)
	}
	c := d.Character(1)
	if c.Level() != 12 || c.Experience() != 0 || c.ExpertLevel() != 30 || c.ExpertExperience() != 0 || c.ReserveExperience() != 789 {
		t.Fatal("linked progression fields differ")
	}
	before := d.Bytes()
	if e := d.Edit(func(s *Document) { s.Character(1).SetProgression(pointer(100), nil) }); e == nil || !bytes.Equal(before, d.Bytes()) {
		t.Fatal("invalid level was not rejected atomically")
	}
}
func TestCatalogsAndLinkedArt(t *testing.T) {
	if len(EquipmentDefinitions) != 1572 || len(SkillDefinitions) != 200 || len(AchievementDefinitions) != 200 {
		t.Fatal("catalog coverage")
	}
	d := fixture(t)
	d.w8(PartyOffset+24, 3)
	d.w16(PartyOffset+4, 7)
	d.w32(CharacterOffset+6*CharacterSize, 23)
	d.w8(ArtsOffset+(103-1)*2, 1)
	if e := d.Edit(func(s *Document) { s.Character(7).Art(103).SetLevel(10) }); e != nil {
		t.Fatal(e)
	}
	if d.u8(ArtsOffset+(97-1)*2) != 10 {
		t.Fatal("discharge art level not linked")
	}
}

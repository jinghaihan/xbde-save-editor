package save

import (
	"embed"
	"encoding/csv"
	"fmt"
	"sort"
	"strconv"
	"strings"
)

//go:embed data/*.tsv
var catalogs embed.FS

func rows(name string) [][]string {
	b, e := catalogs.ReadFile("data/" + name + ".tsv")
	if e != nil {
		panic(e)
	}
	r := csv.NewReader(strings.NewReader(string(b)))
	r.Comma = '\t'
	r.LazyQuotes = true
	all, e := r.ReadAll()
	if e != nil {
		panic(e)
	}
	return all[1:]
}
func number(s string) int {
	v, e := strconv.Atoi(s)
	if e != nil {
		panic(e)
	}
	return v
}
func names(name string) map[int]string {
	m := map[int]string{}
	for _, r := range rows(name) {
		m[number(r[0])] = r[1]
	}
	return m
}

var ItemNames = names("equipment")
var GemNames = names("gem-effects")

func ItemName(id int) string {
	if n := ItemNames[id]; n != "" {
		return n
	}
	return fmt.Sprintf("Item #%d", id)
}
func GemName(id int) string {
	if n := GemNames[id]; n != "" {
		return n
	}
	return fmt.Sprintf("Effect #%d", id)
}
func CharacterName(id int, language string) string {
	all := map[string][]string{
		"en":      {"", "Shulk", "Reyn", "Fiora", "Dunban", "Sharla", "Riki", "Melia", "Fiora", "Dickson", "Mumkhar", "Alvis", "Dunban", "Dunban", "Kino", "Nene"},
		"zh-Hans": {"", "修尔克", "莱恩", "菲奥伦", "丹邦", "卡露娜", "力奇", "梅莉亚", "菲奥伦", "迪克森", "穆姆卡", "阿尔维斯", "丹邦", "丹邦", "奇诺", "宁宁"},
		"zh-Hant": {"", "修爾克", "萊恩", "菲奧倫", "丹邦", "卡露娜", "力奇", "梅莉亞", "菲奧倫", "迪克森", "穆姆卡", "阿爾維斯", "丹邦", "丹邦", "奇諾", "寧寧"},
		"ja":      {"", "シュルク", "ライン", "フィオルン", "ダンバン", "カルナ", "リキ", "メリア", "フィオルン", "ディクソン", "ムムカ", "アルヴィース", "ダンバン", "ダンバン", "キノ", "ネネ"},
		"ko":      {"", "슈르크", "라인", "피오른", "단반", "카르나", "리키", "멜리아", "피오른", "딕슨", "무므카", "알비스", "단반", "단반", "키노", "네네"},
	}
	ns := all[language]
	if ns == nil {
		ns = all["en"]
	}
	if id > 0 && id < len(ns) {
		return ns[id]
	}
	return fmt.Sprintf("ID %d", id)
}

type ArtDefinition struct {
	Id, CharacterId              int
	Name                         string
	IsTalent                     bool
	Order, LearnType, LearnLevel int
}

func (a ArtDefinition) IsMonado() bool { return a.Id >= 3 && a.Id <= 10 }
func (a ArtDefinition) LinkedArtId() int {
	return map[int]int{97: 103, 98: 104, 99: 105, 100: 111, 101: 112, 102: 113}[a.Id]
}
func (a ArtDefinition) IsLevelLearned() bool { return a.LearnType == 1 && !a.IsTalent && !a.IsMonado() }
func (a ArtDefinition) HasMasterBook() bool {
	return !a.IsTalent && !a.IsMonado() && a.CharacterId != 3 && a.CharacterId != 14 && a.CharacterId != 15
}

var ArtDefinitions = func() []ArtDefinition {
	out := []ArtDefinition{}
	for _, r := range rows("arts") {
		out = append(out, ArtDefinition{number(r[0]), number(r[1]), r[2], r[3] == "1", number(r[4]), number(r[5]), number(r[6])})
	}
	return out
}()

type SkillDefinition struct {
	Id                               int
	Name                             string
	RequiredSP, Shape, AffinityCoins int
}

var Shapes = []string{"Circle", "Square", "Diamond", "Hexagon", "Octagram"}

func shape(s string) int {
	for i, v := range Shapes {
		if v == s {
			return i
		}
	}
	panic("unknown skill shape: " + s)
}
func (s SkillDefinition) CharacterId() int { return (s.Id-1)/25 + 1 }
func (s SkillDefinition) TreeIndex() int   { return (s.Id-1)%25/5 + 1 }
func (s SkillDefinition) NodeIndex() int   { return (s.Id - 1) % 5 }

var SkillDefinitions = func() map[int]SkillDefinition {
	out := map[int]SkillDefinition{}
	for _, r := range rows("skills") {
		id := number(r[0])
		out[id] = SkillDefinition{id, r[1], number(r[2]), shape(r[3]), number(r[4])}
	}
	return out
}()

type SkillTreeDefinition struct {
	CharacterId, Index int
	Name               string
	Skills             []SkillDefinition
}

func (t SkillTreeDefinition) UnlockFlag() int {
	if t.Index <= 3 {
		return 0
	}
	id := t.CharacterId
	if id == 8 {
		id = 3
	}
	return (id-1)*2 + t.Index - 3
}

var SkillTrees = func() []SkillTreeDefinition {
	out := []SkillTreeDefinition{}
	for _, r := range rows("skill-trees") {
		t := SkillTreeDefinition{CharacterId: number(r[0]), Index: number(r[1]), Name: r[2], Skills: []SkillDefinition{}}
		for i := 0; i < 5; i++ {
			id := (t.CharacterId-1)*25 + (t.Index-1)*5 + i + 1
			t.Skills = append(t.Skills, SkillDefinitions[id])
		}
		out = append(out, t)
	}
	return out
}()
var LinkShapes = func() map[[2]int][5]int {
	out := map[[2]int][5]int{}
	for _, r := range rows("skill-link-shapes") {
		id, src := number(r[0]), number(r[1])
		if id == src {
			continue
		}
		var ss [5]int
		for i := range ss {
			ss[i] = shape(r[i+2])
		}
		out[[2]int{id, src}] = ss
	}
	return out
}()

type AchievementDefinition struct {
	Id                                        int
	Name                                      string
	Category, Order                           int
	Condition                                 string
	RewardExperience, Required, ConditionType int
}

func (a AchievementDefinition) HasCounter() bool { return a.ConditionType == 3 || a.ConditionType == 4 }

var AchievementDefinitions = func() []AchievementDefinition {
	out := []AchievementDefinition{}
	for _, r := range rows("achievements") {
		out = append(out, AchievementDefinition{number(r[0]), r[1], number(r[2]), number(r[3]), r[4], number(r[5]), number(r[6]), number(r[7])})
	}
	return out
}()

type InventoryDefinition struct {
	Id, Kind int
	Name     string
}

var InventoryDefinitions = func() []InventoryDefinition {
	out := []InventoryDefinition{}
	for _, r := range rows("inventory") {
		out = append(out, InventoryDefinition{number(r[0]), number(r[1]), r[2]})
	}
	return out
}()

func InventoryDefinitionByID(id int) *InventoryDefinition {
	for i := range InventoryDefinitions {
		if InventoryDefinitions[i].Id == id {
			return &InventoryDefinitions[i]
		}
	}
	return nil
}

type EquipmentDefinition struct {
	Id, Slot, ArmorClass, GemSlotCount, FixedGem1, FixedGem2, FixedGem3, WeaponFlags int
	Name                                                                             string
	Characters                                                                       []int
}

func (d EquipmentDefinition) FixedGem(i int) int {
	return []int{d.FixedGem1, d.FixedGem2, d.FixedGem3}[i-1]
}
func (d EquipmentDefinition) Ordinary() bool {
	return d.Name != "" && d.Name != strconv.Itoa(d.Id) && !strings.HasPrefix(d.Name, "Item #") && !strings.EqualFold(d.Name, "Not Equipped") && !strings.Contains(strings.ToLower(d.Name), "dummy") && (d.Slot == 0 && (d.WeaponFlags == 0 || d.WeaponFlags == 3 && contains(d.Characters, 8)) || d.Slot != 0 && d.ArmorClass > 0)
}
func contains[T comparable](xs []T, x T) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

var EquipmentDefinitions = func() []EquipmentDefinition {
	rules := map[int][]string{}
	for _, r := range rows("equipment-rules") {
		rules[number(r[0])] = r
	}
	out := []EquipmentDefinition{}
	for _, r := range rows("equipment-initializers") {
		id := number(r[0])
		rule := rules[id]
		chars := []int{}
		for _, c := range strings.Split(rule[4], ",") {
			if c != "" {
				chars = append(chars, number(c))
			}
		}
		out = append(out, EquipmentDefinition{id, number(rule[1]), number(rule[2]), number(r[1]), number(r[2]), number(r[3]), number(r[4]), number(rule[3]), ItemName(id), chars})
	}
	return out
}()

func EquipmentDefinitionByID(id int) *EquipmentDefinition {
	for i := range EquipmentDefinitions {
		if EquipmentDefinitions[i].Id == id {
			return &EquipmentDefinitions[i]
		}
	}
	return nil
}

type GemDefinition struct{ EffectId, Rank, Minimum, Maximum, Chance, Attribute, ValueType, Attachment int }

func (g GemDefinition) Allows(slot int) bool {
	return g.Attachment == 0 || slot == 0 && g.Attachment == 1 || slot != 0 && g.Attachment == 2
}
func (g GemDefinition) Name() string { return GemName(g.EffectId) }
func (g GemDefinition) Unit() string {
	if g.ValueType == 3 {
		return "%"
	}
	return ""
}
func (g GemDefinition) Describe(v, chance int) string {
	if g.Maximum == 0 {
		return fmt.Sprintf("%d%%", chance)
	}
	s := fmt.Sprintf("%d%s", v, g.Unit())
	if chance != 0 {
		s += fmt.Sprintf(" / %d%%", chance)
	}
	return s
}

var GemDefinitions = func() []GemDefinition {
	out := []GemDefinition{}
	for _, r := range rows("gem-rules") {
		out = append(out, GemDefinition{number(r[0]), number(r[1]), number(r[2]), number(r[3]), number(r[4]), number(r[5]), number(r[6]), number(r[7])})
	}
	return out
}()

func GemDefinitionFor(effect, rank int) *GemDefinition {
	for i := range GemDefinitions {
		g := &GemDefinitions[i]
		if g.EffectId == effect && g.Rank == rank {
			return g
		}
	}
	return nil
}
func sortedKeys[T any](m map[int]T) []int {
	ks := []int{}
	for k := range m {
		ks = append(ks, k)
	}
	sort.Ints(ks)
	return ks
}

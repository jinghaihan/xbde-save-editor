package save

type Affinity struct {
	d                        *Document
	Index, FirstID, SecondID int
}

func canon(id int) int {
	if id == 8 {
		return 3
	}
	return id
}
func (d *Document) Affinities() []Affinity {
	out := []Affinity{}
	if d.Campaign() != "MainStory" {
		return out
	}
	ids := map[int]int{}
	for _, id := range d.Party() {
		if id >= 1 && id <= 8 && (ids[canon(id)] == 0 || id == 8) {
			ids[canon(id)] = id
		}
	}
	index := 0
	for a := 1; a <= 7; a++ {
		for b := a + 1; b <= 7; b++ {
			index++
			if ids[a] != 0 && ids[b] != 0 {
				out = append(out, Affinity{d, index, ids[a], ids[b]})
			}
		}
	}
	return out
}
func (d *Document) Affinity(a, b int) Affinity {
	for _, p := range d.Affinities() {
		if canon(p.FirstID) == min(canon(a), canon(b)) && canon(p.SecondID) == max(canon(a), canon(b)) && contains(d.Party(), a) && contains(d.Party(), b) {
			return p
		}
	}
	panic(editError("Choose two distinct joined characters."))
}
func (p Affinity) Points() int { return p.d.u16(0xe00 + p.Index*2) }
func (p Affinity) Slots(a, b int) int {
	v := p.d.u32(CharacterOffset + (a-1)*CharacterSize + 0xa4 + (b-1)*4)
	if v > 4 {
		return -1
	}
	return v + 1
}
func (p Affinity) SetPoints(n int) {
	require(p.d.MainSupported() && n >= 0 && n <= 5000, "Affinity must be 0–5,000 in main-story format 7.")
	p.d.w16(0xe00+p.Index*2, n)
	aliases := func(id int) []int {
		if canon(id) == 3 {
			return []int{3, 8}
		}
		return []int{id}
	}
	for _, a := range aliases(p.FirstID) {
		for _, b := range aliases(p.SecondID) {
			for _, pair := range [][2]int{{a, b}, {b, a}} {
				pos := CharacterOffset + (pair[0]-1)*CharacterSize + 0xa4 + (pair[1]-1)*4
				p.d.w32(pos, max(p.d.u32(pos), min(n/1000, 4)))
			}
		}
	}
}

var RegionNames = []string{"Colony 9 Area", "Colony 6 Area", "Central Bionis", "Upper Bionis", "Hidden Village"}

type Region struct {
	d  *Document
	ID int
}

func (d *Document) Regions() []Region {
	out := []Region{}
	if d.Campaign() == "MainStory" {
		for id := 1; id <= 5; id++ {
			out = append(out, Region{d, id})
		}
	}
	return out
}
func (d *Document) Region(id int) Region {
	require(d.Campaign() == "MainStory" && id >= 1 && id <= 5, "Choose one of the five main-story regions.")
	return Region{d, id}
}
func (r Region) Points() int { return r.d.u16(0xdf2 + r.ID*2) }
func (r Region) Stars() int  { return min(r.Points()/2000+1, 5) }
func (r Region) SetPoints(n int) {
	require(r.d.MainSupported() && n >= 0 && n <= 10000, "Region affinity must be 0–10,000 in main-story format 7.")
	r.d.w16(0xdf2+r.ID*2, n)
}
func (r Region) SetStars(n int) {
	require(r.d.MainSupported() && n >= 1 && n <= 5, "Stars must be 1–5.")
	if n != r.Stars() {
		r.SetPoints((n - 1) * 2000)
	}
}

type Achievement struct {
	d   *Document
	Def AchievementDefinition
}

func (d *Document) Achievements() []Achievement {
	out := []Achievement{}
	for _, def := range AchievementDefinitions {
		out = append(out, Achievement{d, def})
	}
	return out
}
func (d *Document) Achievement(id int) Achievement {
	for _, a := range d.Achievements() {
		if a.Def.Id == id {
			return a
		}
	}
	panic(editError("Unknown achievement."))
}
func (a Achievement) Completed() bool { return a.d.bit(0x50, 0x2838+a.Def.Id%200) }
func (a Achievement) Progress() int   { return a.d.u16(0xe30 + 2*(a.Def.Id%200)) }
func (a Achievement) Unmet() bool {
	return a.Completed() && a.Def.HasCounter() && (a.Def.ConditionType == 3 && a.Progress() != a.Def.Required || a.Progress() < a.Def.Required)
}
func (a Achievement) Unlock() {
	require(a.d.MainSupported(), "Achievements require main-story format 7.")
	if a.Completed() {
		return
	}
	if a.Def.HasCounter() {
		v := a.Def.Required
		if a.Def.ConditionType == 4 {
			v = max(v, a.Progress())
		}
		a.d.w16(0xe30+2*(a.Def.Id%200), v)
	}
	a.d.setBit(0x50, 0x2838+a.Def.Id%200)
}
func (a Achievement) Repair() {
	require(a.d.MainSupported(), "Achievement repair requires main-story format 7.")
	if a.Unmet() {
		a.d.w16(0xe30+2*(a.Def.Id%200), a.Def.Required)
	}
}

type ColonyState struct {
	Development, Population, Housing, Commerce, Nature, Special, OverallLevel int
	CanMaximize                                                               bool
	UnavailableReason                                                         string
}

var colonyUpgrades = [][5]int{{2, 4, 0, 51, 991}, {2, 8, 0, 52, 992}, {3, 15, 0, 53, 993}, {3, 18, 0, 54, 994}, {3, 27, 46, 55, 995}, {2, 1, 0, 0, 0}, {2, 1, 0, 96, 0}, {2, 1, 0, 0, 0}, {3, 1, 0, 0, 0}, {4, 2, 0, 0, 982}, {1, 0, 0, 0, 984}, {2, 0, 38, 0, 985}, {2, 0, 0, 0, 986}, {3, 0, 0, 0, 987}, {5, 3, 0, 0, 998}, {3, 0, 0, 0, 934}, {3, 2, 39, 0, 990}, {4, 0, 53, 0, 997}, {5, 0, 44, 0, 937}, {6, 3, 29, 0, 938}, {0, 0, 32, 0, 0}, {0, 0, 33, 0, 0}, {0, 0, 34, 0, 0}, {0, 0, 35, 0, 0}, {0, 0, 36, 0, 0}}

func (d *Document) missingUpgrades() [][5]int {
	out := [][5]int{}
	for cat := 2; cat <= 6; cat++ {
		for n := d.u8(0xcf8 + cat); n < 5; n++ {
			out = append(out, colonyUpgrades[(cat-2)*5+n])
		}
	}
	return out
}
func (d *Document) colonyRestriction() string {
	if !d.MainSupported() {
		return "Colony 6 requires main-story format 7."
	}
	minlevel := 5
	for i := 2; i <= 6; i++ {
		v := d.u8(0xcf8 + i)
		if v > 5 {
			return "Inconsistent Colony 6 levels."
		}
		if i < 6 {
			minlevel = min(minlevel, v)
		}
	}
	if d.u8(0xcfe) > minlevel {
		return "Inconsistent Colony 6 levels."
	}
	if d.u8(0xcfa)+d.u8(0xcfb)+d.u8(0xcfc)+d.u8(0xcfd) == 0 {
		return "Complete the first Colony 6 upgrade in-game."
	}
	dev, pop := d.u8(0xcf8), d.u8(0xcf9)
	for _, s := range d.missingUpgrades() {
		dev += s[0]
		pop += s[1]
		if s[3] != 0 && d.u8(0xc94+s[3]) > 1 || s[4] != 0 && d.u8(0x5f0+s[4]) > 200 {
			return "Unrecognized linked Colony 6 flag."
		}
	}
	if dev > 255 || pop > 255 {
		return "Colony 6 development or population would overflow."
	}
	return ""
}
func (d *Document) Colony6() *ColonyState {
	if d.Campaign() != "MainStory" {
		return nil
	}
	reason := d.colonyRestriction()
	return &ColonyState{d.u8(0xcf8), d.u8(0xcf9), d.u8(0xcfa), d.u8(0xcfb), d.u8(0xcfc), d.u8(0xcfd), d.u8(0xcfe), reason == "", reason}
}
func (d *Document) MaxColony6() {
	require(d.colonyRestriction() == "", d.colonyRestriction())
	dev, pop := d.u8(0xcf8), d.u8(0xcf9)
	for _, s := range d.missingUpgrades() {
		dev += s[0]
		pop += s[1]
		if s[2] != 0 {
			d.setBit(0x50, 0x1d6a+s[2])
		}
		if s[3] != 0 {
			d.w8(0xc94+s[3], 1)
		}
		if s[4] != 0 {
			d.w8(0x5f0+s[4], 200)
		}
	}
	d.w8(0xcf8, dev)
	d.w8(0xcf9, pop)
	for i := 2; i <= 6; i++ {
		d.w8(0xcf8+i, 5)
	}
}

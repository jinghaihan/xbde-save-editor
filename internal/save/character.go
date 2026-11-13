package save

import "sort"

var levelCosts = []int{0, 0, 40, 60, 145, 208, 280, 338, 400, 436, 507, 585, 707, 790, 928, 1016, 1170, 1263, 1432, 1531, 1716, 1924, 2238, 2457, 2802, 3031, 3408, 3647, 4056, 4305, 4745, 5265, 6016, 6557, 7371, 7932, 8808, 9391, 10329, 10933, 11934, 13182, 14911, 16200, 18054, 19385, 21364, 22737, 24840, 26254, 28483, 31395, 35287, 38282, 42424, 45502, 49894, 53055, 57696, 60941, 65832, 72488, 81122, 87945, 97078, 104067, 113700, 120855, 130988, 138309, 148941, 163917, 182868, 198177, 218127, 233768, 254716, 270691, 292637, 308945, 331890, 365170, 406419, 440364, 483610, 518221, 563464, 598741, 645980, 681922, 731159, 804375, 893547, 968094, 1061260, 1137138, 1234298, 1311507, 1412660, 1491201}

type Character struct {
	d  *Document
	ID int
}

func (c Character) offset() int            { return CharacterOffset + (c.ID-1)*CharacterSize }
func (c Character) Level() int             { return c.d.u32(c.offset()) }
func (c Character) Experience() int        { return c.d.u32(c.offset() + 4) }
func (c Character) AP() int                { return c.d.u32(c.offset() + 8) }
func (c Character) AffinityCoins() int     { return c.d.u32(c.offset() + 12) }
func (c Character) ExpertLevel() int       { return c.d.u32(c.offset() + 0xec) }
func (c Character) ExpertExperience() int  { return c.d.u32(c.offset() + 0xf0) }
func (c Character) ReserveExperience() int { return c.d.u32(c.offset() + 0xf4) }
func (c Character) CanEditProgression() bool {
	return c.d.Campaign() == "MainStory" && c.ID >= 1 && c.ID <= 8 || c.d.Campaign() == "FutureConnected" && contains([]int{1, 7, 14, 15}, c.ID)
}
func (c Character) MinimumLevel() int {
	require(c.CanEditProgression(), "Progression editing requires an identified campaign and supported character.")
	if c.d.Campaign() == "FutureConnected" {
		if c.ID == 1 || c.ID == 7 {
			return 60
		}
		return 58
	}
	return []int{0, 1, 1, 2, 20, 10, 22, 23, 40}[c.ID]
}
func ExperienceToNextLevel(level int) int {
	require(level >= 1 && level <= 99, "Invalid level.")
	if level == 99 {
		return 0
	}
	return levelCosts[level+1]
}
func (c Character) SetProgression(level, experience *int) {
	target := c.Level()
	if level != nil {
		target = *level
	}
	require(target >= c.MinimumLevel() && target <= 99, "Level is outside this character's range.")
	if level == nil && experience == nil {
		return
	}
	exp := 0
	if target == c.Level() {
		exp = c.Experience()
	}
	if experience != nil {
		exp = *experience
	}
	require(exp >= 0 && exp <= 99999999, "Experience must be between 0 and 99,999,999.")
	for target < 99 && exp >= levelCosts[target+1] {
		target++
		exp -= levelCosts[target]
	}
	if target != c.Level() {
		c.d.w32(c.offset()+0xec, max(c.ExpertLevel(), target))
		c.d.w32(c.offset()+0xf0, 0)
	}
	c.d.w32(c.offset(), target)
	c.d.w32(c.offset()+4, exp)
}
func (c Character) SetResources(ap, coins, reserve *int) {
	for _, x := range []struct {
		offset, limit int
		p             *int
	}{{8, 99999999, ap}, {12, 999, coins}, {0xf4, 199999998, reserve}} {
		if x.p != nil {
			require(*x.p >= 0 && *x.p <= x.limit, "Character resource is outside its allowed range.")
			if x.offset == 12 {
				require(c.d.Campaign() == "MainStory", "Affinity Coins can only be edited in an identified main-story save.")
			}
			c.d.w32(c.offset()+x.offset, *x.p)
		}
	}
}
func (c Character) Arts() []Art {
	out := []Art{}
	for _, def := range ArtDefinitions {
		if def.CharacterId == c.ID && def.LinkedArtId() == 0 {
			out = append(out, Art{c.d, def})
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Def.Order != out[j].Def.Order {
			return out[i].Def.Order < out[j].Def.Order
		}
		return out[i].Def.Id < out[j].Def.Id
	})
	return out
}
func (c Character) Art(id int) Art {
	for _, a := range c.Arts() {
		if a.Def.Id == id {
			return a
		}
	}
	panic(editError("This art does not belong to the selected character."))
}
func (c Character) MaxArts(learn bool) {
	require(c.CanEditProgression(), "Arts require an identified campaign and supported character.")
	for _, a := range c.Arts() {
		if learn && a.CanLearn() {
			a.Learn(a.MaximumLevel())
		}
		if a.CanEdit() {
			a.SetLevel(a.MaximumLevel())
		}
	}
}

type Art struct {
	d   *Document
	Def ArtDefinition
}

func (a Art) offset() int      { return ArtsOffset + (a.Def.Id-1)*2 }
func (a Art) Level() int       { return a.d.u8(a.offset()) }
func (a Art) ManualFlags() int { return a.d.u8(a.offset() + 1) }
func (a Art) Name() string {
	if a.Def.IsMonado() {
		return "Monado " + a.Def.Name
	}
	return a.Def.Name
}
func (a Art) Learned() bool { return a.Level() != 0 }
func (a Art) CanLearn() bool {
	return !a.Learned() && (a.Def.IsLevelLearned() || contains([]int{118, 143}, a.Def.Id) && a.d.MainSupported()) && a.d.Character(a.Def.CharacterId).CanEditProgression()
}
func (a Art) CanEdit() bool {
	return a.Learned() && !a.Def.IsTalent && a.d.Character(a.Def.CharacterId).CanEditProgression() && !(a.d.Campaign() == "FutureConnected" && a.Def.Id == 4)
}
func (a Art) ManualLimit() int {
	switch a.ManualFlags() & 7 {
	case 7:
		return 12
	case 3:
		return 10
	case 1, 5:
		return 7
	}
	return 4
}
func (a Art) MaximumLevel() int {
	if a.Def.IsTalent {
		return 1
	}
	if a.Def.IsMonado() {
		return min(10, a.ManualLimit())
	}
	if a.d.Campaign() == "MainStory" && a.Def.HasMasterBook() {
		return 12
	}
	return 10
}
func (a Art) UnlockedMaximum() int { return min(a.MaximumLevel(), a.ManualLimit()) }
func (a Art) SetLevel(level int) {
	require(a.CanEdit(), "Only learned, upgradeable arts in an identified campaign can be edited.")
	a.write(level)
}
func (a Art) Learn(level int) {
	require(a.CanLearn(), "Only supported unlearned arts can be learned directly.")
	a.write(level)
}
func (a Art) write(level int) {
	require(level >= 1 && level <= a.MaximumLevel(), "Invalid art level.")
	a.d.w8(a.offset(), level)
	if !a.Def.IsMonado() {
		flags := 0
		if level > 10 {
			flags = 7
		} else if level > 7 {
			flags = 3
		} else if level > 4 {
			flags = 1
		}
		a.d.w8(a.offset()+1, a.ManualFlags()|flags)
	}
	for _, def := range ArtDefinitions {
		if def.LinkedArtId() == a.Def.Id {
			a.d.w8(ArtsOffset+(def.Id-1)*2, level)
		}
	}
}

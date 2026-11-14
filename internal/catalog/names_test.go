package catalog

import "testing"

func TestOfficialNamesAndCoverage(t *testing.T) {
	if len(names) != 9 {
		t.Fatal("language coverage")
	}
	for lang, domains := range names {
		for domain, count := range map[string]int{"item": 3537, "art": 178, "skill": 200, "gem": 98, "achievement": 200, "map": 23, "region": 5, "achievement-condition": 200} {
			if len(domains[domain]) != count {
				t.Errorf("%s/%s: %d, want %d", lang, domain, len(domains[domain]), count)
			}
		}
	}
	if Item("zh-Hans", 1, "") != "蒙纳多" || Name("zh-Hans", "art", "1", "") != "蒙纳多发动" || Name("zh-Hans", "map", "2", "") != "９号殖民地" {
		t.Fatal("original ID/message join changed")
	}
	if Name("fr", "tree", "3/4", "Innocence") != "Innocence" {
		t.Fatal("ambiguous translation guessed")
	}
	if Item("en", 3, "3") != "3" {
		t.Fatal("blank placeholder changed")
	}
}

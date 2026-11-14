package catalog

import (
	"embed"
	"encoding/json"
	"strconv"
)

//go:embed data/*.json
var files embed.FS
var names = func() map[string]map[string]map[string]string {
	b, e := files.ReadFile("data/names.json")
	if e != nil {
		panic(e)
	}
	var n map[string]map[string]map[string]string
	if e = json.Unmarshal(b, &n); e != nil {
		panic(e)
	}
	return n
}()
var manualTiers = func() map[string]string {
	b, e := files.ReadFile("data/manual-tiers.json")
	if e != nil {
		panic(e)
	}
	var n map[string]string
	if e = json.Unmarshal(b, &n); e != nil {
		panic(e)
	}
	return n
}()

func Name(language, domain, key, fallback string) string {
	if v := names[language][domain][key]; v != "" {
		return v
	}
	return fallback
}
func Item(language string, id int, fallback string) string {
	key := strconv.Itoa(id)
	name := Name(language, "item", key, "")
	if name == "" {
		return fallback
	}
	if tier := manualTiers[key]; tier != "" {
		name += " (" + tier + ")"
	}
	return name
}

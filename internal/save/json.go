package save

import "encoding/json"

func (s SkillDefinition) MarshalJSON() ([]byte, error) {
	return json.Marshal(map[string]any{"Id": s.Id, "Name": s.Name, "RequiredSP": s.RequiredSP, "Shape": s.Shape, "AffinityCoins": s.AffinityCoins, "CharacterId": s.CharacterId(), "TreeIndex": s.TreeIndex(), "NodeIndex": s.NodeIndex()})
}

package helpers

import (
	"encoding/json"
)

// rawDiff represents the diff as it comes out of GAWD
type rawDiff struct {
	Type string `json:"type"`
	Old  struct {
		Path  string `json:"path"`
		Value string `json:"value"`
	} `json:"old"`
	New struct {
		Path  string `json:"path"`
		Value string `json:"value"`
	} `json:"new"`
}

// DiffBody represents the actual changes present in the [FlatDiff] struct
type DiffBody struct {
	Type    string `bson:"type"`
	PathMod string `bson:"path_mod"`
	Old     string `bson:"old"`
	New     string `bson:"new"`
}

// FlatDiff represents the restructured diff grouped by path
type FlatDiff struct {
	FromCommit string              `bson:"from_commit"`
	ToCommit   string              `bson:"to_commit"`
	Diff       map[string]DiffBody `bson:"diff"`
}

// GroupByPath converts the diff outputted by GAWD, and groups it by path
func GroupByPath(raw []byte, precFile, succFile string) FlatDiff {
	var jsonDiff []rawDiff
	var bsonFlatDiff FlatDiff

	bsonFlatDiff.FromCommit = precFile
	bsonFlatDiff.ToCommit = succFile
	bsonFlatDiff.Diff = map[string]DiffBody{}

	err := json.Unmarshal(raw, &jsonDiff)

	if err != nil {
		panic(err)
	}

	for _, diff := range jsonDiff {
		var path string
		var pathMod string

		switch diff.Type {
		case "added":
			path = diff.New.Path
		case "renamed", "moved":
			path = diff.Old.Path
			pathMod = diff.New.Path
		default:
			path = diff.Old.Path
		}

		bsonFlatDiff.Diff[path] = DiffBody{
			Type:    diff.Type,
			PathMod: pathMod,
			Old:     diff.Old.Value,
			New:     diff.New.Value,
		}
	}

	return bsonFlatDiff
}

package cmd

import (
	"hash/fnv"
	"math/rand"
)

// Docker-style adjectives
var adjectives = []string{
	"admiring", "adoring", "agitated", "amazing", "angry",
	"awesome", "blissful", "bold", "boring", "brave",
	"busy", "charming", "clever", "cool", "cranky",
	"crazy", "dazzling", "determined", "distracted", "dreamy",
	"eager", "ecstatic", "elastic", "elegant", "eloquent",
	"epic", "fervent", "festive", "flamboyant", "focused",
	"friendly", "frosty", "funny", "gallant", "gifted",
	"goofy", "gracious", "happy", "hardcore", "heuristic",
	"hopeful", "hungry", "infallible", "inspiring", "jolly",
	"jovial", "keen", "kind", "laughing", "loving",
}

// Famous scientists and inventors (Docker-style)
var scientists = []string{
	"albattani", "allen", "almeida", "archimedes", "ardinghelli",
	"babbage", "banach", "bardeen", "bartik", "bassi",
	"bell", "benz", "bhabha", "bhaskara", "blackwell",
	"bohr", "booth", "borg", "bose", "boyd",
	"brahmagupta", "brattain", "brown", "carson", "chandrasekhar",
	"chatelet", "colden", "cori", "cray", "curie",
	"darwin", "davinci", "dijkstra", "dubinsky", "easley",
	"edison", "einstein", "elion", "engelbart", "euclid",
	"euler", "fermat", "fermi", "feynman", "franklin",
	"galileo", "gates", "goldberg", "golick", "hopper",
}

// GenerateBuildName creates a Docker-style build name from a timestamp.
// The same timestamp always produces the same name.
func GenerateBuildName(buildTime string) string {
	h := fnv.New64a()
	h.Write([]byte(buildTime))
	seed := int64(h.Sum64())

	r := rand.New(rand.NewSource(seed))

	adj := adjectives[r.Intn(len(adjectives))]
	sci := scientists[r.Intn(len(scientists))]

	return adj + "_" + sci
}

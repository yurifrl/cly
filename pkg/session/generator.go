package session

import (
	"math/rand"
	"strings"
	"time"
)

var adjectives = []string{
	"Quick", "Bright", "Swift", "Calm", "Bold",
	"Clever", "Eager", "Fresh", "Gentle", "Happy",
	"Jolly", "Keen", "Lively", "Merry", "Noble",
	"Proud", "Quiet", "Rapid", "Sharp", "Warm",
	"Active", "Brave", "Clear", "Daring", "Epic",
	"Fair", "Grand", "Humble", "Ideal", "Just",
	"Kind", "Lucky", "Mighty", "Neat", "Open",
	"Prime", "Royal", "Solid", "True", "Vivid",
	"Wise", "Young", "Agile", "Cosmic", "Divine",
	"Eternal", "Fluid", "Golden", "Hidden", "Ionic",
}

var animals = []string{
	"Fox", "Owl", "Bear", "Wolf", "Hawk",
	"Eagle", "Tiger", "Lion", "Deer", "Falcon",
	"Panda", "Koala", "Otter", "Raven", "Shark",
	"Whale", "Horse", "Bison", "Crane", "Dove",
	"Swan", "Heron", "Moose", "Zebra", "Gecko",
	"Viper", "Cobra", "Finch", "Robin", "Sparrow",
	"Parrot", "Salmon", "Trout", "Orca", "Seal",
	"Badger", "Ferret", "Marten", "Stoat", "Weasel",
	"Coyote", "Jackal", "Lynx", "Bobcat", "Cougar",
	"Panther", "Jaguar", "Lemur", "Gibbon", "Tapir",
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func GenerateName() string {
	adj := adjectives[rand.Intn(len(adjectives))]
	animal := animals[rand.Intn(len(animals))]
	return strings.Title(adj) + strings.Title(animal)
}

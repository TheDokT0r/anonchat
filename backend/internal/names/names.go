package names

import (
	"math/rand/v2"
)

var adjectives = []string{
	"Red", "Blue", "Green", "Gold", "Silver",
	"Swift", "Brave", "Calm", "Dark", "Bright",
	"Wild", "Shy", "Bold", "Cool", "Warm",
	"Quiet", "Loud", "Quick", "Slow", "Sharp",
}

var animals = []string{
	"Fox", "Bear", "Wolf", "Hawk", "Owl",
	"Deer", "Lynx", "Crow", "Otter", "Panda",
	"Tiger", "Eagle", "Heron", "Cobra", "Raven",
	"Bison", "Crane", "Viper", "Moose", "Falcon",
}

func Generate(existing []string) string {
	taken := make(map[string]bool, len(existing))
	for _, name := range existing {
		taken[name] = true
	}

	for range 100 {
		name := adjectives[rand.IntN(len(adjectives))] + " " + animals[rand.IntN(len(animals))]
		if !taken[name] {
			return name
		}
	}

	for _, adj := range adjectives {
		for _, animal := range animals {
			name := adj + " " + animal
			if !taken[name] {
				return name
			}
		}
	}

	base := adjectives[rand.IntN(len(adjectives))] + " " + animals[rand.IntN(len(animals))]
	return base + " " + string(rune('0'+rand.IntN(10)))
}

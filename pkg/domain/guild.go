package domain

// Guild represents one of the six Grimora guilds.
type Guild struct {
	ID       string
	Name     string
	Animal   string
	Color    string
	HexColor string
}

// The six guilds — locked, do not change.
var Guilds = map[string]Guild{
	"loomari":  {ID: "loomari", Name: "Loomari", Animal: "Spider", Color: "purple", HexColor: "#9B59B6"},
	"ashborne": {ID: "ashborne", Name: "Ashborne", Animal: "Phoenix", Color: "orange", HexColor: "#E67E22"},
	"amarok":   {ID: "amarok", Name: "Amarok", Animal: "Wolf", Color: "blue", HexColor: "#3498DB"},
	"nyx":      {ID: "nyx", Name: "Nyx", Animal: "Raven", Color: "gray", HexColor: "#95A5A6"},
	"cipher":   {ID: "cipher", Name: "Cipher", Animal: "Serpent", Color: "green", HexColor: "#2ECC71"},
	"fathom":   {ID: "fathom", Name: "Fathom", Animal: "Octopus", Color: "teal", HexColor: "#1ABC9C"},
}

// ValidGuildID returns true if the given ID is a known guild.
func ValidGuildID(id string) bool {
	_, ok := Guilds[id]
	return ok
}

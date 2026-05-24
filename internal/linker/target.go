package linker

import "strings"

type TargetProfile struct {
	Name   string
	Triple string
	Flash  Region
	Ram    Region
}

var targetProfiles = map[string]TargetProfile{
	"baremetal": {
		Name:   "baremetal",
		Triple: "baremetal",
		Flash:  Region{"FLASH", 0x08000000, 0x00100000, PermRead | PermExec},
		Ram:    Region{"RAM", 0x20000000, 0x00010000, PermRead | PermWrite},
	},
	"cortex-m3": {
		Name:   "cortex-m3",
		Triple: "baremetal",
		Flash:  Region{"FLASH", 0x08000000, 0x00100000, PermRead | PermExec},
		Ram:    Region{"RAM", 0x20000000, 0x00010000, PermRead | PermWrite},
	},
	"cortex-m4": {
		Name:   "cortex-m4",
		Triple: "baremetal",
		Flash:  Region{"FLASH", 0x08000000, 0x00100000, PermRead | PermExec},
		Ram:    Region{"RAM", 0x20000000, 0x00010000, PermRead | PermWrite},
	},
}

func SetTarget(target string) {
	Target = target
}

func TargetProfileFor(target string) (TargetProfile, bool) {
	normalized := strings.ToLower(strings.TrimSpace(target))
	profile, ok := targetProfiles[normalized]
	return profile, ok
}

func IsBareMetalTarget() bool {
	if _, ok := TargetProfileFor(Target); ok {
		return true
	}
	if strings.Contains(Target, "baremetal") || strings.Contains(Target, "unknown-elf") || strings.Contains(Target, "none-") {
		return true
	}
	return false
}

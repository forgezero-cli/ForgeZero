/*
 *   Copyright (c) 2026 ForgeZero-cli
 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

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

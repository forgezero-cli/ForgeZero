/*
 * Copyright (c) 2026 ForgeZero-cli
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package helpers

import (
	"flag"
	"runtime"

	"github.com/forgezero-cli/ForgeZero/internal/profiles"
)

func SetupProfile(flags *Flags) {
	profileProvided := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "profile" || f.Name == "p" {
			profileProvided = true
		}
	})
	
	if !profileProvided {
		if saved, err := profiles.ReadSavedProfile(""); err == nil && saved != "" {
			flags.ProfileFlag = saved
		}
	}
	
	p := profiles.ParseUserProfile(flags.ProfileFlag)
	flags.ProfileFlag = p.Name
	maxProcs := p.DefaultJobs()
	if maxProcs < 1 {
		maxProcs = 1
	}
	runtime.GOMAXPROCS(maxProcs)
	
	if flags.Jobs < 0 {
		flags.Jobs = 1
	}
	
	if profileProvided {
		_ = profiles.SaveProfile("", flags.ProfileFlag)
	}
}
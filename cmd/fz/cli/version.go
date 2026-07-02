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

package cli

import (
	"runtime"
	"strings"

	"github.com/forgezero-cli/ForgeZero/cmd/fz/stdio"
)

const (
	VersionCodename = "Forge"
)

var VersionCore = "unknown"

var BuildDate = "unknown"

func VersionText() string {
    var b strings.Builder
    b.Grow(180)
    b.WriteString("ForgeZero ")
    b.WriteString(VersionCore)
    b.WriteString(" [")
    b.WriteString(VersionCodename)
    b.WriteString("] built on ")
    b.WriteString(BuildDate)
    b.WriteString(" (")
    b.WriteString(runtime.GOOS)
    b.WriteByte('/')
    b.WriteString(runtime.GOARCH)
    b.WriteString(") · GPLv3 · (c) ForgeZero-cli")
    return b.String()
}
func OutputVersion() {
	stdio.WriteStdout(VersionText())
}

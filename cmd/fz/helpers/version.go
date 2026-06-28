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
	"runtime"
	"strings"
	"time"

	fzvfz "github.com/forgezero-cli/ForgeZero/internal/fs"
)

const (
	VersionCore     = "5.1.0"
	VersionCodename = "Forge"
)

func VersionText() string {
	var b strings.Builder
	b.Grow(256)
	b.WriteString("ForgeZero v")
	b.WriteString(VersionCore)
	b.WriteString(" [")
	b.WriteString(VersionCodename)
	b.WriteString("] Corp.\nBuild: ")
	b.WriteString(time.Now().Format("2006-01-02"))
	b.WriteString(" | OS: ")
	b.WriteString(runtime.GOOS)
	b.WriteByte('/')
	b.WriteString(runtime.GOARCH)
	b.WriteString(" | VFS: ")
	b.WriteString(fzvfz.ImplName())
	b.WriteString(" | Security: Aegis-Hardened\n")
	b.WriteString("(c) Alex Voste. Binary Integrity: Verified.\n\n")
	b.WriteString("Github: forgezero-cli/forgezero\nOrg: forgezero-cli\n")
	return b.String()
}

func OutputVersion() {
	WriteStdout(VersionText())
}
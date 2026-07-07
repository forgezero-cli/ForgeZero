/*
 *   Copyright (c) 2026 ForgeZero-cli

 *
 *   This program is free software: you can redistribute it and/or modify
 *   it under the terms of the GNU General Public License as published by
 *   the Free Software Foundation, either version 3 of the License, or
 *   (at your option) any later version of the License.
 *
 *   This program is distributed in the hope that it will be useful,
 *   but WITHOUT ANY WARRANTY; without even the implied warranty of
 *   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 *   GNU General Public License for more details.
 *
 *   You should have received a copy of the GNU General Public License
 *   along with this program.  If not, see <https://www.gnu.org/licenses/>.
 */

package logger

import "os"

var Enabled bool

func Debug(msg string) {
	if Enabled {
		os.Stdout.WriteString(msg)
	}
}

func Debugf(format string, args ...any) {
	if Enabled {
		var out string
		out = format
		for _, arg := range args {
			out += " "
			out += toString(arg)
		}
		os.Stdout.WriteString(out)
	}
}

func toString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	default:
		return ""
	}
}

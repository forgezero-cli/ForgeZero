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

package fzerr

type Code uint16

const (
	CodeOK Code = 0

	CodeFileNotFound Code = 1
	CodeParseFailed  Code = 2
	CodeConfigInvalid Code = 3
	CodeBuildActionFailed Code = 4
	CodeDepResolutionFailed Code = 5

	CodeHashOpen Code = 10
	CodeHashMmap Code = 11
	CodeHashSize Code = 12
	CodeHashRead Code = 13

	CodeCacheEmpty Code = 20
	CodeCacheWrite Code = 21
	CodeCacheRead  Code = 22

	CodeIORead  Code = 30
	CodeIOWrite Code = 31

	CodeLinkFailed Code = 40
	CodeLinkSkip   Code = 41

	CodeSchedulerFull Code = 50
	CodeSchedulerTask Code = 51

	CodePathInvalid Code = 60
	CodePathOutside Code = 61

	CodeGeneric Code = 255
)

var codeNames = [...]string{
	0:  "ok",
	1:  "file_not_found",
	2:  "parse_failed",
	3:  "config_invalid",
	4:  "build_action_failed",
	5:  "dependency_resolution_failed",
	10: "hash_open",
	11: "hash_mmap",
	12: "hash_size",
	13: "hash_read",
	20: "cache_empty",
	21: "cache_write",
	22: "cache_read",
	30: "io_read",
	31: "io_write",
	40: "link_failed",
	41: "link_skip",
	50: "scheduler_full",
	51: "scheduler_task",
	60: "path_invalid",
	61: "path_outside",
}

func CodeName(c Code) string {
	if int(c) < len(codeNames) && codeNames[c] != "" {
		return codeNames[c]
	}
	return "error"
}

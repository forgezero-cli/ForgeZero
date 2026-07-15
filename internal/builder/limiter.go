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

package builder

import (
	"bytes"
	"os"
	"runtime"
	"strconv"
	"strings"
)

const compileWorkerMemMB = 1024

func AdjustJobs(requested int) int {
	if requested <= 0 {
		requested = 1
	}
	available := availableMemMB()
	if available == 0 {
		if requested > runtime.NumCPU() {
			requested = runtime.NumCPU()
		}
		return requested
	}
	maxWorkers := int(available / compileWorkerMemMB)
	if maxWorkers < 1 {
		maxWorkers = 1
	}
	if requested > maxWorkers {
		requested = maxWorkers
	}
	if requested > runtime.NumCPU() {
		requested = runtime.NumCPU()
	}
	return requested
}

func availableMemMB() uint64 {
	data, err := os.ReadFile("/proc/meminfo")
	if err != nil {
		return 0
	}
	available, free := parseMemInfo(data)
	if available > 0 {
		return available / 1024
	}
	return free / 1024
}

func parseMemInfo(data []byte) (uint64, uint64) {
	available := uint64(0)
	free := uint64(0)
	scanner := bytes.NewReader(data)
	for {
		line, err := readLine(scanner)
		if err != nil {
			break
		}
		fields := strings.Fields(string(line))
		if len(fields) < 2 {
			continue
		}
		key := strings.TrimSuffix(fields[0], ":")
		value, parseErr := strconv.ParseUint(fields[1], 10, 64)
		if parseErr != nil {
			continue
		}
		switch key {
		case "MemAvailable":
			available = value
		case "MemFree":
			free = value
		}
	}
	return available, free
}

func readLine(r *bytes.Reader) ([]byte, error) {
	buf := make([]byte, 0, 128)
	for {
		b, err := r.ReadByte()
		if err != nil {
			if len(buf) == 0 {
				return nil, err
			}
			return buf, nil
		}
		if b == '\n' {
			return buf, nil
		}
		buf = append(buf, b)
	}
}

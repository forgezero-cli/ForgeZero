/*
 *   Copyright (c) 2026 forgezero-cli
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

import (
	"errors"

	"github.com/forgezero-cli/ForgeZero/internal/forge"
)

type Permissions uint8

const (
	PermRead Permissions = 1 << iota
	PermWrite
	PermExec
)

const (
	SectionText = ".text"
	SectionData = ".data"
	SectionBSS  = ".bss"
)

type Region struct {
	Name        string
	Origin      uint32
	Length      uint32
	Permissions Permissions
}

type SectionLayout struct {
	Name        string
	Origin      uint32
	Length      uint32
	Permissions Permissions
	Data        []byte
}

type Layout struct {
	Regions  [2]Region
	Sections [3]SectionLayout
}

var (
	ErrInvalidAlignment = errors.New("address must be 4-byte aligned")
	ErrRegionOverflow   = errors.New("section exceeds region bounds")
	ErrSectionOverlap   = errors.New("sections overlap inside region")
)

func Align4(addr uint32) uint32 {
	return (addr + 3) &^ 3
}

func NewNakedMemoryLayout(flash Region, ram Region, textData, dataData []byte, bssSize uint32) (Layout, error) {
	if flash.Origin&3 != 0 || ram.Origin&3 != 0 {
		return Layout{}, ErrInvalidAlignment
	}
	var layout Layout
	layout.Regions[0] = flash
	layout.Regions[1] = ram
	textOrigin, textOriginOK := align4Checked(uint64(flash.Origin))
	textLength, textLengthOK := align4Checked(uint64(len(textData)))
	if !textOriginOK || !textLengthOK || textLength > 0 && textOrigin+textLength > uint64(flash.Origin)+uint64(flash.Length) {
		return Layout{}, ErrRegionOverflow
	}
	layout.Sections[0] = SectionLayout{
		Name:        SectionText,
		Origin:      uint32(textOrigin),
		Length:      uint32(textLength),
		Permissions: PermRead | PermExec,
		Data:        textData,
	}
	dataOrigin, dataOriginOK := align4Checked(uint64(ram.Origin))
	dataLength, dataLengthOK := align4Checked(uint64(len(dataData)))
	if !dataOriginOK || !dataLengthOK || dataLength > 0 && dataOrigin+dataLength > uint64(ram.Origin)+uint64(ram.Length) {
		return Layout{}, ErrRegionOverflow
	}
	layout.Sections[1] = SectionLayout{
		Name:        SectionData,
		Origin:      uint32(dataOrigin),
		Length:      uint32(dataLength),
		Permissions: PermRead | PermWrite,
		Data:        dataData,
	}
	bssOrigin, bssOriginOK := align4Checked(dataOrigin + dataLength)
	bssLength, bssLengthOK := align4Checked(uint64(bssSize))
	if !bssOriginOK || !bssLengthOK || bssLength > 0 && bssOrigin+bssLength > uint64(ram.Origin)+uint64(ram.Length) {
		return Layout{}, ErrRegionOverflow
	}
	layout.Sections[2] = SectionLayout{
		Name:        SectionBSS,
		Origin:      uint32(bssOrigin),
		Length:      uint32(bssLength),
		Permissions: PermRead | PermWrite,
	}
	return layout, nil
}

func align4Checked(value uint64) (uint64, bool) {
	aligned := (value + 3) &^ 3
	return aligned, aligned <= uint64(^uint32(0))
}

func EmitFlatBinary(layout Layout) ([]byte, error) {
	ordered, regionSizes, totalSize, err := flatBinarySize(layout)
	if err != nil {
		return nil, err
	}
	buffer := make([]byte, totalSize)
	writeFlatBinaryInto(buffer, layout, ordered, regionSizes)
	return buffer, nil
}

func flatBinarySize(layout Layout) ([2]Region, [2]int, int, error) {
	ordered := layout.Regions
	if ordered[1].Origin < ordered[0].Origin {
		ordered[0], ordered[1] = ordered[1], ordered[0]
	}

	totalSize := 0
	regionSizes := [2]int{}
	for i := 0; i < 2; i++ {
		ofRegion, count := collectSectionsForRegion(ordered[i], layout.Sections)
		if count == 0 {
			continue
		}
		size, err := regionOutputSize(ordered[i], ofRegion, count)
		if err != nil {
			return [2]Region{}, [2]int{}, 0, err
		}
		regionSizes[i] = size
		totalSize += size
	}
	return ordered, regionSizes, totalSize, nil
}

func writeFlatBinaryInto(buffer []byte, layout Layout, ordered [2]Region, regionSizes [2]int) {
	offset := 0
	for i := 0; i < 2; i++ {
		if regionSizes[i] == 0 {
			continue
		}
		ofRegion, count := collectSectionsForRegion(ordered[i], layout.Sections)
		writeRegionOutput(buffer[offset:offset+regionSizes[i]], ordered[i], ofRegion, count)
		offset += regionSizes[i]
	}
}

func collectSectionsForRegion(region Region, sections [3]SectionLayout) ([3]SectionLayout, int) {
	var result [3]SectionLayout
	count := 0
	for i := 0; i < 3; i++ {
		s := sections[i]
		if s.Length == 0 && s.Data == nil {
			continue
		}
		if uint64(s.Origin) >= uint64(region.Origin) && uint64(s.Origin) < uint64(region.Origin)+uint64(region.Length) {
			result[count] = s
			count++
		}
	}
	for i := 0; i < count; i++ {
		for j := i + 1; j < count; j++ {
			if result[j].Origin < result[i].Origin {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result, count
}

func regionOutputSize(region Region, sections [3]SectionLayout, count int) (int, error) {
	var offset uint64
	for i := 0; i < count; i++ {
		s := sections[i]
		if s.Origin < region.Origin {
			return 0, ErrRegionOverflow
		}
		relative := uint64(s.Origin) - uint64(region.Origin)
		if relative < offset {
			return 0, ErrSectionOverlap
		}
		end := relative + uint64(s.Length)
		if end > uint64(region.Length) {
			return 0, ErrRegionOverflow
		}
		offset = end
	}
	return int(offset), nil
}

func writeRegionOutput(buffer []byte, region Region, sections [3]SectionLayout, count int) {
	var offset uint32
	for i := 0; i < count; i++ {
		s := sections[i]
		relative := s.Origin - region.Origin
		if relative > offset {
			offset = relative
		}
		if len(s.Data) > 0 {
			forge.MoveBytes(buffer[offset:], s.Data, forge.FlagNoCache)
			offset += uint32(len(s.Data))
			if pad := s.Length - uint32(len(s.Data)); pad > 0 {
				offset += pad
			}
			continue
		}
		offset += s.Length
	}
}

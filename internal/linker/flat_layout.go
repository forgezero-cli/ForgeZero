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

import "errors"

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
	textOrigin := Align4(flash.Origin)
	textLength := Align4(uint32(len(textData)))
	if textLength > 0 && textOrigin+textLength > flash.Origin+flash.Length {
		return Layout{}, ErrRegionOverflow
	}
	layout.Sections[0] = SectionLayout{
		Name:        SectionText,
		Origin:      textOrigin,
		Length:      textLength,
		Permissions: PermRead | PermExec,
		Data:        textData,
	}
	dataOrigin := Align4(ram.Origin)
	dataLength := Align4(uint32(len(dataData)))
	if dataLength > 0 && dataOrigin+dataLength > ram.Origin+ram.Length {
		return Layout{}, ErrRegionOverflow
	}
	layout.Sections[1] = SectionLayout{
		Name:        SectionData,
		Origin:      dataOrigin,
		Length:      dataLength,
		Permissions: PermRead | PermWrite,
		Data:        dataData,
	}
	bssOrigin := Align4(dataOrigin + dataLength)
	bssLength := Align4(bssSize)
	if bssLength > 0 && bssOrigin+bssLength > ram.Origin+ram.Length {
		return Layout{}, ErrRegionOverflow
	}
	layout.Sections[2] = SectionLayout{
		Name:        SectionBSS,
		Origin:      bssOrigin,
		Length:      bssLength,
		Permissions: PermRead | PermWrite,
	}
	return layout, nil
}

func EmitFlatBinary(layout Layout) ([]byte, error) {
	ordered := layout.Regions
	if ordered[1].Origin < ordered[0].Origin {
		ordered[0], ordered[1] = ordered[1], ordered[0]
	}
	var buffer []byte
	for i := 0; i < 2; i++ {
		ofRegion, count := collectSectionsForRegion(ordered[i], layout.Sections)
		if count == 0 {
			continue
		}
		size, err := regionOutputSize(ordered[i], ofRegion, count)
		if err != nil {
			return nil, err
		}
		start := len(buffer)
		buffer = append(buffer, make([]byte, size)...)
		writeRegionOutput(buffer[start:], ordered[i], ofRegion, count)
	}
	return buffer, nil
}

func collectSectionsForRegion(region Region, sections [3]SectionLayout) ([3]SectionLayout, int) {
	var result [3]SectionLayout
	count := 0
	for i := 0; i < 3; i++ {
		s := sections[i]
		if s.Length == 0 && s.Data == nil {
			continue
		}
		if s.Origin >= region.Origin && s.Origin < region.Origin+region.Length {
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
	var offset uint32
	for i := 0; i < count; i++ {
		s := sections[i]
		if s.Origin < region.Origin {
			return 0, ErrRegionOverflow
		}
		relative := s.Origin - region.Origin
		if relative < offset {
			return 0, ErrSectionOverlap
		}
		end := relative + s.Length
		if end > region.Length {
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
			copy(buffer[offset:], s.Data)
			offset += uint32(len(s.Data))
			if pad := s.Length - uint32(len(s.Data)); pad > 0 {
				offset += pad
			}
			continue
		}
		offset += s.Length
	}
}

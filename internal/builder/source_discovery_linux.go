//go:build linux

/*
 * Copyright (c) 2026 forgezero-cli
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

package builder

import (
	"encoding/binary"
	"sort"
	"unsafe"

	"github.com/forgezero-cli/ForgeZero/internal/io_uring"
	"golang.org/x/sys/unix"
)

type discoveredSourceFile struct {
	path    string
	inode   uint64
	size    int64
	modTime int64
}

type sourceDirectory struct {
	fd   int
	path string
}

func discoverSourceFiles(root string) ([]discoveredSourceFile, error) {
	rootFD, err := unix.Open(root, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
	if err != nil {
		return nil, err
	}

	files := make([]discoveredSourceFile, 0, 128)
	stack := []sourceDirectory{{fd: rootFD, path: root}}
	buffer := make([]byte, 64<<10)
	for len(stack) != 0 {
		last := len(stack) - 1
		directory := stack[last]
		stack = stack[:last]
		for {
			n, readErr := readDirectoryEntries(directory.fd, buffer)
			if readErr != nil {
				closeSourceDirectories(stack)
				unix.Close(directory.fd)
				return nil, readErr
			}
			if n == 0 {
				break
			}
			for offset := 0; offset < n; {
				if offset+19 > n {
					closeSourceDirectories(stack)
					unix.Close(directory.fd)
					return nil, unix.EIO
				}
				recordLength := int(binary.LittleEndian.Uint16(buffer[offset+16 : offset+18]))
				if recordLength < 19 || offset+recordLength > n {
					closeSourceDirectories(stack)
					unix.Close(directory.fd)
					return nil, unix.EIO
				}
				nameBytes := buffer[offset+19 : offset+recordLength]
				nameLength := 0
				for nameLength < len(nameBytes) && nameBytes[nameLength] != 0 {
					nameLength++
				}
				if nameLength == 0 {
					offset += recordLength
					continue
				}
				name := unsafe.String(&nameBytes[0], nameLength)
				if name == "." || name == ".." {
					offset += recordLength
					continue
				}
				linkInfo, err := statSourceEntry(directory.fd, name, unix.AT_SYMLINK_NOFOLLOW)
				if err != nil {
					closeSourceDirectories(stack)
					unix.Close(directory.fd)
					return nil, err
				}
				isLink := linkInfo.mode&unix.S_IFMT == unix.S_IFLNK
				info := linkInfo
				if isLink {
					info, err = statSourceEntry(directory.fd, name, 0)
					if err != nil {
						closeSourceDirectories(stack)
						unix.Close(directory.fd)
						return nil, err
					}
				}
				path := joinSourcePath(directory.path, name)
				if info.mode&unix.S_IFMT == unix.S_IFDIR {
					if isLink {
						offset += recordLength
						continue
					}
					child, openErr := unix.Openat(directory.fd, name, unix.O_RDONLY|unix.O_DIRECTORY|unix.O_CLOEXEC|unix.O_NOFOLLOW, 0)
					if openErr != nil {
						closeSourceDirectories(stack)
						unix.Close(directory.fd)
						return nil, openErr
					}
					stack = append(stack, sourceDirectory{fd: child, path: path})
					offset += recordLength
					continue
				}
				files = append(files, discoveredSourceFile{path: path, inode: info.inode, size: info.size, modTime: info.modTime})
				offset += recordLength
			}
		}
		unix.Close(directory.fd)
	}
	sort.Slice(files, func(left, right int) bool {
		return files[left].path < files[right].path
	})
	return files, nil
}

func joinSourcePath(directory, name string) string {
	if len(directory) == 0 {
		return name
	}
	if directory[len(directory)-1] == '/' {
		return directory + name
	}
	return directory + "/" + name
}

type sourceEntryInfo struct {
	inode   uint64
	size    int64
	modTime int64
	mode    uint32
}

func statSourceEntry(dirFD int, name string, flags int) (sourceEntryInfo, error) {
	if io_uring.Enabled() {
		statx, err := io_uring.StatxAt(dirFD, name, flags, unix.STATX_TYPE|unix.STATX_SIZE|unix.STATX_MTIME|unix.STATX_INO)
		if err == nil {
			return sourceEntryInfo{
				inode:   statx.Ino,
				size:    int64(statx.Size),
				modTime: statx.MtimeSec*1e9 + int64(statx.MtimeNsec),
				mode:    uint32(statx.Mode),
			}, nil
		}
	}
	var statx unix.Statx_t
	err := unix.Statx(dirFD, name, flags, unix.STATX_TYPE|unix.STATX_SIZE|unix.STATX_MTIME|unix.STATX_INO, &statx)
	if err == nil {
		return sourceEntryInfo{
			inode:   statx.Ino,
			size:    int64(statx.Size),
			modTime: statx.Mtime.Sec*1e9 + int64(statx.Mtime.Nsec),
			mode:    uint32(statx.Mode),
		}, nil
	}
	if err != unix.ENOSYS && err != unix.EINVAL {
		return sourceEntryInfo{}, err
	}
	var legacy unix.Stat_t
	if err := unix.Fstatat(dirFD, name, &legacy, flags); err != nil {
		return sourceEntryInfo{}, err
	}
	return sourceEntryInfo{
		inode:   legacy.Ino,
		size:    legacy.Size,
		modTime: int64(legacy.Mtim.Sec)*1e9 + int64(legacy.Mtim.Nsec),
		mode:    legacy.Mode,
	}, nil
}

func readDirectoryEntries(fd int, buffer []byte) (int, error) {
	for {
		n, err := unix.Getdents(fd, buffer)
		if err != unix.EINTR {
			return n, err
		}
	}
}

func closeSourceDirectories(stack []sourceDirectory) {
	for _, directory := range stack {
		unix.Close(directory.fd)
	}
}

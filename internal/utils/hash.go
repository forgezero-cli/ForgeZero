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

package utils

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"unsafe"

	"github.com/zeebo/blake3"
)

var (
	hashKey    = [32]byte{0x9d, 0x74, 0x31, 0x6f, 0xd5, 0x23, 0x1b, 0xe4, 0xa1, 0x8f, 0x03, 0x71, 0x42, 0x5d, 0x6b, 0x9a, 0x3c, 0xf4, 0x75, 0x28, 0x0d, 0x62, 0x8a, 0x19, 0xbf, 0x4e, 0x50, 0x33, 0x13, 0x21, 0x97, 0x6c}
	hasherPool = sync.Pool{New: func() any { h, _ := blake3.NewKeyed(hashKey[:]); return h }}
	hashSep    = []byte{0}
)

var (
	ErrHashOpen = errors.New("hash: open")
	ErrHashMmap = errors.New("hash: mmap")
	ErrHashSize = errors.New("hash: size")
	ErrHashRead = errors.New("hash: read")
)

var alignedBufPool = sync.Pool{
	New: func() any {
		b := make([]byte, 1024*1024+64)
		base := uintptr(unsafe.Pointer(&b[0]))
		off := int((64 - (base % 64)) % 64)
		s := b[off : off+1024*1024]
		return &s
	},
}

func getAlignedBuf(size int) []byte {
	bufPtr := alignedBufPool.Get().(*[]byte)
	b := *bufPtr
	if cap(b) < size {
		b = make([]byte, size+64)
		base := uintptr(unsafe.Pointer(&b[0]))
		off := int((64 - (base % 64)) % 64)
		b = b[off : off+size]
		*bufPtr = b
	}
	return b[:size]
}

func putAlignedBuf(b []byte) {
	alignedBufPool.Put(&b)
}

func blake3HexDigestToString(d [32]byte) string {
	var out [64]byte
	const hextable = "0123456789abcdef"
	for i := 0; i < 32; i++ {
		b := d[i]
		out[i*2] = hextable[b>>4]
		out[i*2+1] = hextable[b&0x0f]
	}
	return string(out[:])
}

func BuildMerkleRoot(root string) ([32]byte, error) {
	var out [32]byte
	if root == "" {
		return out, errors.New("invalid merkle root")
	}
	files, err := collectRootFiles(root)
	if err != nil {
		return out, err
	}
	var reg [256][32]byte
	count := 0
	for i := range files {
		if count >= len(reg) {
			return out, errors.New("merkle registry overflow")
		}
		h, err := HashFileDigest(files[i])
		if err != nil {
			return out, err
		}
		reg[count] = h
		count++
	}
	if count == 0 {
		return hashEmptyDigest()
	}
	for count > 1 {
		next := 0
		for i := 0; i < count; i += 2 {
			left := reg[i]
			right := left
			if i+1 < count {
				right = reg[i+1]
			}
			reg[next] = hashDataPair(left, right)
			next++
		}
		count = next
	}
	return reg[0], nil
}

func hashDataPair(left, right [32]byte) [32]byte {
	var buf [64]byte
	copy(buf[:32], left[:])
	copy(buf[32:], right[:])
	h, err := HashDataDigest(buf[:])
	if err != nil {
		return [32]byte{}
	}
	return h
}

func collectRootFiles(root string) ([]string, error) {
	rootAbs, err := resolveOrAbs(root)
	if err != nil {
		return nil, err
	}
	var files []string
	walkErr := filepath.Walk(rootAbs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(rootAbs, path)
		if err != nil {
			return err
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return errors.New("invalid path outside root: " + path)
		}
		files = append(files, path)
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}
	sort.Strings(files)
	return files, nil
}

func hashEmptyDigest() ([32]byte, error) {
	var out [32]byte
	hasher := hasherPool.Get().(*blake3.Hasher)
	digest := hasher.Digest()
	if _, err := digest.Read(out[:]); err != nil {
		hasher.Reset()
		hasherPool.Put(hasher)
		return out, err
	}
	hasher.Reset()
	hasherPool.Put(hasher)
	return out, nil
}

func HashDataDigest(data []byte) ([32]byte, error) {
	var out [32]byte
	hasher := hasherPool.Get().(*blake3.Hasher)
	if _, err := hasher.Write(data); err != nil {
		hasher.Reset()
		hasherPool.Put(hasher)
		return out, err
	}
	digest := hasher.Digest()
	if _, err := digest.Read(out[:]); err != nil {
		hasher.Reset()
		hasherPool.Put(hasher)
		return out, err
	}
	hasher.Reset()
	hasherPool.Put(hasher)
	return out, nil
}

func HashFileDigest(path string) ([32]byte, error) {
	var out [32]byte
	resolved, err := ResolveSecurePathCached(path)
	if err != nil {
		return out, ErrHashOpen
	}
	return hashRawFileDigest(resolved)
}

func HashFile(path string) (string, error) {
	out, err := HashFileDigest(path)
	if err != nil {
		return "", err
	}
	return blake3HexDigestToString(out), nil
}

func HashFileCached(path string) (string, error) {
	if v, ok := fileHashCache.Load(path); ok {
		return v.(string), nil
	}
	h, err := HashFile(path)
	if err == nil {
		fileHashCache.Store(path, h)
	}
	return h, err
}

func HashDir(root string) (string, error) {
	rootAbs, err := resolveOrAbs(root)
	if err != nil {
		return "", errors.New("hash dir " + root + ": " + err.Error())
	}
	return HashDirWithRoot(rootAbs, rootAbs)
}

func HashDirWithRoot(rootAbs, dir string) (string, error) {
	digest, err := HashDirDigest(rootAbs, dir)
	if err != nil {
		return "", err
	}
	return blake3HexDigestToString(digest), nil
}

func HashDirDigest(rootAbs, dir string) ([32]byte, error) {
	var out [32]byte
	dirAbs, err := resolveOrAbs(dir)
	if err != nil {
		return out, ErrHashRead
	}
	rootEval, _ := ResolveSecurePath(rootAbs)
	if rootEval == "" {
		rootEval = filepath.Clean(rootAbs)
	}
	var files []string
	walkErr := filepath.Walk(dirAbs, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.Mode()&os.ModeSymlink != 0 {
			ok, serr := symlinkAllowed(rootEval, path, "")
			if serr != nil {
				return serr
			}
			if !ok {
				return nil
			}
			abs, aerr := resolveOrAbs(path)
			if aerr != nil {
				return aerr
			}
			target, aerr := fileSystem().EvalSymlinks(abs)
			if aerr != nil {
				return aerr
			}
			tinfo, aerr := fileSystem().Lstat(target)
			if aerr != nil {
				return aerr
			}
			if tinfo.IsDir() {
				return nil
			}
			path = target
			info = tinfo
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dirAbs, path)
		if err != nil {
			return err
		}
		if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
			return ErrHashRead
		}
		files = append(files, rel)
		return nil
	})
	if walkErr != nil {
		return out, ErrHashRead
	}
	sort.Strings(files)

	type fileHash struct {
		rel string
		h   [32]byte
		err error
	}
	results := make([]fileHash, len(files))
	workers := runtime.GOMAXPROCS(0)
	if workers > len(files) {
		workers = len(files)
	}
	next := int64(-1)
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func() {
			defer wg.Done()
			for {
				idx := int(atomic.AddInt64(&next, 1))
				if idx >= len(files) {
					return
				}
				rel := files[idx]
				h, err := hashRawFileDigest(filepath.Join(dirAbs, rel))
				results[idx] = fileHash{rel: rel, h: h, err: err}
			}
		}()
	}
	wg.Wait()

	hasher := hasherPool.Get().(*blake3.Hasher)
	buf := getAlignedBuf(32 * 1024)
	defer putAlignedBuf(buf)

	for _, res := range results {
		if res.err != nil {
			hasher.Reset()
			hasherPool.Put(hasher)
			return out, res.err
		}
		n := copy(buf, res.rel)
		if _, err := hasher.Write(buf[:n]); err != nil {
			hasher.Reset()
			hasherPool.Put(hasher)
			return out, ErrHashRead
		}
		if _, err := hasher.Write(hashSep); err != nil {
			hasher.Reset()
			hasherPool.Put(hasher)
			return out, ErrHashRead
		}
		if _, err := hasher.Write(res.h[:]); err != nil {
			hasher.Reset()
			hasherPool.Put(hasher)
			return out, ErrHashRead
		}
		if _, err := hasher.Write(hashSep); err != nil {
			hasher.Reset()
			hasherPool.Put(hasher)
			return out, ErrHashRead
		}
	}
	digest := hasher.Digest()
	if _, err := digest.Read(out[:]); err != nil {
		hasher.Reset()
		hasherPool.Put(hasher)
		return out, err
	}
	hasher.Reset()
	hasherPool.Put(hasher)
	return out, nil
}

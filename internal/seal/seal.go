package seal

import (
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"bytes"

	"github.com/zeebo/blake3"
	"golang.org/x/sys/unix"
)

var sealed bool
var combinedSeal [32]byte
var allowed map[string]struct{}
var allowedMu sync.RWMutex

func getExecPath() (string, error) {
	var buf [4096]byte
	n, err := unix.Readlink("/proc/self/exe", buf[:])
	if err != nil {
		return "", err
	}
	return filepath.Clean(string(buf[:n])), nil
}

func getMachineIDZeroAlloc() (string, error) {
	fd, err := unix.Open("/etc/machine-id", unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return "", err
	}
	defer unix.Close(fd)
	var buf [128]byte
	n, err := unix.Read(fd, buf[:])
	if err != nil && err != unix.EINTR && n == 0 {
		return "", err
	}
	s := string(buf[:n])
	s = strings.TrimSpace(s)
	return s, nil
}

func computeFileHash(path string) ([32]byte, error) {
	var out [32]byte
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return out, err
	}
	defer unix.Close(fd)
	hasher := blake3.New()
	var buf [32768]byte
	for {
		n, err := unix.Read(fd, buf[:])
		if n > 0 {
			hasher.Write(buf[:n])
		}
		if err != nil {
			break
		}
	}
	sum := hasher.Sum(nil)
	copy(out[:], sum[:32])
	return out, nil
}

func writeAll(fd int, b []byte) error {
	for len(b) > 0 {
		n, err := unix.Write(fd, b)
		if err != nil {
			return err
		}
		b = b[n:]
	}
	return nil
}

func setImmutable(path string) error {
	fd, err := unix.Open(path, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	const FS_IMMUTABLE_FL = 0x00000010
	return unix.IoctlSetInt(fd, unix.FS_IOC_SETFLAGS, FS_IMMUTABLE_FL)
}

func Seal() error {
	execPath, err := getExecPath()
	if err != nil {
		return err
	}
	execHash, err := computeFileHash(execPath)
	if err != nil {
		return err
	}
	mid, _ := getMachineIDZeroAlloc()
	hasher := blake3.New()
	hasher.Write(execHash[:])
	hasher.Write([]byte(mid))
	sum := hasher.Sum(nil)
	copy(combinedSeal[:], sum[:32])
	dir := filepath.Dir(execPath)
	sealPath := filepath.Join(dir, ".fz_seal")
	fd, err := unix.Open(sealPath, unix.O_WRONLY|unix.O_CREAT|unix.O_TRUNC|unix.O_CLOEXEC, 0600)
	if err != nil {
		return err
	}
	defer unix.Close(fd)
	hexb := make([]byte, hex.EncodedLen(len(sum)))
	hex.Encode(hexb, sum)
	hexb = append(hexb, '\n')
	if err := writeAll(fd, hexb); err != nil {
		return err
	}
	root := filepath.Dir(execPath)
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		h, err := computeFileHash(path)
		if err != nil {
			return nil
		}
		hb := make([]byte, hex.EncodedLen(len(h)))
		hex.Encode(hb, h[:])
		hb = append(hb, '\t')
		hb = append(hb, []byte(path)...)
		hb = append(hb, '\n')
		_ = writeAll(fd, hb)
		return nil
	})
	if err := setImmutable(sealPath); err == nil {
	}
	sealed = true
	return nil
}

func Verify() (bool, error) {
	if sealed {
		return true, nil
	}
	execPath, err := getExecPath()
	if err != nil {
		return false, err
	}
	dir := filepath.Dir(execPath)
	sealPath := filepath.Join(dir, ".fz_seal")
	fd, err := unix.Open(sealPath, unix.O_RDONLY|unix.O_CLOEXEC, 0)
	if err != nil {
		return false, err
	}
	defer unix.Close(fd)
	var st unix.Stat_t
	if err := unix.Fstat(fd, &st); err != nil {
		return false, err
	}
	size := int(st.Size)
	if size <= 0 {
		return false, nil
	}
	buf := make([]byte, size)
	off := 0
	for off < size {
		n, err := unix.Read(fd, buf[off:])
		if n > 0 {
			off += n
		}
		if err != nil {
			break
		}
	}
	lines := strings.Split(string(buf[:off]), "\n")
	if len(lines) == 0 {
		return false, nil
	}
	data, err := hex.DecodeString(strings.TrimSpace(lines[0]))
	if err != nil {
		return false, err
	}
	execHash, err := computeFileHash(execPath)
	if err != nil {
		return false, err
	}
	mid, _ := getMachineIDZeroAlloc()
	hasher := blake3.New()
	hasher.Write(execHash[:])
	hasher.Write([]byte(mid))
	sum := hasher.Sum(nil)
	var local [32]byte
	copy(local[:], sum[:32])
	if !bytes.Equal(local[:], data[:32]) {
		return false, nil
	}
	copy(combinedSeal[:], local[:])
	allowedMu.Lock()
	allowed = make(map[string]struct{})
	for i := 1; i < len(lines); i++ {
		ln := strings.TrimSpace(lines[i])
		if ln == "" {
			continue
		}
		parts := strings.SplitN(ln, "\t", 2)
		if len(parts) >= 1 {
			allowed[parts[0]] = struct{}{}
		}
	}
	allowedMu.Unlock()
	sealed = true
	return true, nil
}

func GetCombined() [32]byte {
	return combinedSeal
}

func IsAllowedHex(h string) bool {
	allowedMu.RLock()
	defer allowedMu.RUnlock()
	if allowed == nil {
		return false
	}
	_, ok := allowed[h]
	return ok
}

//go:build !linux
// +build !linux

package seal

import (
    "bytes"
    "encoding/hex"
    "io"
    "io/fs"
    "os"
    "path/filepath"
    "strings"
    "sync"
    "sync/atomic"

    "github.com/zeebo/blake3"
)

var sealed bool
var combinedSeal [32]byte
var allowed map[string]struct{}
var allowedMu sync.RWMutex
var journalBuf []byte
var journalPos uint32
var journalMu sync.Mutex
var stateMu sync.RWMutex
var globalState [32]byte
var decoy atomic.Bool
var machineIDPath = "/etc/machine-id"

func init() {
    journalBuf = make([]byte, 1<<20)
}

func getExecPath() (string, error) {
    path, err := os.Executable()
    if err != nil {
        return "", err
    }
    return filepath.Clean(path), nil
}

func getMachineIDZeroAlloc() (string, error) {
    data, err := os.ReadFile(machineIDPath)
    if err != nil {
        return "", err
    }
    return strings.TrimSpace(string(data)), nil
}

func computeFileHash(path string) ([32]byte, error) {
    var out [32]byte
    f, err := os.Open(path)
    if err != nil {
        return out, err
    }
    defer f.Close()
    hasher := blake3.New()
    if _, err := io.Copy(hasher, f); err != nil {
        return out, err
    }
    sum := hasher.Sum(nil)
    copy(out[:], sum[:32])
    return out, nil
}

func writeAll(f *os.File, b []byte) error {
    for len(b) > 0 {
        n, err := f.Write(b)
        if err != nil {
            return err
        }
        b = b[n:]
    }
    return nil
}

func setImmutable(path string) error {
    return nil
}

func isStagingMode() bool {
    return os.Getenv("FZ_STAGING") == "1"
}

func MachineID() (string, error) {
    return getMachineIDZeroAlloc()
}

func debuggerPresent() bool {
    return os.Getenv("FZ_DEBUGGER_SIMULATE") == "1"
}

func walkProjectFiles(root string, visit func(path string, info os.FileInfo, err error) error) error {
    root = filepath.Clean(root)
    info, err := os.Lstat(root)
    if err != nil {
        return visit(root, nil, err)
    }
    if err := visit(root, info, nil); err != nil {
        return err
    }
    return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
        if err != nil {
            return visit(path, nil, err)
        }
        if path == root {
            return nil
        }
        info, err := d.Info()
        if err != nil {
            return visit(path, nil, err)
        }
        if err := visit(path, info, nil); err != nil {
            return err
        }
        return nil
    })
}

func zeroizeRegion(data []byte) {
    if len(data) == 0 {
        return
    }
    for i := range data {
        data[i] = 0
    }
}

func triggerDecoy() {
    if decoy.Load() {
        return
    }
    decoy.Store(true)
    journalMu.Lock()
    atomic.StoreUint32(&journalPos, 0)
    if len(journalBuf) > 0 {
        zeroizeRegion(journalBuf)
    }
    journalMu.Unlock()
    stateMu.Lock()
    zeroizeRegion(globalState[:])
    zeroizeRegion(combinedSeal[:])
    stateMu.Unlock()
}

func resetSealState() {
    journalMu.Lock()
    atomic.StoreUint32(&journalPos, 0)
    if len(journalBuf) > 0 {
        zeroizeRegion(journalBuf)
    }
    journalMu.Unlock()
    stateMu.Lock()
    sealed = false
    zeroizeRegion(globalState[:])
    zeroizeRegion(combinedSeal[:])
    stateMu.Unlock()
    allowedMu.Lock()
    allowed = nil
    allowedMu.Unlock()
    decoy.Store(false)
}

func UpdateGlobalState(data []byte) {
    stateMu.Lock()
    hasher := blake3.New()
    hasher.Write(globalState[:])
    hasher.Write(data)
    sum := hasher.Sum(nil)
    copy(globalState[:], sum[:32])
    stateMu.Unlock()
}

func JournalEvent(data []byte) {
    if len(journalBuf) == 0 {
        return
    }
    journalMu.Lock()
    pos := int(atomic.LoadUint32(&journalPos))
    if pos+len(data) <= len(journalBuf) {
        copy(journalBuf[pos:], data)
        pos += len(data)
    } else {
        n := copy(journalBuf[pos:], data)
        copy(journalBuf, data[n:])
        pos = len(data) - n
    }
    atomic.StoreUint32(&journalPos, uint32(pos))
    journalMu.Unlock()
    UpdateGlobalState(data)
}

func Seal() error {
    if debuggerPresent() && !isStagingMode() {
        triggerDecoy()
        return nil
    }
    execPath, err := getExecPath()
    if err != nil {
        return err
    }
    execHash, err := computeFileHash(execPath)
    if err != nil {
        return err
    }
    mid, err := getMachineIDZeroAlloc()
    if err != nil {
        return err
    }
    hasher := blake3.New()
    hasher.Write(execHash[:])
    hasher.Write([]byte(mid))
    sum := hasher.Sum(nil)
    stateMu.Lock()
    copy(combinedSeal[:], sum[:32])
    stateMu.Unlock()
    dir := filepath.Dir(execPath)
    sealPath := filepath.Join(dir, ".fz_seal")
    f, err := os.OpenFile(sealPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
    if err != nil {
        return err
    }
    defer f.Close()
    hexb := make([]byte, hex.EncodedLen(len(sum)))
    hex.Encode(hexb, sum)
    hexb = append(hexb, '\n')
    if err := writeAll(f, hexb); err != nil {
        return err
    }
    root := filepath.Dir(execPath)
    if err := walkProjectFiles(root, func(path string, info os.FileInfo, err error) error {
        if err != nil {
            return err
        }
        if info.IsDir() {
            return nil
        }
        h, err := computeFileHash(path)
        if err != nil {
            return err
        }
        JournalEvent(h[:])
        hb := make([]byte, hex.EncodedLen(len(h)))
        hex.Encode(hb, h[:])
        hb = append(hb, '\t')
        hb = append(hb, []byte(path)...)
        hb = append(hb, '\n')
        if err := writeAll(f, hb); err != nil {
            return err
        }
        return nil
    }); err != nil {
        return err
    }
    if !isStagingMode() {
        if err := setImmutable(sealPath); err != nil {
            return err
        }
    }
    stateMu.Lock()
    sealed = true
    stateMu.Unlock()
    return nil
}

func Verify() (bool, error) {
    stateMu.RLock()
    if sealed {
        stateMu.RUnlock()
        return true, nil
    }
    stateMu.RUnlock()
    if debuggerPresent() && !isStagingMode() {
        triggerDecoy()
        return true, nil
    }
    execPath, err := getExecPath()
    if err != nil {
        return false, err
    }
    dir := filepath.Dir(execPath)
    sealPath := filepath.Join(dir, ".fz_seal")
    buf, err := os.ReadFile(sealPath)
    if err != nil {
        return false, err
    }
    lines := strings.Split(string(buf), "\n")
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
    mid, err := getMachineIDZeroAlloc()
    if err != nil {
        return false, err
    }
    hasher := blake3.New()
    hasher.Write(execHash[:])
    hasher.Write([]byte(mid))
    sum := hasher.Sum(nil)
    var local [32]byte
    copy(local[:], sum[:32])
    if !bytes.Equal(local[:], data[:32]) {
        if debuggerPresent() && !isStagingMode() {
            triggerDecoy()
            return true, nil
        }
        return false, nil
    }
    stateMu.Lock()
    copy(combinedSeal[:], local[:])
    sealed = true
    stateMu.Unlock()
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
    stateMu.RLock()
    out := combinedSeal
    stateMu.RUnlock()
    return out
}

func getGlobalState() [32]byte {
    stateMu.RLock()
    out := globalState
    stateMu.RUnlock()
    return out
}

func IsDecoyMode() bool {
    return decoy.Load()
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

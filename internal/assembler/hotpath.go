package assembler

import (
	"context"
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"unsafe"
)

const (
	maxHotArgs   = 32
	maxPathBytes = 4096
	errAsmFail   = 71
	errAsmPath   = 72
	errAsmTool   = 73
)

type pathBuf struct {
	data [maxPathBytes]byte
	n    int
}

type argBuf struct {
	data [128]byte
	n    int
}

type hotArgVec struct {
	args [maxHotArgs]argBuf
	n    int
}

func (p *pathBuf) reset() {
	p.n = 0
}

func (p *pathBuf) appendString(s string) bool {
	if p.n+len(s) > len(p.data) {
		return false
	}
	copy(p.data[p.n:], s)
	p.n += len(s)
	return true
}

func (p *pathBuf) string() string {
	return unsafe.String(&p.data[0], p.n)
}

func (a *argBuf) appendString(s string) bool {
	if a.n+len(s) > len(a.data) {
		return false
	}
	copy(a.data[a.n:], s)
	a.n += len(s)
	return true
}

func (a *argBuf) string() string {
	return unsafe.String(&a.data[0], a.n)
}

func (v *hotArgVec) reset() {
	v.n = 0
}

func (v *hotArgVec) pushLiteral(s string) bool {
	if v.n >= maxHotArgs {
		return false
	}
	if !v.args[v.n].appendString(s) {
		return false
	}
	v.n++
	return true
}

func (v *hotArgVec) pushPath(p *pathBuf) bool {
	if v.n >= maxHotArgs {
		return false
	}
	if !v.args[v.n].appendString(p.string()) {
		return false
	}
	v.n++
	return true
}

func cleanPathHot(raw string, dst *pathBuf) bool {
	dst.reset()
	if raw == "" {
		return false
	}
	if !dst.appendString(raw) {
		return false
	}
	cleaned := filepath.Clean(dst.string())
	dst.reset()
	return dst.appendString(cleaned)
}

func statFileHot(p *pathBuf) bool {
	info, err := os.Stat(p.string())
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func hotTesting() bool {
	return flag.Lookup("test.v") != nil
}

func writeErrHot(code byte) {
	var buf [16]byte
	buf[0] = 'f'
	buf[1] = 'z'
	buf[2] = ':'
	buf[3] = 'a'
	buf[4] = 's'
	buf[5] = 'm'
	buf[6] = ':'
	i := 7
	if code >= 100 {
		buf[i] = byte('0' + code/100)
		i++
		code %= 100
	}
	if code >= 10 {
		buf[i] = byte('0' + code/10)
		i++
		code %= 10
	}
	buf[i] = byte('0' + code)
	i++
	buf[i] = '\n'
	_, _ = os.Stderr.Write(buf[:i+1])
}

func failHot(code byte) {
	writeErrHot(code)
	os.Exit(int(code))
}

func runHotNASM(ctx context.Context, nasm *pathBuf, vec *hotArgVec) error {
	if vec.n == 0 || nasm == nil || nasm.n == 0 {
		if hotTesting() {
			return errHotTool
		}
		failHot(errAsmTool)
	}
	var argv [maxHotArgs]string
	for i := 0; i < vec.n; i++ {
		argv[i] = vec.args[i].string()
	}
	cmd := exec.CommandContext(ctx, nasm.string(), argv[:vec.n]...)
	cmd.Stdout = nil
	cmd.Stderr = nil
	runErr := cmd.Run()
	if runErr != nil {
		if hotTesting() {
			return errHotAsm
		}
		failHot(errAsmFail)
	}
	return nil
}

var (
	errHotAsm  = errHot{code: errAsmFail}
	errHotPath = errHot{code: errAsmPath}
	errHotTool = errHot{code: errAsmTool}
)

type errHot struct {
	code byte
}

func (e errHot) Error() string {
	return "asm hot fail"
}

func assembleNASMHot(ctx context.Context, src, obj string, debug, verbose bool) error {
	if verbose || debug {
		return assembleNASMSlow(ctx, src, obj, debug, verbose)
	}
	var srcPB pathBuf
	var objPB pathBuf
	if !cleanPathHot(src, &srcPB) {
		if hotTesting() {
			return errHotPath
		}
		failHot(errAsmPath)
	}
	if !cleanPathHot(obj, &objPB) {
		if hotTesting() {
			return errHotPath
		}
		failHot(errAsmPath)
	}
	if len(src) > maxPathBytes || len(obj) > maxPathBytes {
		if hotTesting() {
			return errHotPath
		}
		rejectPathLen()
	}
	if !statFileHot(&srcPB) {
		if hotTesting() {
			return errHotPath
		}
		failHot(errAsmPath)
	}
	var nasmPB pathBuf
	if !getToolPathHot("nasm", &nasmPB) {
		if hotTesting() {
			return errHotTool
		}
		failHot(errAsmTool)
	}
	var vec hotArgVec
	vec.reset()
	if !vec.pushLiteral("-fbin") {
		return errHotPath
	}
	if !vec.pushPath(&srcPB) {
		return errHotPath
	}
	if !vec.pushLiteral("-o") {
		return errHotPath
	}
	if !vec.pushPath(&objPB) {
		return errHotPath
	}
	for i := 0; i < len(AsmFlags); i++ {
		if !vec.pushLiteral(AsmFlags[i]) {
			return errHotPath
		}
	}
	if err := runHotNASM(ctx, &nasmPB, &vec); err != nil {
		return err
	}
	writeFlatAssembled(&objPB)
	return nil
}

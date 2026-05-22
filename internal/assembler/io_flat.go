package assembler

import "syscall"

func writeFlatAssembled(p *pathBuf) {
	if p == nil || p.n == 0 {
		return
	}
	if p.n > maxPathBytes {
		rejectPathLen()
	}
	var msg [4224]byte
	prefix := []byte("Assembled flat binary: ")
	n := copy(msg[:], prefix)
	n += copy(msg[n:], p.data[:p.n])
	msg[n] = '\n'
	_, _ = syscall.Write(1, msg[:n+1])
}

func WriteFlatAssembledNotice(path string) {
	var pb pathBuf
	if !cleanPathHot(path, &pb) {
		return
	}
	writeFlatAssembled(&pb)
}

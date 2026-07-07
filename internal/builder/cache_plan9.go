package builder

import (
	"os"
)

func joinPathPlan9Fallback(base, name string) string {
	var pb pathBuffer
	pb.appendString(base)
	if len(base) > 0 && base[len(base)-1] != byte(os.PathSeparator) {
		pb.appendByte(byte(os.PathSeparator))
	}
	pb.appendString(name)
	return pb.String()
}

func buildCacheKeyPlan9Fallback(hash string, debug bool, mode string) string {
	var pb pathBuffer
	pb.appendString(hash)
	pb.appendByte('_')
	if debug {
		pb.appendByte('1')
	} else {
		pb.appendByte('0')
	}
	pb.appendByte('_')
	pb.appendString(mode)
	return pb.String()
}

func cacheEntryPathPlan9Fallback(dir, key string) string {
	var pb pathBuffer
	pb.appendString(dir)
	if len(dir) > 0 && dir[len(dir)-1] != byte(os.PathSeparator) {
		pb.appendByte(byte(os.PathSeparator))
	}
	pb.appendString(key)
	return pb.String()
}




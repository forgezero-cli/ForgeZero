//go:build !amd64 && !arm64

package builder

func pathBuffer_appendStringPlan9(p *pathBuffer, s string) {
	p.appendString(s)
}

func pathBuffer_appendBytePlan9(p *pathBuffer, b byte) {
	p.appendByte(b)
}

func pathBuffer_appendBytesPlan9(p *pathBuffer, b []byte) {
	p.appendBytes(b)
}

func joinPathPlan9(base, name string) string {
	return joinPathPlan9Fallback(base, name)
}

func buildCacheKeyPlan9(hash string, debug bool, mode string) string {
	return buildCacheKeyPlan9Fallback(hash, debug, mode)
}

func cacheEntryPathPlan9(dir, key string) string {
	return cacheEntryPathPlan9Fallback(dir, key)
}



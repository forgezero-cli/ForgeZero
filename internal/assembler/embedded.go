package assembler

import "embed"

//go:embed assets/*
var assetFS embed.FS

func loadEmbeddedTool(tool string) ([]byte, bool) {
	name := embeddedAssetName(tool)
	path := "assets/" + name
	data, err := assetFS.ReadFile(path)
	if err != nil {
		return nil, false
	}
	if len(data) < 64 {
		return nil, false
	}
	return data, true
}

package shell

type BuildState struct {
	Mode          string
	Format        string
	Strict        bool
	Sanitize      bool
	Verbose       bool
	Debug         bool
	NoCache       bool
	NoSymbolCheck bool
	KeepObj       bool
	LdScript      string
	TextAddr      string
	Out           string
	SourcePath    string
	SourceType    string
}

func DefaultState() *BuildState {
	return &BuildState{
		Mode:          "auto",
		Format:        "elf64",
		Strict:        false,
		Sanitize:      true,
		Verbose:       false,
		Debug:         false,
		NoCache:       false,
		NoSymbolCheck: false,
		KeepObj:       false,
		LdScript:      "",
		TextAddr:      "",
		Out:           "",
		SourcePath:    ".",
		SourceType:    "dir",
	}
}

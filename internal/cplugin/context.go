package cplugin

type GoContext struct {
	PluginPath string
	ConfigPath string
	SourcePath string
	DirPath    string
	OutBin     string
	OutObj     string
	BuildType  string
	Target     string
	Toolchain  string
	Mode       string
	CcFlags    string
	LdFlags    string
	Format     string
	Isolation  string
	SourceDirs []string
}

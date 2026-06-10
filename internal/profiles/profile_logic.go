package profiles

import "runtime"

type BuildProfile struct {
	Name string
}

func normalizeName(s string) string {
	if len(s) == 0 {
		return "balanced"
	}
	lower := s
	b := []byte(lower)
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		}
	}
	lower = string(b)

	switch lower {
	case "performance", "perf", "max", "full", "powered":
		return "performance"
	case "balanced", "normal", "default", "mid":
		return "balanced"
	case "power-saver", "powersaver", "powersave", "eco", "min", "low", "power_saver":
		return "power-saver"
	default:
		return "balanced"
	}
}

func ParseUserProfile(s string) BuildProfile {
	return BuildProfile{Name: normalizeName(s)}
}

func (p BuildProfile) DefaultJobs() int {
	cpu := runtime.NumCPU()
	if cpu <= 0 {
		cpu = 1
	}
	switch p.Name {
	case "performance":
		return cpu
	case "power-saver":
		return 1
	default: // balanced
		if cpu <= 1 {
			return 1
		}
		return cpu / 2
	}
}

func (p BuildProfile) EffectiveJobs(requestedJobs int) int {
	if requestedJobs > 0 {
		return requestedJobs
	}
	return p.DefaultJobs()
}

func (p BuildProfile) OptimizationFlag() string {
	switch p.Name {
	case "performance":
		return "-O3"
	case "power-saver":
		return "-Os"
	default: // balanced
		return "-O2"
	}
}

package version

import (
	"runtime"
	"runtime/debug"
)

var (
	RecorderVersion string
	GoVersion       string
)

func init() {
	if GoVersion == "" {
		GoVersion = runtime.Version()
	}
	if RecorderVersion == "" {
		RecorderVersion = buildRevision()
	}
}

func buildRevision() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	rev := ""
	dirty := false
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			rev = s.Value
			if len(rev) > 7 {
				rev = rev[:7]
			}
		case "vcs.modified":
			dirty = s.Value == "true"
		}
	}
	if rev == "" {
		return "dev"
	}
	if dirty {
		return rev + "-dirty"
	}

	return rev
}

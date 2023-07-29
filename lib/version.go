package router

import (
	"fmt"
	"runtime/debug"
)

// VersionInfo returns human-readable version information in a format suitable
// for concatenation with other messages.
func VersionInfo() (v string) {
	v = "(version info unavailable)"

	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}

	rev, commitTime, dirty := buildSettings(bi.Settings)
	if rev == "" {
		return
	}

	commitTimeOrDirty := "dirty"
	if dirty == "false" {
		commitTimeOrDirty = commitTime
	}
	return fmt.Sprintf("built from commit %.8s (%s) using %s", rev, commitTimeOrDirty, bi.GoVersion)
}

func buildSettings(bs []debug.BuildSetting) (rev, commitTime, dirty string) {
	for _, b := range bs {
		switch b.Key {
		case "vcs.modified":
			dirty = b.Value
		case "vcs.revision":
			rev = b.Value
		case "vcs.time":
			commitTime = b.Value
		}
	}
	return
}

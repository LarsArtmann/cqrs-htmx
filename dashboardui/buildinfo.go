package dashboardui

import (
	"reflect"
	"runtime/debug"
)

// vcsBuildInfo carries the VCS stamp of the binary the dashboard is compiled
// into. Fields are empty when the binary was built without VCS metadata
// (buildvcs=false, test binaries, some CI archives).
type vcsBuildInfo struct {
	Revision string
	Time     string
	Modified string
}

// ownModulePath reports the module path of the package this code was compiled
// from, derived from its own import path via reflection. Unlike the
// modulePath constant it stays truthful in forks: a binary compiled from a
// fork reports the fork's module path. Falls back to the constant when
// reflection yields nothing (not expected for compiled code).
func ownModulePath() string {
	if pkgPath := reflect.TypeOf(Config{}).PkgPath(); pkgPath != "" {
		return pkgPath
	}

	return modulePath
}

// resolveModuleVersion reports the dashboardui module's version as recorded
// in the binary's build info: the main module's version when developing or
// testing in-module ("(devel)"), the dependency's version when a consumer
// binary links it, a replacing module's version when a versioned replace is
// in effect, and empty for local directory replaces. A nil BuildInfo yields
// empty.
func resolveModuleVersion(info *debug.BuildInfo, module string) string {
	if info == nil {
		return ""
	}

	if info.Main.Path == module {
		return info.Main.Version
	}

	for _, dep := range info.Deps {
		if dep.Path != module {
			continue
		}

		if dep.Replace != nil && dep.Replace.Version != "" {
			return dep.Replace.Version
		}

		return dep.Version
	}

	return ""
}

// resolveVCS extracts the vcs.* build settings. They describe the MAIN module
// — the binary an operator is talking to — which is what a version endpoint
// should identify, independent of which module version dashboardui itself is.
func resolveVCS(settings []debug.BuildSetting) vcsBuildInfo {
	var stamp vcsBuildInfo

	for _, setting := range settings {
		switch setting.Key {
		case "vcs.revision":
			stamp.Revision = setting.Value
		case "vcs.time":
			stamp.Time = setting.Value
		case "vcs.modified":
			stamp.Modified = setting.Value
		}
	}

	return stamp
}

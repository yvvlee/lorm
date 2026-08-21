package main

import "runtime/debug"

const developmentVersion = "dev"

func currentVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok || info.Main.Version == "" || info.Main.Version == "(devel)" {
		return developmentVersion
	}
	return info.Main.Version
}

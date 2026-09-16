package main

import (
	"fmt"
	"go/ast"
	"go/build/constraint"
	"path/filepath"
	"strings"
)

// Preserve explicit constraints and filename constraints without changing the
// companion filename. Otherwise model_linux.go would produce an unrestricted file.
func sourceBuildConstraint(path string, file *ast.File) (string, error) {
	var modern string
	var legacy []string
	for _, group := range file.Comments {
		if group.Pos() >= file.Package {
			break
		}
		for _, comment := range group.List {
			if constraint.IsGoBuild(comment.Text) {
				if modern != "" {
					return "", fmt.Errorf("%s: multiple //go:build lines", path)
				}
				modern = comment.Text
			} else if constraint.IsPlusBuild(comment.Text) {
				legacy = append(legacy, comment.Text)
			}
		}
	}
	lines := legacy
	if modern != "" {
		lines = []string{modern}
	}
	var result constraint.Expr
	for _, line := range lines {
		expr, err := constraint.Parse(line)
		if err != nil {
			return "", fmt.Errorf("%s: invalid build constraint: %w", path, err)
		}
		result = andBuildConstraint(result, expr)
	}
	for _, tag := range filenameBuildTags(filepath.Base(path)) {
		result = andBuildConstraint(result, &constraint.TagExpr{Tag: tag})
	}
	if result == nil {
		return "", nil
	}
	return result.String(), nil
}

func andBuildConstraint(left, right constraint.Expr) constraint.Expr {
	if left == nil {
		return right
	}
	return &constraint.AndExpr{X: left, Y: right}
}

func filenameBuildTags(name string) []string {
	name, _, _ = strings.Cut(name, ".")
	index := strings.IndexByte(name, '_')
	if index < 0 {
		return nil
	}
	parts := strings.Split(strings.TrimSuffix(name[index:], "_test"), "_")
	n := len(parts)
	if n >= 2 && knownOS(parts[n-2]) && knownArch(parts[n-1]) {
		return parts[n-2:]
	}
	if knownOS(parts[n-1]) || knownArch(parts[n-1]) {
		return parts[n-1:]
	}
	return nil
}

// These names follow Go's internal/syslist, including historical targets whose
// filename constraints remain meaningful. Tests compare matching with go/build.
func knownOS(name string) bool {
	switch name {
	case "aix", "android", "darwin", "dragonfly", "freebsd", "hurd", "illumos", "ios", "js", "linux", "nacl", "netbsd", "openbsd", "plan9", "solaris", "wasip1", "windows", "zos":
		return true
	}
	return false
}

func knownArch(name string) bool {
	switch name {
	case "386", "amd64", "amd64p32", "arm", "armbe", "arm64", "arm64be", "loong64", "mips", "mipsle", "mips64", "mips64le", "mips64p32", "mips64p32le", "ppc", "ppc64", "ppc64le", "riscv", "riscv64", "s390", "s390x", "sparc", "sparc64", "wasm":
		return true
	}
	return false
}

package common

import (
	"fmt"
	"runtime"
	"time"
)

const (
	// Application information
	ProjectName    = "Enhanced DCA Bot"
	ProjectVersion = "2.0.0"
	ProjectRepo    = "github.com/ducminhle1904/crypto-dca-bot"
	
	// Build information - these would normally be set during build via -ldflags
	BuildDate      = "2024-01-01"  // Will be overridden during build
	BuildCommit    = "dev"         // Will be overridden during build
)

// VersionInfo contains version and build information
type VersionInfo struct {
	ProjectName   string `json:"project_name"`
	Version       string `json:"version"`
	BuildDate     string `json:"build_date"`
	BuildCommit   string `json:"build_commit"`
	GoVersion     string `json:"go_version"`
	Architecture  string `json:"architecture"`
	Repository    string `json:"repository"`
}

// GetVersionInfo returns complete version information
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		ProjectName:   ProjectName,
		Version:       ProjectVersion,
		BuildDate:     BuildDate,
		BuildCommit:   BuildCommit,
		GoVersion:     runtime.Version(),
		Architecture:  runtime.GOOS + "/" + runtime.GOARCH,
		Repository:    ProjectRepo,
	}
}

// PrintVersion prints version information in a formatted way
func PrintVersion(appName string) {
	info := GetVersionInfo()
	
	fmt.Printf("%s v%s\n", appName, info.Version)
	fmt.Printf("Build: %s (%s)\n", info.BuildCommit, info.BuildDate)
	fmt.Printf("Go: %s (%s)\n", info.GoVersion, info.Architecture)
}

// PrintDetailedVersion prints detailed version information
func PrintDetailedVersion(appName string) {
	info := GetVersionInfo()
	
	fmt.Printf("╔═══════════════════════════════════════╗\n")
	fmt.Printf("║           VERSION INFORMATION         ║\n")
	fmt.Printf("╠═══════════════════════════════════════╣\n")
	fmt.Printf("║ Application: %-24s ║\n", appName)
	fmt.Printf("║ Version:     %-24s ║\n", info.Version)
	fmt.Printf("║ Project:     %-24s ║\n", info.ProjectName)
	fmt.Printf("║ Repository:  %-24s ║\n", info.Repository)
	fmt.Printf("║ Build Date:  %-24s ║\n", info.BuildDate)
	fmt.Printf("║ Build Hash:  %-24s ║\n", info.BuildCommit)
	fmt.Printf("║ Go Version:  %-24s ║\n", info.GoVersion)
	fmt.Printf("║ Platform:    %-24s ║\n", info.Architecture)
	fmt.Printf("║ Build Time:  %-24s ║\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("╚═══════════════════════════════════════╝\n")
}

// GetShortVersion returns a short version string
func GetShortVersion() string {
	return ProjectVersion
}

// GetFullVersion returns a full version string with build info
func GetFullVersion() string {
	info := GetVersionInfo()
	return fmt.Sprintf("%s-%s (%s)", info.Version, info.BuildCommit, info.BuildDate)
}

// IsDevBuild returns true if this is a development build
func IsDevBuild() bool {
	return BuildCommit == "dev"
}

// GetBuildInfo returns build-specific information
func GetBuildInfo() map[string]string {
	return map[string]string{
		"version":      ProjectVersion,
		"build_date":   BuildDate,
		"build_commit": BuildCommit,
		"go_version":   runtime.Version(),
		"platform":     runtime.GOOS + "/" + runtime.GOARCH,
	}
}

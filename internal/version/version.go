package version

import (
	"fmt"
	"os"
	"strings"
	"time"
)

var (
	// Version wird beim Build gesetzt
	Version = "1.0.0"

	// BuildTime wird beim Build gesetzt
	BuildTime = time.Now().Format(time.RFC3339)

	// GitCommit wird beim Build gesetzt
	GitCommit = "unknown"

	// GoVersion wird beim Build gesetzt
	GoVersion = "unknown"
)

// Info gibt Versionsinformationen zurück
func Info() map[string]string {
	return map[string]string{
		"version":    Version,
		"build_time": BuildTime,
		"git_commit": GitCommit,
		"go_version": GoVersion,
	}
}

// String gibt eine formatierte Versionsstring zurück
func String() string {
	return fmt.Sprintf("WebCrawler v%s (Build: %s, Commit: %s)",
		Version, BuildTime, GitCommit)
}

// PrintVersion druckt Versionsinformationen
func PrintVersion() {
	fmt.Printf("🌐 %s\n", String())
	fmt.Printf("📅 Build Time: %s\n", BuildTime)
	fmt.Printf("🔗 Git Commit: %s\n", GitCommit)
	fmt.Printf("⚡ Go Version: %s\n", GoVersion)
}

// IsDevelopment prüft ob es sich um eine Development-Version handelt
func IsDevelopment() bool {
	return strings.Contains(Version, "dev") ||
		strings.Contains(Version, "alpha") ||
		strings.Contains(Version, "beta")
}

// GetVersionFromFile liest Version aus VERSION Datei
func GetVersionFromFile() string {
	data, err := os.ReadFile("VERSION")
	if err != nil {
		return Version
	}
	return strings.TrimSpace(string(data))
}

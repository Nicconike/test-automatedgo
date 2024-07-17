package main

import (
	"fmt"
	"log"
	"os/exec"

	"github.com/Nicconike/goautomate/pkg"
)

func main() {
	// Get the latest Go version
	latestVersion, err := pkg.GetLatestVersion()
	if err != nil {
		log.Fatalf("Error getting latest version: %v", err)
	}
	fmt.Printf("Latest Go version: %s\n", latestVersion)

	// Get current version
	currentVersion, err := pkg.GetCurrentVersion("tests/Dockerfile", "")
	if err != nil {
		log.Fatalf("Error getting current version: %v", err)
	}
	fmt.Printf("Current Go version: %s\n", currentVersion)

	// Check if update is needed
	if !pkg.IsNewer(latestVersion, currentVersion) {
		fmt.Println("Already on the latest version. No update needed.")
		return
	}

	// Download the latest Go version
	err = pkg.DownloadGo(latestVersion, "", "")
	if err != nil {
		log.Fatalf("Error downloading Go: %v", err)
	}
	fmt.Println("Successfully downloaded new Go version")

	// Commit and push changes
	err = commitAndPush(latestVersion)
	if err != nil {
		log.Fatalf("Error committing and pushing changes: %v", err)
	}

	fmt.Println("Successfully committed and pushed new Go version to GitHub")
}

func commitAndPush(version string) error {
	commands := []struct {
		name string
		args []string
	}{
		{"git", []string{"config", "--local", "user.name", "github-actions[bot]"}},
		{"git", []string{"config", "--local", "user.email", "41898282+github-actions[bot]@users.noreply.github.com"}},
		{"git", []string{"add", "."}},
		{"git", []string{"commit", "-m", fmt.Sprintf("Update Go version to %s", version)}},
		{"git", []string{"push"}},
	}

	for _, cmd := range commands {
		if err := exec.Command(cmd.name, cmd.args...).Run(); err != nil {
			return fmt.Errorf("error running '%s %v': %v", cmd.name, cmd.args, err)
		}
	}

	return nil
}

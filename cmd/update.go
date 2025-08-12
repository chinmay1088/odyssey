package cmd

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName     string `json:"tag_name"`
	Name        string `json:"name"`
	Body        string `json:"body"`
	PublishedAt string `json:"published_at"`
	ZipballURL  string `json:"zipball_url"`
	TarballURL  string `json:"tarball_url"`
}

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update Odyssey to the latest version",
	Long: `Check for and build the latest version of Odyssey wallet from source.
	
This command will:
  • Check GitHub releases for the latest version
  • Compare with your current version (` + version + `)
  • Download source code and build automatically if newer version exists
  • Backup current version before updating

Examples:
  odyssey update           # Check and build latest version
  odyssey update --check   # Only check for updates, don't install`,
	RunE: runUpdate,
}

var (
	checkOnly bool
)

func init() {
	updateCmd.Flags().BoolVar(&checkOnly, "check", false, "Only check for updates, don't install")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	fmt.Println("🔄 Checking for Odyssey updates...")
	fmt.Printf("📦 Current version: %s\n", color.CyanString("v"+version))
	fmt.Println()

	// Verify Go is installed
	if err := verifyGoDependencies(); err != nil {
		return err
	}

	// Get latest release from GitHub
	latest, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	// Compare versions
	currentVer := "v" + version
	latestVer := latest.TagName

	if latestVer == currentVer {
		fmt.Printf("✅ You're running the latest version (%s)\n", color.GreenString(currentVer))
		return nil
	}

	// Check if we have a newer version
	if isNewerVersion(latestVer, currentVer) {
		fmt.Printf("🚀 New version available: %s\n", color.GreenString(latestVer))
		fmt.Printf("📅 Released: %s\n", formatReleaseDate(latest.PublishedAt))

		if latest.Body != "" {
			fmt.Println("\n📝 Release Notes:")
			fmt.Println(latest.Body)
		}
		fmt.Println()

		if checkOnly {
			fmt.Printf("💡 Run '%s' to build and install the update\n", color.YellowString("odyssey update"))
			return nil
		}

		// Ask for confirmation
		if !confirmUpdate(latestVer) {
			fmt.Println("❌ Update cancelled")
			return nil
		}

		// Perform update by building from source
		return performSourceUpdate(latest)
	} else {
		fmt.Printf("ℹ️  You're running a newer version (%s) than the latest release (%s)\n",
			color.YellowString(currentVer), color.CyanString(latestVer))
		return nil
	}
}

func verifyGoDependencies() error {
	// Check if Go is installed
	_, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("go compiler not found. Please install Go from https://golang.org/dl/")
	}

	// Check if Git is installed (for Go modules)
	_, err = exec.LookPath("git")
	if err != nil {
		return fmt.Errorf("git not found. Please install Git from https://git-scm.com/download")
	}

	// Verify Go version
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to check Go version: %w", err)
	}

	fmt.Printf("🔧 Build environment: %s", color.CyanString(strings.TrimSpace(string(output))))
	
	return nil
}

func getLatestRelease() (*GitHubRelease, error) {
	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Get("https://api.github.com/repos/chinmay1088/odyssey/releases/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

func isNewerVersion(latest, current string) bool {
    latestParts := strings.Split(strings.TrimPrefix(latest, "v"), ".")
    currentParts := strings.Split(strings.TrimPrefix(current, "v"), ".")

    // Normalize length so both have the same number of segments
    maxLen := len(latestParts)
    if len(currentParts) > maxLen {
        maxLen = len(currentParts)
    }

    for len(latestParts) < maxLen {
        latestParts = append(latestParts, "0")
    }
    for len(currentParts) < maxLen {
        currentParts = append(currentParts, "0")
    }

    for i := 0; i < maxLen; i++ {
        latestNum, _ := strconv.Atoi(strings.SplitN(latestParts[i], "-", 2)[0])
        currentNum, _ := strconv.Atoi(strings.SplitN(currentParts[i], "-", 2)[0])
        if latestNum > currentNum {
            return true
        }
        if latestNum < currentNum {
            return false
        }
    }

    return false
}

func formatReleaseDate(dateStr string) string {
	t, err := time.Parse(time.RFC3339, dateStr)
	if err != nil {
		return dateStr
	}
	return t.Format("January 2, 2006")
}

func confirmUpdate(newVersion string) bool {
	fmt.Printf("🔧 Build and install %s from source? This will replace your current installation (y/N): ", color.GreenString(newVersion))

	var response string
	fmt.Scanln(&response)

	response = strings.ToLower(strings.TrimSpace(response))
	return response == "y" || response == "yes"
}

func performSourceUpdate(release *GitHubRelease) error {
	fmt.Printf("⬇️  Downloading source code for %s...\n", release.TagName)

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "odyssey-update-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Download source code (use zipball for Windows compatibility)
	sourceURL := fmt.Sprintf("https://github.com/chinmay1088/odyssey/archive/refs/tags/%s.zip", release.TagName)
	zipPath := filepath.Join(tempDir, "source.zip")
	
	if err := downloadFile(sourceURL, zipPath); err != nil {
		return fmt.Errorf("failed to download source code: %w", err)
	}

	fmt.Println("📦 Extracting source code...")

	// Extract source code
	extractDir := filepath.Join(tempDir, "extracted")
	if err := extractZip(zipPath, extractDir); err != nil {
		return fmt.Errorf("failed to extract source code: %w", err)
	}

	// Find the source directory (GitHub creates a folder like odyssey-1.0.5)
	sourceDir, err := findSourceDirectory(extractDir)
	if err != nil {
		return fmt.Errorf("failed to locate source directory: %w", err)
	}

	fmt.Println("🔨 Building from source...")

	// Build the binary
	binaryPath, err := buildFromSource(sourceDir)
	if err != nil {
		return fmt.Errorf("failed to build from source: %w", err)
	}

	fmt.Println("🔧 Installing update...")

	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Backup current version
	backupPath := currentExe + ".backup"
	if err := copyFile(currentExe, backupPath); err != nil {
		fmt.Printf("⚠️  Warning: failed to create backup: %v\n", err)
	} else {
		fmt.Printf("💾 Backup created: %s\n", backupPath)
	}

	// Replace current executable
	if err := copyFile(binaryPath, currentExe); err != nil {
		return fmt.Errorf("failed to replace executable: %w", err)
	}

	fmt.Printf("✅ Successfully updated to %s!\n", color.GreenString(release.TagName))
	fmt.Printf("🔄 The new version is now active\n")

	// Verify installation
	fmt.Println("\n🔍 Verifying installation...")
	cmd := exec.Command(currentExe, "version")
	output, err := cmd.Output()
	if err == nil {
		fmt.Printf("✅ Verification successful: %s", string(output))
	} else {
		fmt.Printf("⚠️  Verification failed: %v\n", err)
		fmt.Printf("💡 You can restore the backup if needed: mv %s %s\n", backupPath, currentExe)
	}

	return nil
}

func findSourceDirectory(extractDir string) (string, error) {
	entries, err := os.ReadDir(extractDir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.HasPrefix(entry.Name(), "odyssey-") {
			return filepath.Join(extractDir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("source directory not found in extracted archive")
}

func buildFromSource(sourceDir string) (string, error) {
	// Change to source directory
	originalDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	defer os.Chdir(originalDir)

	if err := os.Chdir(sourceDir); err != nil {
		return "", fmt.Errorf("failed to change to source directory: %w", err)
	}

	// Initialize Go modules if go.mod doesn't exist
	if _, err := os.Stat("go.mod"); os.IsNotExist(err) {
		fmt.Println("  📝 Initializing Go modules...")
		cmd := exec.Command("go", "mod", "init", "odyssey")
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to initialize Go modules: %w", err)
		}

		cmd = exec.Command("go", "mod", "tidy")
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("failed to tidy Go modules: %w", err)
		}
	}

	// Download dependencies
	fmt.Println("  📥 Downloading dependencies...")
	cmd := exec.Command("go", "mod", "download")
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to download dependencies: %w", err)
	}

	// Build the binary
	fmt.Println("  🔨 Compiling binary...")
	binaryName := "odyssey"
	if runtime.GOOS == "windows" {
		binaryName = "odyssey.exe"
	}

	buildCmd := exec.Command("go", "build", "-ldflags", "-s -w", "-o", binaryName, ".")
	buildCmd.Env = append(os.Environ(), "CGO_ENABLED=0")
	
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("build failed: %w\nOutput: %s", err, string(output))
	}

	binaryPath := filepath.Join(sourceDir, binaryName)
	if _, err := os.Stat(binaryPath); os.IsNotExist(err) {
		return "", fmt.Errorf("binary was not created at expected path: %s", binaryPath)
	}

	return binaryPath, nil
}

func downloadFile(url, filepath string) error {
	client := &http.Client{Timeout: 5 * time.Minute}

	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func extractZip(src, dest string) error {
	reader, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer reader.Close()

	dest = filepath.Clean(dest)

	if err := os.MkdirAll(dest, 0755); err != nil {
		return err
	}

	for _, file := range reader.File {
		// Build full path and clean it
		path := filepath.Join(dest, file.Name)
		path = filepath.Clean(path)

		// Prevent directory traversal
		if !strings.HasPrefix(path, dest+string(os.PathSeparator)) && path != dest {
			// skip suspicious entry
			continue
		}

		// Handle directories
		if file.FileInfo().IsDir() {
			if err := os.MkdirAll(path, file.Mode()); err != nil {
				return err
			}
			continue
		}

		// Skip symlinks for safety
		if file.Mode()&os.ModeSymlink != 0 {
			// do not extract symlinks from archives
			continue
		}

		// Ensure parent directory exists
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			return err
		}

		// Open zipped file
		fileReader, err := file.Open()
		if err != nil {
			return err
		}

		// Create target file
		targetFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			fileReader.Close()
			return err
		}

		// Copy then close immediately to avoid leaking fds
		_, err = io.Copy(targetFile, fileReader)

		// close both in correct order
		closeErr1 := targetFile.Close()
		closeErr2 := fileReader.Close()

		if err != nil {
			return err
		}
		if closeErr1 != nil {
			return closeErr1
		}
		if closeErr2 != nil {
			return closeErr2
		}
	}

	return nil
}


func copyFile(src, dst string) error {
	// open source
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	// ensure dst dir exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	// write to temp file in same dir for atomic rename
	tmp := dst + ".tmp"
	out, err := os.OpenFile(tmp, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}

	_, err = io.Copy(out, in)
	if err2 := out.Close(); err == nil {
		err = err2
	}
	if err != nil {
		// remove tmp on error
		os.Remove(tmp)
		return err
	}

	// preserve mode from source if possible
	if fi, err := os.Stat(src); err == nil {
		os.Chmod(tmp, fi.Mode())
	}

	// atomic replace
	if err := os.Rename(tmp, dst); err != nil {
		// cleanup tmp if rename fails
		os.Remove(tmp)
		return err
	}

	return nil
}

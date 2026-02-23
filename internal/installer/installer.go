package installer

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"regexp"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/logger"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/platform"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/printer"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/software"
)

var (
	// ErrAllMethodsFailed is returned when all install methods fail
	ErrAllMethodsFailed = errors.New("all installation methods failed")

	// versionRegex extracts semantic version from command output
	versionRegex = regexp.MustCompile(`v?(\d+\.\d+\.\d+)`)
)

// Installer manages tool installations
type Installer struct {
	archiver platform.Archiver
	platform platform.Platform
	printer  *printer.Printer
}

// New creates a new installer instance
func New(plat platform.Platform, arch platform.Archiver, p *printer.Printer) *Installer {
	return &Installer{
		archiver: arch,
		platform: plat,
		printer:  p,
	}
}

// InstallSoftware installs software using the configuration-driven approach
func (i *Installer) InstallSoftware(ctx context.Context, sw *software.Software) InstallResult {
	log := logger.FromContext(ctx)

	log.Info("starting installation", "software", sw.ID, "displayName", sw.DisplayName)
	i.printer.Info(fmt.Sprintf("Installing %s", sw.DisplayName))

	// Ensure bin directory exists
	binDir := i.platform.GetBinDir()
	log.Debug("ensuring bin directory exists", "path", binDir)
	if err := i.platform.EnsureBinDir(); err != nil {
		log.Error("failed to create bin directory", "error", err, "path", binDir)
		return InstallResult{
			Error:      err,
			SoftwareID: sw.ID,
		}
	}

	// Ensure bin directory is in PATH
	inPath := i.platform.IsBinDirInPath()
	log.Debug("checking if bin directory is in PATH", "binDir", binDir, "inPath", inPath)
	if !inPath {
		log.Info("adding bin directory to PATH", "binDir", binDir)
		if err := i.platform.AddBinDirToPath(); err != nil {
			log.Error("failed to add bin directory to PATH", "error", err)
			return InstallResult{
				Error:      err,
				SoftwareID: sw.ID,
			}
		}
	}

	// Check if already installed
	log.Debug("checking if software is already installed", "software", sw.ID, "binaryName", sw.GetBinaryName(i.platform.OS()))
	if i.isInstalled(sw) {
		version := i.getVersion(sw)
		log.Info("software already installed", "software", sw.ID, "version", version)
		i.printer.Success(fmt.Sprintf("Already installed: %s %s", sw.DisplayName, version))
		return InstallResult{
			AlreadyInstalled: true,
			SoftwareID:       sw.ID,
			Version:          version,
		}
	}

	log.Info("software not installed, trying install methods", "software", sw.ID, "totalMethods", len(sw.InstallMethods))

	// Try each install method for current platform
	var scriptOutput string
	for idx, method := range sw.InstallMethods {
		if !i.platformSupportsMethod(method) {
			log.Debug("skipping install method (platform mismatch)",
				"method", method.Type,
				"methodIndex", idx+1,
				"totalMethods", len(sw.InstallMethods),
				"currentOS", i.platform.OS(),
				"supportedPlatforms", method.Platforms)
			continue
		}

		log.Info("attempting install method",
			"method", method.Type,
			"methodIndex", idx+1,
			"totalMethods", len(sw.InstallMethods),
			"software", sw.ID)

		var err error
		switch method.Type {
		case "homebrew":
			err = i.installViaHomebrew(ctx, method, sw)
		case "installer_script":
			var stdout, stderr string
			stdout, stderr, err = i.installViaScript(ctx, method, sw)
			if stdout != "" || stderr != "" {
				scriptOutput = stdout
				if stderr != "" {
					if scriptOutput != "" {
						scriptOutput += "\n"
					}
					scriptOutput += stderr
				}
			}
		case "github_release":
			err = i.installViaGitHubRelease(ctx, method, sw)
		default:
			continue
		}

		if err == nil {
			log.Info("install method completed without error", "method", method.Type, "software", sw.ID)
			log.Debug("verifying installation", "software", sw.ID, "binaryName", sw.GetBinaryName(i.platform.OS()))
			if i.isInstalled(sw) {
				version := i.getVersion(sw)
				log.Info("installation verified successfully", "software", sw.ID, "version", version, "method", method.Type)
				i.printer.Success(fmt.Sprintf("Installed %s %s", sw.DisplayName, version))
				return InstallResult{
					ScriptOutput: scriptOutput,
					SoftwareID:   sw.ID,
					Version:      version,
				}
			}
			log.Warn("install method completed but verification failed", "method", method.Type, "software", sw.ID)
		} else {
			log.Warn("install method failed",
				"method", method.Type,
				"software", sw.ID,
				"methodIndex", idx+1,
				"error", err)
		}
	}

	log.Error("all install methods failed", "software", sw.ID, "totalMethodsTried", len(sw.InstallMethods))
	i.printer.FinalError(fmt.Sprintf("Failed to install %s", sw.DisplayName))
	return InstallResult{
		Error:        ErrAllMethodsFailed,
		ScriptOutput: scriptOutput,
		SoftwareID:   sw.ID,
	}
}

// isInstalled checks if software is installed by running its verify command
func (i *Installer) isInstalled(sw *software.Software) bool {
	binaryName := sw.GetBinaryName(i.platform.OS())

	if i.platform.OS() == "windows" {
		return i.platform.CommandExistsInBinDir(binaryName) || i.platform.CommandExists(binaryName)
	}

	return i.platform.CommandExists(binaryName) || i.platform.CommandExistsInBinDir(binaryName)
}

// getVersion extracts version from verify command output
func (i *Installer) getVersion(sw *software.Software) string {
	if sw.VerifyCommand == "" {
		return "unknown"
	}

	var cmd *exec.Cmd
	if i.platform.OS() == "windows" {
		cmd = exec.Command("cmd", "/c", sw.VerifyCommand)
	} else {
		cmd = exec.Command("sh", "-c", sw.VerifyCommand)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return "unknown"
	}

	matches := versionRegex.FindStringSubmatch(string(output))
	if len(matches) > 1 {
		return matches[1]
	}

	return "unknown"
}

// platformSupportsMethod checks if the current OS is supported by the install method
func (i *Installer) platformSupportsMethod(method software.InstallMethod) bool {
	currentOS := i.platform.OS()
	for _, p := range method.Platforms {
		if p == currentOS {
			return true
		}
	}
	return false
}

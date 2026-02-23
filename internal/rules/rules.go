package rules

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/registry"
)

const (
	// WatermarkPrefix is the prefix used to identify marketplace-managed rule files
	WatermarkPrefix = "Marketplace: "
	// Number of lines to check for watermark (at end of file)
	watermarkCheckLines = 5
)

// CopyRulesResult contains the result of copying rules
type CopyRulesResult struct {
	BackedUp     []string
	CopiedFiles  []string
	SkippedFiles []string
}

// Manager handles global rules operations
type Manager struct {
	resolver registry.PathResolver
	rulesDir string
}

// NewManager creates a new rules manager with the given path resolver
func NewManager(resolver registry.PathResolver) *Manager {
	return &Manager{
		resolver: resolver,
		rulesDir: filepath.Join(resolver.ClaudeDir(), "rules"),
	}
}

// GetRulesDir returns the target rules directory path
func (m *Manager) GetRulesDir() string {
	return m.rulesDir
}

// CopyRules copies rule files from all marketplaces to user's rules directory.
// Rule files are prefixed with the marketplace ID to avoid conflicts.
func (m *Manager) CopyRules() (*CopyRulesResult, error) {
	result := &CopyRulesResult{}

	resolver, ok := m.resolver.(*registry.Resolver)
	if !ok {
		return nil, fmt.Errorf("resolver is not a *registry.Resolver")
	}
	rulesPaths, err := resolver.GetAllRulesPaths()
	if err != nil {
		return nil, fmt.Errorf("failed to get marketplace rules paths: %w", err)
	}

	if len(rulesPaths) == 0 {
		result.SkippedFiles = append(result.SkippedFiles, "no marketplaces with rules found")
		return result, nil
	}

	if err := os.MkdirAll(m.rulesDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create rules directory: %w", err)
	}

	for marketplaceID, sourceDir := range rulesPaths {
		mpResult, err := m.copyRulesFromMarketplace(marketplaceID, sourceDir)
		if err != nil {
			return result, fmt.Errorf("failed to copy rules from %s: %w", marketplaceID, err)
		}
		result.CopiedFiles = append(result.CopiedFiles, mpResult.CopiedFiles...)
		result.BackedUp = append(result.BackedUp, mpResult.BackedUp...)
		result.SkippedFiles = append(result.SkippedFiles, mpResult.SkippedFiles...)
	}

	return result, nil
}

// copyRulesFromMarketplace copies rule files from a single marketplace.
// Files are prefixed with the marketplace ID to avoid conflicts.
func (m *Manager) copyRulesFromMarketplace(marketplaceID, sourceDir string) (*CopyRulesResult, error) {
	result := &CopyRulesResult{}

	entries, err := os.ReadDir(sourceDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read source directory: %w", err)
	}

	var mdFiles []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			mdFiles = append(mdFiles, entry.Name())
		}
	}

	if len(mdFiles) == 0 {
		result.SkippedFiles = append(result.SkippedFiles, fmt.Sprintf("no rule files in %s", marketplaceID))
		return result, nil
	}

	for _, filename := range mdFiles {
		sourcePath := filepath.Join(sourceDir, filename)
		prefixedFilename := marketplaceID + "_" + filename
		targetPath := filepath.Join(m.rulesDir, prefixedFilename)

		backupPath, err := m.handleExistingFile(targetPath)
		if err != nil {
			return result, fmt.Errorf("failed to handle existing file %s: %w", prefixedFilename, err)
		}
		if backupPath != "" {
			result.BackedUp = append(result.BackedUp, fmt.Sprintf("%s -> %s", prefixedFilename, filepath.Base(backupPath)))
		}

		if err := m.copyFile(sourcePath, targetPath, marketplaceID); err != nil {
			return result, fmt.Errorf("failed to copy %s: %w", prefixedFilename, err)
		}

		result.CopiedFiles = append(result.CopiedFiles, fmt.Sprintf("%s (from %s)", prefixedFilename, marketplaceID))
	}

	return result, nil
}

// handleExistingFile checks if target file exists and backs it up if it doesn't have watermark.
// Returns the backup path if a backup was created, empty string otherwise.
func (m *Manager) handleExistingFile(targetPath string) (string, error) {
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return "", nil
	}

	hasWatermark, err := m.hasWatermark(targetPath)
	if err != nil {
		return "", fmt.Errorf("failed to check watermark: %w", err)
	}

	if hasWatermark {
		return "", nil
	}

	backupPath := m.generateBackupPath(targetPath)
	if err := os.Rename(targetPath, backupPath); err != nil {
		return "", fmt.Errorf("failed to backup file: %w", err)
	}

	return backupPath, nil
}

// hasWatermark checks if file contains any marketplace watermark in the last few lines
func (m *Manager) hasWatermark(filePath string) (bool, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return false, err
	}

	lines := strings.Split(string(data), "\n")
	totalLines := len(lines)

	checkLines := watermarkCheckLines
	if totalLines < checkLines {
		checkLines = totalLines
	}

	startIdx := totalLines - checkLines
	for i := startIdx; i < totalLines; i++ {
		if strings.Contains(lines[i], WatermarkPrefix) {
			return true, nil
		}
	}

	return false, nil
}

// generateBackupPath creates a timestamped backup filename.
// Preserves .md extension: filename.md -> filename.YYYYMMDD-HHMMSS.backup.md
func (m *Manager) generateBackupPath(filePath string) string {
	dir := filepath.Dir(filePath)
	filename := filepath.Base(filePath)

	nameWithoutExt := strings.TrimSuffix(filename, ".md")
	timestamp := time.Now().Format("20060102-150405")
	backupFilename := fmt.Sprintf("%s.%s.backup.md", nameWithoutExt, timestamp)

	return filepath.Join(dir, backupFilename)
}

// copyFile copies a file from source to destination with 0644 permissions
// and appends a watermark comment to identify the file as marketplace-managed.
func (m *Manager) copyFile(sourcePath, targetPath, marketplaceID string) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open source: %w", err)
	}
	defer sourceFile.Close() //nolint:errcheck // file close

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return fmt.Errorf("failed to create target: %w", err)
	}
	defer targetFile.Close() //nolint:errcheck // file close

	if _, err := io.Copy(targetFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy content: %w", err)
	}

	watermark := fmt.Sprintf("\n\n<!-- %s -->\n", getWatermark(marketplaceID))
	if _, err := targetFile.WriteString(watermark); err != nil {
		return fmt.Errorf("failed to write watermark: %w", err)
	}

	if err := os.Chmod(targetPath, 0644); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	return nil
}

// getWatermark returns the watermark string for a specific marketplace
func getWatermark(marketplaceID string) string {
	return WatermarkPrefix + marketplaceID
}

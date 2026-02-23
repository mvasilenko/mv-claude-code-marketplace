package syncer

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/logger"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/managed"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/merger"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/printer"
	"github.com/mvasilenko/mv-claude-code-marketplace/internal/settings"
)

// SettingsSyncInput contains parameters for settings synchronization
type SettingsSyncInput struct {
	BaseSettings    any
	ClaudeDir       string
	IsUpdate        bool            // true=update (MergeWithRemoval), false=init (Merge)
	PreviousManaged managed.Entries // nil for init, loaded for update
	Printer         *printer.Printer
}

// SettingsSyncResult contains the result of settings synchronization
type SettingsSyncResult struct {
	BackupPath      string
	Created         bool
	MergeStats      MergeStats
	SettingsChanged bool
	SettingsPath    string
	UpdatedManaged  managed.Entries // only populated for update mode
}

// MergeStats tracks merge statistics
type MergeStats struct {
	ArraysMerged  int
	KeysAdded     int
	ObjectsMerged int
}

// SyncSettings performs the complete settings merge workflow:
// 1. Load user settings
// 2. Check if settings file exists (for "created" flag)
// 3. Merge settings (strategy based on IsUpdate)
// 4. Calculate merge stats
// 5. Check if settings changed
// 6. Create backup if needed
// 7. Save merged settings
func SyncSettings(ctx context.Context, input SettingsSyncInput) (*SettingsSyncResult, error) {
	log := logger.FromContext(ctx)

	log.Info("starting settings sync",
		"claudeDir", input.ClaudeDir,
		"isUpdate", input.IsUpdate,
		"hasPreviousManaged", input.PreviousManaged != nil,
	)

	// Step 1: Create settings manager and get path
	settingsMgr := settings.NewManager(input.ClaudeDir)
	settingsPath := settingsMgr.GetSettingsPath()

	log.Debug("settings path resolved", "path", settingsPath)
	input.Printer.Verbose().Info(fmt.Sprintf("Settings path: %s", settingsPath))

	// Step 2: Load existing user settings
	userSettings, err := settingsMgr.Load(ctx)
	if err != nil {
		return nil, err
	}

	// Step 3: Check if settings file exists
	created := false
	if _, err := os.Stat(settingsPath); os.IsNotExist(err) {
		created = true
		input.Printer.Verbose().Info("Settings file does not exist, will create new file")
	}

	// Step 4: Perform merge based on mode
	var mergedSettings any
	var updatedManaged managed.Entries

	if input.IsUpdate {
		baseMap, ok := input.BaseSettings.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("base settings is not a valid object")
		}

		mergedSettingsMap, mgd := merger.MergeWithRemoval(baseMap, userSettings, input.PreviousManaged)
		mergedSettings = mergedSettingsMap
		updatedManaged = mgd
	} else {
		mergedSettings = merger.Merge(input.BaseSettings, userSettings)
	}

	// Step 5: Calculate merge stats
	stats := calculateStats(input.BaseSettings, userSettings, mergedSettings)
	log.Info("merge statistics calculated",
		"keysAdded", stats.KeysAdded,
		"arraysMerged", stats.ArraysMerged,
		"objectsMerged", stats.ObjectsMerged,
	)

	// Step 6: Check if settings actually changed
	settingsChanged := created || !merger.Equal(mergedSettings, userSettings)

	var backupPath string
	if !settingsChanged {
		input.Printer.Info("No changes to settings, skipping backup and write")
	} else {
		// Step 7: Create backup (only if settings exist and will change)
		if !created {
			timestamp := time.Now().Format("20060102-150405")
			backupPath = fmt.Sprintf("%s.%s.backup", settingsPath, timestamp)

			data, err := os.ReadFile(settingsPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read settings for backup: %w", err)
			}

			if err := os.WriteFile(backupPath, data, 0644); err != nil {
				return nil, fmt.Errorf("failed to create backup: %w", err)
			}

			input.Printer.Verbose().Info(fmt.Sprintf("Created backup: %s", filepath.Base(backupPath)))
		}

		// Step 8: Create directory if needed and save
		if err := os.MkdirAll(filepath.Dir(settingsPath), 0755); err != nil {
			return nil, fmt.Errorf("failed to create settings directory: %w", err)
		}

		mergedSettingsMap, ok := mergedSettings.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("merged settings is not a valid object")
		}

		if err := settingsMgr.Save(ctx, mergedSettingsMap); err != nil {
			return nil, err
		}

		input.Printer.Verbose().Info("Settings written successfully")
	}

	input.Printer.Success("Settings merged successfully")

	return &SettingsSyncResult{
		BackupPath:      backupPath,
		Created:         created,
		MergeStats:      stats,
		SettingsChanged: settingsChanged,
		SettingsPath:    settingsPath,
		UpdatedManaged:  updatedManaged,
	}, nil
}

// calculateStats computes merge statistics
func calculateStats(base, user, merged any) MergeStats {
	stats := MergeStats{}

	baseMap, baseIsMap := base.(map[string]any)
	userMap, userIsMap := user.(map[string]any)
	_, mergedIsMap := merged.(map[string]any)

	if !baseIsMap || !mergedIsMap {
		return stats
	}

	if userIsMap {
		for key := range baseMap {
			if _, exists := userMap[key]; !exists {
				stats.KeysAdded++
			}
		}
	} else {
		stats.KeysAdded = len(baseMap)
	}

	countMerges(baseMap, userMap, &stats)

	return stats
}

// countMerges recursively counts merged arrays and objects
func countMerges(base, user map[string]any, stats *MergeStats) {
	for key, baseValue := range base {
		userValue, exists := user[key]
		if !exists {
			continue
		}

		switch baseVal := baseValue.(type) {
		case []any:
			if _, ok := userValue.([]any); ok {
				stats.ArraysMerged++
			}
		case map[string]any:
			if userMap, ok := userValue.(map[string]any); ok {
				stats.ObjectsMerged++
				countMerges(baseVal, userMap, stats)
			}
		}
	}
}

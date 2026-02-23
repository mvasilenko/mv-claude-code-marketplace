package software

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/registry"
)

// embeddedSoftwareConfig contains the fallback software configuration
// This is the canonical version - edit software_config.json in this directory
//
//go:embed software_config.json
var embeddedSoftwareConfig []byte

// Loader loads software configuration
type Loader struct {
	marketplacePath string
}

// NewLoader creates a software config loader with the given path resolver
func NewLoader(resolver registry.PathResolver) *Loader {
	return &Loader{
		marketplacePath: resolver.SoftwareConfigPath(),
	}
}

// Load reads software configuration from marketplace
func (l *Loader) Load() (*Config, error) {
	if _, err := os.Stat(l.marketplacePath); os.IsNotExist(err) {
		return l.getFallbackConfig(), nil
	}

	data, err := os.ReadFile(l.marketplacePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read software config: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse software config: %w", err)
	}

	return &cfg, nil
}

// getFallbackConfig returns embedded software config as fallback
func (l *Loader) getFallbackConfig() *Config {
	var cfg Config
	if err := json.Unmarshal(embeddedSoftwareConfig, &cfg); err != nil {
		// This should never happen since the embedded config is validated at build time
		panic(fmt.Sprintf("failed to parse embedded software config: %v", err))
	}
	return &cfg
}

// LoadEmbedded returns the embedded software config
func (l *Loader) LoadEmbedded() *Config {
	return l.getFallbackConfig()
}

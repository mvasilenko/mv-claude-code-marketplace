package software

import "runtime"

// Config represents the software configuration
type Config struct {
	Software []Software `json:"software"`
	Version  string     `json:"version"`
}

// InstallMethod represents an installation strategy
type InstallMethod struct {
	AssetPattern string   `json:"assetPattern,omitempty"`
	Package      string   `json:"package,omitempty"`
	Platforms    []string `json:"platforms"`
	Repo         string   `json:"repo,omitempty"`
	Type         string   `json:"type"`
	URLUnix      string   `json:"urlUnix,omitempty"`
	URLWindows   string   `json:"urlWindows,omitempty"`
}

// Software represents an installable tool
type Software struct {
	BinaryName        string          `json:"binaryName"`
	BinaryNameWindows string          `json:"binaryNameWindows,omitempty"`
	Category          string          `json:"category"`
	Description       string          `json:"description"`
	DisplayName       string          `json:"displayName"`
	ID                string          `json:"id"`
	InstallMethods    []InstallMethod `json:"installMethods"`
	Priority          int             `json:"priority"`
	VerifyCommand     string          `json:"verifyCommand"`
}

// GetAllIDs returns all software IDs
func (c *Config) GetAllIDs() []string {
	ids := make([]string, len(c.Software))
	for i, sw := range c.Software {
		ids[i] = sw.ID
	}
	return ids
}

// GetSoftware returns software by ID
func (c *Config) GetSoftware(id string) (*Software, error) {
	for i := range c.Software {
		if c.Software[i].ID == id {
			return &c.Software[i], nil
		}
	}
	return nil, ErrSoftwareNotFound
}

// GetBinaryName returns platform-appropriate binary name
func (s *Software) GetBinaryName(goos string) string {
	if goos == "windows" && s.BinaryNameWindows != "" {
		return s.BinaryNameWindows
	}
	return s.BinaryName
}

// GetCurrentPlatformBinaryName returns binary name for current platform
func (s *Software) GetCurrentPlatformBinaryName() string {
	return s.GetBinaryName(runtime.GOOS)
}

// SupportsCurrentPlatform checks if any install method supports current OS
func (s *Software) SupportsCurrentPlatform(goos string) bool {
	for _, method := range s.InstallMethods {
		for _, platform := range method.Platforms {
			if platform == goos {
				return true
			}
		}
	}
	return false
}

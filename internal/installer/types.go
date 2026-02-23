package installer

// InstallResult contains the result of an installation attempt
type InstallResult struct {
	AlreadyInstalled bool
	Error            error
	ScriptOutput     string
	SoftwareID       string
	Version          string
}

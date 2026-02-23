package installer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"

	"github.com/mvasilenko/mv-claude-code-marketplace/internal/logger"
)

// GitHubClient fetches release information from GitHub
type GitHubClient struct {
	owner string
	repo  string
}

// releaseAsset represents a downloadable asset in a release
type releaseAsset struct {
	browserDownloadURL string
	name               string
}

// releaseInfo contains information about a GitHub release
type releaseInfo struct {
	assets     []releaseAsset
	prerelease bool
	tagName    string
	version    Version
}

// githubReleaseResponse matches the GitHub API response
type githubReleaseResponse struct {
	Assets []struct {
		BrowserDownloadURL string `json:"browser_download_url"`
		Name               string `json:"name"`
	} `json:"assets"`
	Prerelease bool   `json:"prerelease"`
	TagName    string `json:"tag_name"`
}

// NewGitHubClient creates a new GitHub client
func NewGitHubClient(owner, repo string) *GitHubClient {
	return &GitHubClient{
		owner: owner,
		repo:  repo,
	}
}

// GetLatestRelease fetches the latest release from GitHub
func (c *GitHubClient) GetLatestRelease(ctx context.Context) (*releaseInfo, error) {
	log := logger.FromContext(ctx)
	log.Debug("fetching releases from GitHub", "owner", c.owner, "repo", c.repo)

	// Try using gh CLI first for authenticated requests (handles private repos)
	releases, err := c.fetchReleasesWithGH()
	if err != nil {
		log.Debug("gh CLI fetch failed, falling back to HTTP", "error", err)
		releases, err = c.fetchReleasesWithHTTP()
		if err != nil {
			log.Debug("HTTP fetch also failed", "error", err)
			return nil, err
		}
		log.Debug("successfully fetched releases via HTTP", "count", len(releases))
	} else {
		log.Debug("successfully fetched releases via gh CLI", "count", len(releases))
	}

	// Find first release with v* tag that's not a prerelease
	for _, release := range releases {
		if !strings.HasPrefix(release.TagName, "v") {
			log.Debug("skipping release without 'v' prefix", "tag", release.TagName)
			continue
		}

		if release.Prerelease {
			log.Debug("skipping prerelease", "tag", release.TagName)
			continue
		}

		versionStr := strings.TrimPrefix(release.TagName, "v")
		version, err := ParseVersion(versionStr)
		if err != nil {
			log.Debug("failed to parse version", "tag", release.TagName, "error", err)
			continue
		}

		var assets []releaseAsset
		for _, asset := range release.Assets {
			assets = append(assets, releaseAsset{
				browserDownloadURL: asset.BrowserDownloadURL,
				name:               asset.Name,
			})
		}

		log.Debug("found valid release", "tag", release.TagName, "version", version.String(), "assetCount", len(assets))
		return &releaseInfo{
			assets:     assets,
			prerelease: release.Prerelease,
			tagName:    release.TagName,
			version:    version,
		}, nil
	}

	log.Debug("no valid releases found")
	return nil, fmt.Errorf("no releases found with tag pattern 'v*.*.*'")
}

// fetchReleasesWithGH fetches releases using gh CLI (authenticated)
func (c *GitHubClient) fetchReleasesWithGH() ([]githubReleaseResponse, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return nil, fmt.Errorf("gh CLI not found")
	}

	endpoint := fmt.Sprintf("repos/%s/%s/releases", c.owner, c.repo)
	cmd := exec.Command("gh", "api", endpoint)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("gh api failed: %s", stderr.String())
	}

	var releases []githubReleaseResponse
	if err := json.Unmarshal(stdout.Bytes(), &releases); err != nil {
		return nil, fmt.Errorf("failed to decode gh api response: %w", err)
	}

	return releases, nil
}

// fetchReleasesWithHTTP fetches releases using unauthenticated HTTP
func (c *GitHubClient) fetchReleasesWithHTTP() ([]githubReleaseResponse, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", c.owner, c.repo)

	resp, err := http.Get(url) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("failed to fetch releases: %w", err)
	}
	defer resp.Body.Close() //nolint:errcheck // HTTP response body

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("GitHub API rate limit exceeded, try again later")
	}

	if resp.StatusCode == 404 {
		return nil, fmt.Errorf("repository not found or no releases available (404)")
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []githubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return nil, fmt.Errorf("failed to decode releases: %w", err)
	}

	return releases, nil
}

// DownloadAsset downloads a release asset using gh CLI (with HTTP fallback)
func (c *GitHubClient) DownloadAsset(ctx context.Context, version, assetName, knownDownloadURL, outputPath string, downloadFileFunc func(string, string) error) error {
	log := logger.FromContext(ctx)
	log.Debug("downloading release asset", "owner", c.owner, "repo", c.repo, "version", version, "asset", assetName)

	// Try gh CLI first (works with private repos)
	if err := c.downloadAssetWithGH(ctx, version, assetName, outputPath); err == nil {
		log.Info("downloaded asset via gh CLI", "asset", assetName)
		return nil
	} else {
		log.Debug("gh CLI download failed, falling back to HTTP", "error", err)
	}

	// Fall back to HTTP download using the URL we already have
	downloadURL := knownDownloadURL
	if downloadURL == "" {
		log.Error("no download URL available for HTTP fallback", "asset", assetName)
		return fmt.Errorf("asset %s: no download URL available", assetName)
	}

	log.Info("downloading asset via HTTP", "url", downloadURL)
	if err := downloadFileFunc(downloadURL, outputPath); err != nil {
		log.Error("HTTP download failed", "error", err, "url", downloadURL)
		return fmt.Errorf("failed to download via HTTP: %w", err)
	}

	log.Debug("downloaded asset successfully via HTTP", "asset", assetName)
	return nil
}

// downloadAssetWithGH downloads an asset using gh CLI
func (c *GitHubClient) downloadAssetWithGH(ctx context.Context, version, assetName, outputPath string) error {
	log := logger.FromContext(ctx)

	if _, err := exec.LookPath("gh"); err != nil {
		log.Debug("gh CLI not found")
		return fmt.Errorf("gh CLI not available")
	}

	cmd := exec.Command("gh", "release", "download", "v"+version,
		"--repo", fmt.Sprintf("%s/%s", c.owner, c.repo),
		"--pattern", assetName,
		"--output", outputPath)

	log.Debug("running gh release download", "version", version, "asset", assetName)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Debug("gh release download failed", "error", err, "output", string(output))
		return fmt.Errorf("gh release download failed: %w (output: %s)", err, string(output))
	}

	log.Debug("gh release download succeeded", "output", string(output))
	return nil
}

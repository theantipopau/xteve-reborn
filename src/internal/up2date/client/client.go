// Package up2date checks GitHub's Releases API for a newer build of this
// fork and, if the admin has opted in, downloads and installs it.
//
// The original xTeve self-updater this package replaces spoke a proprietary
// protocol to a custom update server the original author ran - a server
// this fork has no access to and never will, which is exactly why
// self-update was hardcoded off earlier (see xteve.go). Pointing that same
// protocol at this fork's own GitHub repo wasn't an option since there's no
// server behind it here at all; this package talks to GitHub's public
// Releases API directly instead, using the release-asset naming this
// fork's own release process already produces
// (xteve-reborn_<version>_<os>_<arch>.zip).
package up2date

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

// releaseAsset is one downloadable file attached to a GitHub release.
type releaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// releaseInfo is the subset of GitHub's release API response this package
// actually uses.
type releaseInfo struct {
	TagName string         `json:"tag_name"`
	Draft   bool           `json:"draft"`
	Assets  []releaseAsset `json:"assets"`
}

// Release describes an update found on GitHub: the release tag and the
// download URL for the asset matching the running binary's OS/architecture.
type Release struct {
	Found    bool
	Tag      string
	ZipURL   string
	Filename string
}

var httpClient = &http.Client{Timeout: 10 * time.Second}

// GetLatestRelease queries GitHub's Releases API for the newest release of
// owner/repo and looks for the asset matching the running binary's
// OS/architecture. Deliberately uses the /releases list endpoint rather
// than /releases/latest: the latter explicitly excludes prereleases, and
// this fork is still shipping those. The list is returned newest-first, so
// the first non-draft entry is the one being checked.
func GetLatestRelease(owner, repo, binaryName string) (release Release, err error) {

	var url = fmt.Sprintf("https://api.github.com/repos/%s/%s/releases", owner, repo)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err = fmt.Errorf("GitHub API: %d %s", resp.StatusCode, resp.Status)
		return
	}

	var releases []releaseInfo
	err = json.NewDecoder(resp.Body).Decode(&releases)
	if err != nil {
		return
	}

	for _, r := range releases {

		if r.Draft {
			continue
		}

		var assetName = fmt.Sprintf("%s_%s_%s_%s.zip", binaryName, strings.TrimPrefix(r.TagName, "v"), runtime.GOOS, runtime.GOARCH)

		for _, asset := range r.Assets {

			if asset.Name == assetName {
				release.Found = true
				release.Tag = r.TagName
				release.ZipURL = asset.BrowserDownloadURL
				release.Filename = binaryName
				return
			}

		}

	}

	return
}

package env

import (
	"encoding/json"
	"errors"
	"fmt"
	"iter"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
)

const (
	downloadListURL      = "https://golang.org/dl/?mode=json&include=all"
	downloadListFileName = "downloads.json"
)

type remoteFile struct {
	Filename       string `json:"filename"`
	OS             string `json:"os"`
	Arch           string `json:"arch"`
	Version        string `json:"version"`
	ChecksumSHA256 string `json:"sha256"`
	Size           int64  `json:"size"`
	Kind           string `json:"kind"` // "archive", "installer", "source"
}

type remoteRelease struct {
	Version string       `json:"version"`
	Stable  bool         `json:"stable"`
	Files   []remoteFile `json:"files"`
}

type Release struct {
	Version        *Version
	Stable         bool
	Filename       string
	ChecksumSHA256 string
	Size           int64
}

func (env *Env) UpdateDownloadList() error {
	req, err := http.NewRequest(http.MethodGet, downloadListURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create http request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to get download list: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to get download list: %s", res.Status)
	}

	mimeType, _, err := mime.ParseMediaType(res.Header.Get("Content-Type"))
	if err != nil {
		return fmt.Errorf("failed to parse Content-Type: %w", err)
	}

	if mimeType != "application/json" {
		return fmt.Errorf("the server responds unexpected Content-Type: %s", mimeType)
	}

	var releases []remoteRelease
	if err := json.NewDecoder(res.Body).Decode(&releases); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	rls, err := convertReleases(releases)
	if err != nil {
		return err
	}
	env.releases = rls

	if err := os.MkdirAll(env.confDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	file, err := os.Create(filepath.Join(env.confDir, downloadListFileName))
	if err != nil {
		return fmt.Errorf("failed to create download list file: %w", err)
	}
	defer file.Close()

	e := json.NewEncoder(file)
	e.SetIndent("", "  ")
	if err := e.Encode(releases); err != nil {
		return fmt.Errorf("failed to encode JSON: %w", err)
	}

	return nil
}

func convertReleases(releases []remoteRelease) ([]*Release, error) {
	var rls []*Release

	for _, r := range releases {
		for _, f := range r.Files {
			if !targetRelease(f) {
				continue
			}

			version, err := ParseVersion(strings.TrimPrefix(r.Version, "go"))
			if err != nil {
				return nil, err
			}

			rls = append(rls, &Release{
				Version:        version,
				Stable:         r.Stable,
				Filename:       f.Filename,
				ChecksumSHA256: f.ChecksumSHA256,
				Size:           f.Size,
			})
		}
	}

	sort.Slice(rls, func(i, j int) bool {
		return CompareVersion(rls[i].Version, rls[j].Version) < 0
	})

	return rls, nil
}

func targetRelease(f remoteFile) bool {
	if f.OS != runtime.GOOS || f.Kind != "archive" {
		return false
	}

	arch := runtime.GOARCH
	if arch == "arm" {
		arch = "armv6l"
	}

	if f.Arch != arch {
		return false
	}

	return true
}

var ErrReleasesFileNotDownloaded = errors.New("releases file is not found")

func (env *Env) loadReleases() error {
	name := filepath.Join(env.confDir, downloadListFileName)
	if _, err := os.Stat(name); err != nil {
		return ErrReleasesFileNotDownloaded
	}

	file, err := os.Open(name)
	if err != nil {
		return fmt.Errorf("failed to open releases file: %w", err)
	}
	defer file.Close()

	var releases []remoteRelease
	if err := json.NewDecoder(file).Decode(&releases); err != nil {
		return fmt.Errorf("failed to decode JSON: %w", err)
	}

	rls, err := convertReleases(releases)
	if err != nil {
		return err
	}
	env.releases = rls

	return nil
}

func (env *Env) Releases() ([]*Release, error) {
	if env.releases == nil {
		if err := env.loadReleases(); err != nil {
			return nil, err
		}
	}

	if len(env.releases) == 0 {
		return nil, nil
	}

	return slices.Clone(env.releases), nil
}

func (env *Env) RecentReleases() ([]*Release, error) {
	if env.releases == nil {
		if err := env.loadReleases(); err != nil {
			return nil, err
		}
	}

	if len(env.releases) == 0 {
		return nil, nil
	}

	releases := slices.Collect(env.selectRecentReleases(2))
	slices.Reverse(releases)

	return releases, nil
}

func (env *Env) selectRecentReleases(n int) iter.Seq[*Release] {
	return func(yield func(*Release) bool) {
		var latest *Version
		// add the latest unstable release first
		for _, r := range slices.Backward(env.releases) {
			if r.Stable {
				break
			}

			if latest == nil {
				latest = &Version{
					Major: r.Version.Major,
					Minor: r.Version.Minor,
				}
			}

			if r.Version.Major != latest.Major || r.Version.Minor != latest.Minor {
				break
			}

			if !yield(r) {
				return
			}
		}

		latest = nil
		count := 0
		// add the latest stable releases second
		for _, r := range slices.Backward(env.releases) {
			if !r.Stable {
				continue
			}

			if latest == nil || r.Version.Major != latest.Major || r.Version.Minor != latest.Minor {
				if latest != nil {
					n--
				}
				if n == 0 {
					break
				}
				count = 0

				latest = &Version{
					Major: r.Version.Major,
					Minor: r.Version.Minor,
				}
			}

			if count < 2 {
				if !yield(r) {
					return
				}
			}
			count++
		}
	}

}

func (env *Env) FindRelease(v *Version) (*Release, error) {
	if env.releases == nil {
		if err := env.loadReleases(); err != nil {
			return nil, err
		}
	}

	for _, r := range env.releases {
		if EqualVersion(r.Version, v) {
			return r, nil
		}
	}

	return nil, errors.New("specified version is not found")
}

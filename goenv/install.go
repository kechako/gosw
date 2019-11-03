package goenv

import (
	"archive/tar"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	downloadBaseURL = "https://dl.google.com/go/"
)

func (env *Env) Install(v *Version) error {
	if env.releases == nil {
		if err := env.loadReleases(); err != nil {
			return err
		}
	}

	if env.HasVersion(v) {
		return errors.New("specified version is already installed")
	}

	goRoot := env.versionGoRoot(v)
	dlURL, err := env.downloadURL(v)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodGet, dlURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create download request: %w", err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to download archive: %w", err)
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		// ok
	case http.StatusNotFound:
		return errors.New("specified version is not found")
	default:
		return fmt.Errorf("failed to download archive: %s", res.Status)
	}

	if _, err := os.Stat(goRoot); err == nil {
		return errors.New("install target directory already exists")
	}

	if err := os.Mkdir(goRoot, 0755); err != nil {
		return fmt.Errorf("failed to create install target directory: %w", err)
	}

	if err := extractTar(res.Body, goRoot, true, 1); err != nil {
		return err
	}

	if err := env.fixBrokenLink(); err != nil {
		return err
	}

	return nil
}

func (env *Env) downloadURL(v *Version) (string, error) {
	r, err := env.FindRelease(v)
	if err != nil {
		return "", err
	}

	return downloadBaseURL + r.Filename, nil
}

func extractTar(r io.Reader, dest string, isGzip bool, strip int) error {
	if isGzip {
		gzr, err := gzip.NewReader(r)
		if err != nil {
			return fmt.Errorf("failed to decompress gzip: %w", err)
		}
		defer gzr.Close()

		r = gzr
	}

	tr := tar.NewReader(r)

	for {
		h, err := tr.Next()
		if err != nil {
			if err == io.EOF {
				break
			}

			return fmt.Errorf("failed to read tar: %w", err)
		}
		info := h.FileInfo()

		rpath := stripPath(h.Name, strip)
		if rpath == "" {
			continue
		}

		path := filepath.Join(dest, rpath)
		perm := os.FileMode(h.Mode)

		if info.IsDir() {
			os.Mkdir(path, perm)
		} else {
			if err := writeFile(path, perm, tr); err != nil {
				return err
			}
		}
		if !info.ModTime().IsZero() {
			if err := os.Chtimes(path, time.Now(), info.ModTime()); err != nil {
				fmt.Fprintf(os.Stderr, "%s: failed to change the access and modification times: %v\n", path, err)
			}
		}
	}

	return nil
}

func stripPath(path string, strip int) string {
	if path[0] == filepath.Separator {
		path = path[1:]
	}

	for i := 0; i < strip; i++ {
		i := strings.Index(path, string(filepath.Separator))
		if i < 0 {
			return ""
		}

		path = path[i+1:]
	}

	return path
}

func writeFile(path string, perm os.FileMode, r io.Reader) error {
	file, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE|os.O_TRUNC, perm)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, r); err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}

	return nil
}

func (env *Env) Uninstall(v *Version) error {
	if env.HasVersion(v) {
		return errors.New("specified version is not installed")
	}
	goRoot := env.versionGoRoot(v)

	if err := os.RemoveAll(goRoot); err != nil {
		return fmt.Errorf("failed to remove %s: %w", goRoot, err)
	}

	delete(env.installedVersions, v.String())

	if err := env.fixBrokenLink(); err != nil {
		return err
	}

	return nil
}

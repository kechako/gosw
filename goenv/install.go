package goenv

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/cheggaaa/pb/v3"
)

const (
	downloadBaseURL = "https://dl.google.com/go/"
)

func (env *Env) Install(v *Version) error {
	if env.HasVersion(v) {
		return errors.New("specified version is already installed")
	}

	dlURL, dlName, err := env.downloadURL(v)
	if err != nil {
		return err
	}

	cachePath := filepath.Join(env.cacheDir, dlName)
	e, err := getExtractor(cachePath)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(env.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	goRoot := env.versionGoRoot(v)

	if _, err := os.Stat(cachePath); err != nil {
		if err := download(dlURL, cachePath); err != nil {
			return err
		}
	}

	if _, err := os.Stat(goRoot); err == nil {
		return errors.New("install target directory already exists")
	}

	if err := os.Mkdir(goRoot, 0755); err != nil {
		return fmt.Errorf("failed to create install target directory: %w", err)
	}

	fmt.Println("Extract...")
	e.extract(goRoot)

	if err := env.fixBrokenLink(); err != nil {
		return err
	}

	return nil
}

func download(url, path string) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
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

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create cache file: %w", err)
	}
	defer file.Close()

	var r io.Reader = res.Body
	if res.ContentLength > 0 {
		bar := pb.New64(res.ContentLength).SetTemplate(pb.Full)

		r = bar.NewProxyReader(res.Body)
		bar.Set(pb.Bytes, true)
		bar.Set("prefix", "Download... ")

		bar.Start()
		defer bar.Finish()
	} else {
		fmt.Println("Download...")
	}

	if _, err := io.Copy(file, r); err != nil {
		return fmt.Errorf("failed to download archive: %w", err)
	}

	return nil
}

func (env *Env) downloadURL(v *Version) (string, string, error) {
	r, err := env.FindRelease(v)
	if err != nil {
		return "", "", err
	}

	return downloadBaseURL + r.Filename, r.Filename, nil
}

func (env *Env) Uninstall(v *Version) error {
	if !env.HasVersion(v) {
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

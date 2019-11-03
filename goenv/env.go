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
	"runtime"
	"sort"
	"strings"
	"time"
)

var (
	DefaultEnvRoot         = "/usr/local/go"
	DefaultVersionLinkName = "current"
)

const (
	downloadURLFormat = "https://dl.google.com/go/go%s.%s-%s.tar.gz"
)

type Env struct {
	envRoot     string
	verLinkName string

	installedVersions map[string]*Version
}

func New(opts ...Option) (*Env, error) {
	env := &Env{
		envRoot:           DefaultEnvRoot,
		verLinkName:       DefaultVersionLinkName,
		installedVersions: make(map[string]*Version),
	}
	for _, opt := range opts {
		opt.apply(env)
	}

	if err := env.init(); err != nil {
		return nil, err
	}

	return env, nil
}

func (env *Env) init() error {
	versions, err := installedVersions(env.envRoot)
	if err != nil {
		return err
	}

	for _, version := range versions {
		env.installedVersions[version.String()] = version
	}

	return nil
}

func installedVersions(root string) ([]*Version, error) {
	dirs, err := filepath.Glob(filepath.Join(root, "/go*"))
	if err != nil {
		return nil, err
	}

	var versions []*Version
	for _, dir := range dirs {
		if info, err := os.Stat(dir); err == nil {
			if !info.IsDir() {
				continue
			}
			str := info.Name()[2:]
			version, err := ParseVersion(str)
			if err != nil {
				fmt.Println(err, str)
				continue
			}
			versions = append(versions, version)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return CompareVersion(versions[i], versions[j]) < 0
	})

	return versions, nil
}

func (env *Env) InstalledVersions() []*Version {
	versions := make([]*Version, 0, len(env.installedVersions))
	for _, v := range env.installedVersions {
		versions = append(versions, &(*v))
	}

	sort.Slice(versions, func(i, j int) bool {
		return CompareVersion(versions[i], versions[j]) < 0
	})

	return versions
}

func (env *Env) HasVersion(v *Version) bool {
	_, ok := env.installedVersions[v.String()]
	if !ok {
		return false
	}

	return true
}

func (env *Env) Install(v *Version) error {
	if env.HasVersion(v) {
		return errors.New("specified version is already installed")
	}

	goRoot := env.versionGoRoot(v)
	dlURL := downloadURL(env.envRoot, v)

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

func downloadURL(root string, v *Version) string {
	return fmt.Sprintf(downloadURLFormat, v, runtime.GOOS, runtime.GOARCH)
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

func (env *Env) Switch(v *Version) error {
	if env.HasVersion(v) {
		return errors.New("specified version is not installed")
	}

	return env.makeLink(v)
}

func (env *Env) linkPath() string {
	return filepath.Join(env.envRoot, env.verLinkName)
}

func (env *Env) versionGoRoot(v *Version) string {
	return filepath.Join(env.envRoot, "go"+v.String())
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

func (env *Env) fixBrokenLink() error {
	path := env.linkPath()

	if _, err := os.Stat(path); err == nil {
		// link target is not broken
		return nil
	}

	versions := env.InstalledVersions()
	if len(versions) == 0 {
		return nil
	}

	if err := env.makeLink(versions[len(versions)-1]); err != nil {
		return err
	}

	return nil
}

func (env *Env) makeLink(v *Version) error {
	goRoot := env.versionGoRoot(v)

	path := env.linkPath()
	if _, err := os.Lstat(path); err == nil {
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("failed to remove current symbolic link: %w", err)
		}
	}

	if err := os.Symlink(goRoot, path); err != nil {
		return fmt.Errorf("failed to create new symbolic link: %w", err)
	}

	return nil
}

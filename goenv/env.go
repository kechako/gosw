package goenv

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

var (
	DefaultEnvRoot         = "/usr/local/go"
	DefaultVersionLinkName = "current"
)

type Env struct {
	envRoot     string
	verLinkName string
	confDir     string
	cacheDir    string

	installedVersions map[string]*Version
	releases          []*Release
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

	if env.confDir == "" {
		confDir, err := getConfPath()
		if err != nil {
			return nil, err
		}
		env.confDir = confDir
	}

	if env.cacheDir == "" {
		env.cacheDir = getCachePath()
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

func getConfPath() (string, error) {
	confDir, err := os.UserConfigDir()
	if err != nil {
		userDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user config directory: %w", err)
		}

		return userDir + "/.gosw", nil
	}

	return confDir + "/gosw", nil
}

func getCachePath() string {
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		tempDir := os.TempDir()

		return tempDir + "/.gosw"
	}

	return cacheDir + "/gosw"
}

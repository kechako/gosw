package env

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type extractor interface {
	extract(dest string) error
}

type tarArchive struct {
	path   string
	isGzip bool
}

func getExtractor(path string) (extractor, error) {
	switch {
	case strings.HasSuffix(path, ".tar"):
		return &tarArchive{path: path, isGzip: false}, nil
	case strings.HasSuffix(path, ".tar.gz"):
		return &tarArchive{path: path, isGzip: true}, nil
	case strings.HasSuffix(path, ".zip"):
		return &zipArchive{path: path}, nil
	default:
		return nil, errors.New("unsupported archive")
	}
}

func (a *tarArchive) extract(dest string) error {
	var r io.Reader

	file, err := os.Open(a.path)
	if err != nil {
		return fmt.Errorf("failed to open cached archive: %w", err)
	}
	defer file.Close()

	r = file
	if a.isGzip {
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

		rpath := stripPath(h.Name, 1)
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

type zipArchive struct {
	path string
}

func (a *zipArchive) extract(dest string) error {
	zr, err := zip.OpenReader(a.path)
	if err != nil {
		return fmt.Errorf("failed to open cached archive as ZIP: %w", err)
	}
	defer zr.Close()

	for _, file := range zr.File {
		r, err := file.Open()
		if err != nil {
			return fmt.Errorf("failed to open file in ZIP: %w", err)
		}

		rpath := stripPath(file.Name, 1)
		if rpath == "" {
			continue
		}

		path := filepath.Join(dest, rpath)
		perm := file.Mode()

		info := file.FileInfo()
		if info.IsDir() {
			os.Mkdir(path, perm)
		} else {
			if err := writeFile(path, perm, r); err != nil {
				r.Close()
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

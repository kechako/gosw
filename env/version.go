package env

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type VersionType int

const (
	Stable VersionType = iota
	Beta
	RC
	Head
)

type Version struct {
	Type    VersionType
	Major   int
	Minor   int
	Patch   int
	Release int // Release number of beta or rc.
}

var ErrVersionSyntax = errors.New("invalid version syntax")

var versionRegexp = regexp.MustCompile(`^(1)\.([0-9]+)((\.([0-9]+))|((beta|rc)([0-9]+)))?$`)

const headVersion = "go-head"

func ParseVersion(s string) (*Version, error) {
	if s == headVersion {
		return &Version{Type: Head}, nil
	}

	s = strings.TrimPrefix(s, "go")

	matches := versionRegexp.FindStringSubmatch(s)
	if matches == nil {
		return nil, ErrVersionSyntax
	}

	// major
	major, err := strconv.Atoi(matches[1])
	if err != nil {
		return nil, ErrVersionSyntax
	}
	minor, err := strconv.Atoi(matches[2])
	if err != nil {
		return nil, ErrVersionSyntax
	}

	var patch, rn int
	var tp VersionType
	if matches[3] == "" {
		// major and minor only
		tp = Stable
	} else if matches[5] != "" {
		var err error
		patch, err = strconv.Atoi(matches[5])
		if err != nil {
			return nil, ErrVersionSyntax
		}
		tp = Stable
	} else {
		if matches[7] == "beta" {
			tp = Beta
		} else {
			tp = RC
		}

		var err error
		rn, err = strconv.Atoi(matches[8])
		if err != nil {
			return nil, ErrVersionSyntax
		}
	}

	return &Version{
		Major:   major,
		Minor:   minor,
		Patch:   patch,
		Type:    tp,
		Release: rn,
	}, nil
}

func (v *Version) String() string {
	switch v.Type {
	case Stable:
		if v.Patch > 0 {
			return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
		}

		return fmt.Sprintf("%d.%d", v.Major, v.Minor)
	case Beta:
		return fmt.Sprintf("%d.%dbeta%d", v.Major, v.Minor, v.Release)
	case RC:
		return fmt.Sprintf("%d.%drc%d", v.Major, v.Minor, v.Release)
	case Head:
		return headVersion
	}

	return ""
}

func CompareVersion(x, y *Version) int {
	if x.Type == Head && y.Type == Head {
		return 0
	} else if x.Type == Head {
		return 1
	} else if y.Type == Head {
		return -1
	}

	if x.Major != y.Major {
		return compareInt(x.Major, y.Major)
	}

	if x.Minor != y.Minor {
		return compareInt(x.Minor, y.Minor)
	}

	if x.Type != y.Type {
		if x.Type == Stable {
			return 1
		}

		if x.Type == Beta {
			return -1
		}

		if y.Type == Stable {
			return -1
		}

		if y.Type == Beta {
			return 1
		}
	}

	if x.Type == Stable {
		return compareInt(x.Patch, y.Patch)
	} else {
		return compareInt(x.Release, y.Release)
	}
}

func EqualVersion(x, y *Version) bool {
	return CompareVersion(x, y) == 0
}

func compareInt(x, y int) int {
	if x > y {
		return 1
	} else if x < y {
		return -1
	}

	return 0
}

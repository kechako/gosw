package goenv

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
)

type VersionType int

const (
	StableVersion VersionType = iota
	BetaVersion
	RCVersion
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

func ParseVersion(s string) (*Version, error) {
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
		tp = StableVersion
	} else if matches[5] != "" {
		var err error
		patch, err = strconv.Atoi(matches[5])
		if err != nil {
			return nil, ErrVersionSyntax
		}
		tp = StableVersion
	} else {
		if matches[7] == "beta" {
			tp = BetaVersion
		} else {
			tp = RCVersion
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
	case StableVersion:
		if v.Patch > 0 {
			return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
		}

		return fmt.Sprintf("%d.%d", v.Major, v.Minor)
	case BetaVersion:
		return fmt.Sprintf("%d.%dbeta%d", v.Major, v.Minor, v.Release)
	case RCVersion:
		return fmt.Sprintf("%d.%drc%d", v.Major, v.Minor, v.Release)
	}

	return ""
}

func CompareVersion(x, y *Version) int {
	if x.Major != y.Major {
		return compareInt(x.Major, y.Major)
	}

	if x.Minor != y.Minor {
		return compareInt(x.Minor, y.Minor)
	}

	if x.Type != y.Type {
		if x.Type == StableVersion {
			return 1
		}

		if x.Type == BetaVersion {
			return -1
		}

		if y.Type == StableVersion {
			return -1
		}

		if y.Type == BetaVersion {
			return 1
		}
	}

	if x.Type == StableVersion {
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

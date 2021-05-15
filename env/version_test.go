package env

import (
	"reflect"
	"testing"
)

var versionTests = map[string]struct {
	s   string
	v   *Version
	err error
}{
	"go-head": {
		s: "go-head",
		v: &Version{
			Type:    Head,
			Major:   0,
			Minor:   0,
			Patch:   0,
			Release: 0,
		},
		err: nil,
	},
	"go1.16": {
		s: "go1.16",
		v: &Version{
			Type:    Stable,
			Major:   1,
			Minor:   16,
			Patch:   0,
			Release: 0,
		},
		err: nil,
	},
	"go1.15.10": {
		s: "go1.15.10",
		v: &Version{
			Type:    Stable,
			Major:   1,
			Minor:   15,
			Patch:   10,
			Release: 0,
		},
		err: nil,
	},
	"go1.16beta3": {
		s: "go1.16beta3",
		v: &Version{
			Type:    Beta,
			Major:   1,
			Minor:   16,
			Patch:   0,
			Release: 3,
		},
		err: nil,
	},
	"go1.15rc10": {
		s: "go1.15rc10",
		v: &Version{
			Type:    RC,
			Major:   1,
			Minor:   15,
			Patch:   0,
			Release: 10,
		},
		err: nil,
	},
	"go1.14.1beta4": {
		s: "go1.14.1beta4",
		v: &Version{
			Type:    Beta,
			Major:   1,
			Minor:   14,
			Patch:   1,
			Release: 4,
		},
		err: nil,
	},
	"go1.13.14rc15": {
		s: "go1.13.14rc15",
		v: &Version{
			Type:    RC,
			Major:   1,
			Minor:   13,
			Patch:   14,
			Release: 15,
		},
		err: nil,
	},
	// errors
	"go--head": {
		s:   "go--head",
		v:   nil,
		err: ErrVersionSyntax,
	},
	"go1.16.": {
		s:   "go1.16.",
		v:   nil,
		err: ErrVersionSyntax,
	},
	"go1.15.10.": {
		s:   "go1.15.10.",
		v:   nil,
		err: ErrVersionSyntax,
	},
	"go1.16beta": {
		s:   "go1.16beta",
		v:   nil,
		err: ErrVersionSyntax,
	},
	"go1.15xxx10": {
		s:   "go1.15xxx10",
		v:   nil,
		err: ErrVersionSyntax,
	},
	"go1.15..10": {
		s:   "go1.15..10",
		v:   nil,
		err: ErrVersionSyntax,
	},
	"go1.2.3.4": {
		s:   "go1.2.3.4",
		v:   nil,
		err: ErrVersionSyntax,
	},
	"go10": {
		s:   "go10",
		v:   nil,
		err: ErrVersionSyntax,
	},
}

func Test_ParseVersion(t *testing.T) {
	for name, tt := range versionTests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			v, err := ParseVersion(tt.s)
			if err != tt.err {
				t.Error(err)
			}
			if !reflect.DeepEqual(v, tt.v) {
				t.Errorf("ParseVersion(%v): got %v, want %v", tt.s, v, tt.v)
			}
		})
	}
}

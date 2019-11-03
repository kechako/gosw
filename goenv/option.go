package goenv

import "path/filepath"

type Option interface {
	apply(env *Env)
}

type optionFunc func(env *Env)

func (f optionFunc) apply(env *Env) {
	f(env)
}

func WithEnvRoot(root string) Option {
	return optionFunc(func(env *Env) {
		env.envRoot = filepath.Clean(root)
	})
}

func WithVersionLinkName(name string) Option {
	return optionFunc(func(env *Env) {
		env.verLinkName = name
	})
}

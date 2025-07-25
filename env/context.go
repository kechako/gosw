package env

import "context"

type ctxKey struct{}

func NewContext(parent context.Context, env *Env) context.Context {
	return context.WithValue(parent, ctxKey{}, env)
}

func FromContext(ctx context.Context) *Env {
	env, _ := ctx.Value(ctxKey{}).(*Env)
	return env
}

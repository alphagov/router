package integration

import (
	"strings"
)

type envMap map[string]string

func newEnvMap(env []string) (em envMap) {
	em = make(map[string]string, len(env))
	for _, item := range env {
		parts := strings.SplitN(item, "=", 2)
		em[parts[0]] = parts[1]
	}
	return
}

func (em envMap) ToEnv() (env []string) {
	env = make([]string, 0, len(em))
	for k, v := range em {
		env = append(env, k+"="+v)
	}
	return
}

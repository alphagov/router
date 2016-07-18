package tablecloth

import (
	"strings"
)

type envMap map[string]string

func newEnvMap(env []string) envMap {
	em := make(map[string]string, len(env))
	for _, item := range env {
		parts := strings.SplitN(item, "=", 2)
		em[parts[0]] = parts[1]
	}
	return em
}

func (em envMap) ToEnv() []string {
	env := make([]string, 0, len(em))
	for k, v := range em {
		env = append(env, k+"="+v)
	}
	return env
}

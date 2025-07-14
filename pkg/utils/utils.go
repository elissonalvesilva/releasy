package utils

import (
	"encoding/json"
	"github.com/elissonalvesilva/releasy/internal/core/domain"
	"strconv"
	"strings"
)

func ParseEnvString(envs string) []string {
	var out []string
	_ = json.Unmarshal([]byte(envs), &out)
	return out
}

func ExtractPort(envs []string) int {
	for _, env := range envs {
		if strings.HasPrefix(env, "APP_PORT=") {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				if portVal, err := strconv.Atoi(parts[1]); err == nil {
					return portVal
				}
			}
		}
	}
	return domain.DefaultServicePort
}

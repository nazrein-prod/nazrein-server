package utils

import (
	"encoding/base64"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/gorilla/securecookie"
)

// DecodeKey reads a base64-encoded key from the given environment variable name.
// If the variable is empty:
//   - in prod: returns an error
//   - in test/dev: generates a random ephemeral key with a warning log
//
// If present, it verifies that the decoded length matches want bytes.
func DecodeKey(log *slog.Logger, name string, want int, production bool) ([]byte, error) {
	raw := os.Getenv(name)
	if raw == "" {
		if production {
			return nil, fmt.Errorf("%s is required in production", name)
		}

		log.Warn("Session key not set, using ephemeral key — sessions will not survive a restart", "key", name)

		return securecookie.GenerateRandomKey(want), nil
	}

	key, decErr := base64.StdEncoding.DecodeString(strings.TrimSpace(raw))
	if decErr != nil {
		return nil, fmt.Errorf("%s is not valid base64: %w", name, decErr)
	}

	if len(key) != want {
		return nil, fmt.Errorf("%s must decode to %d bytes, got %d", name, want, len(key))
	}

	return key, nil
}

func SessionKeys(log *slog.Logger, authEnv, encEnv string, production bool) (authKey, encKey []byte, err error) {
	authKey, err = DecodeKey(log, authEnv, 64, production)
	if err != nil {
		return nil, nil, err
	}

	encKey, err = DecodeKey(log, encEnv, 32, production)
	if err != nil {
		return nil, nil, err
	}

	return authKey, encKey, nil
}

package utils

import (
	"encoding/base64"
	"io"
	"log/slog"
	"os"
	"testing"
)

func TestDecodeKey(t *testing.T) {
	discardLogger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("production error when env missing", func(t *testing.T) {
		const envKey = "TEST_MISSING_KEY_PROD"
		_ = os.Unsetenv(envKey)

		_, err := DecodeKey(discardLogger, envKey, 32, true)
		if err == nil {
			t.Fatal("expected error in production when key is missing, got nil")
		}
	})

	t.Run("non-production generates ephemeral key", func(t *testing.T) {
		const envKey = "TEST_MISSING_KEY_DEV"
		_ = os.Unsetenv(envKey)

		key, err := DecodeKey(discardLogger, envKey, 32, false)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(key) != 32 {
			t.Fatalf("expected 32 bytes, got %d", len(key))
		}
	})

	t.Run("invalid base64 returns error", func(t *testing.T) {
		const envKey = "TEST_INVALID_B64"
		t.Setenv(envKey, "not-valid-base64!!")

		_, err := DecodeKey(discardLogger, envKey, 32, false)
		if err == nil {
			t.Fatal("expected base64 error, got nil")
		}
	})

	t.Run("length mismatch returns error", func(t *testing.T) {
		const envKey = "TEST_WRONG_LEN"
		shortData := base64.StdEncoding.EncodeToString([]byte("too-short"))
		t.Setenv(envKey, shortData)

		_, err := DecodeKey(discardLogger, envKey, 32, false)
		if err == nil {
			t.Fatal("expected length mismatch error, got nil")
		}
	})

	t.Run("valid base64 decodes successfully", func(t *testing.T) {
		const envKey = "TEST_VALID_KEY"
		expected := make([]byte, 32)
		for i := range expected {
			expected[i] = byte(i)
		}
		t.Setenv(envKey, base64.StdEncoding.EncodeToString(expected))

		key, err := DecodeKey(discardLogger, envKey, 32, true)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if string(key) != string(expected) {
			t.Fatalf("decoded key did not match expected bytes")
		}
	})
}

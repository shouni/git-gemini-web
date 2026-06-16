package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidateEssentialConfig(t *testing.T) {
	base := &Config{
		ServiceURL:         "https://example.com",
		GoogleClientID:     "client-id",
		GoogleClientSecret: "client-secret",
		SessionSecret:      "session-secret",
		SessionEncryptKey:  "1234567890123456",
		AllowedEmails:      []string{"user@example.com"},
	}

	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name:   "valid config",
			mutate: func(c *Config) {},
		},
		{
			name:    "insecure service url",
			mutate:  func(c *Config) { c.ServiceURL = "http://example.com" },
			wantErr: "HTTPS",
		},
		{
			name:    "missing oauth setting",
			mutate:  func(c *Config) { c.GoogleClientID = "" },
			wantErr: "Google OAuth",
		},
		{
			name: "missing allow list",
			mutate: func(c *Config) {
				c.AllowedEmails = nil
				c.AllowedDomains = nil
			},
			wantErr: "認可リスト",
		},
		{
			name:    "invalid encrypt key length",
			mutate:  func(c *Config) { c.SessionEncryptKey = "short" },
			wantErr: "長さが不正",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := *base
			tt.mutate(&cfg)

			err := cfg.ValidateEssentialConfig()
			if tt.wantErr == "" && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tt.wantErr != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErr)
				}
				if !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("error %q does not contain %q", err.Error(), tt.wantErr)
				}
			}
		})
	}
}

func TestStorageURI(t *testing.T) {
	cfg := &Config{GCSBucket: "review-bucket"}
	now := time.Date(2026, 4, 15, 12, 34, 56, 0, time.UTC)

	got := cfg.StorageURI("git@github.com:org/repo.git", "feature/new-ui", now)

	if !strings.HasPrefix(got, "gs://review-bucket/reviews/") {
		t.Fatalf("unexpected prefix: %s", got)
	}
	if !strings.Contains(got, "20260415_123456_feature-new-ui.html") {
		t.Fatalf("branch or timestamp format mismatch: %s", got)
	}
}

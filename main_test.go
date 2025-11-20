package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealthzHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()

	healthzHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	body := w.Body.String()
	if body != "OK" {
		t.Errorf("expected body 'OK', got '%s'", body)
	}
}

func TestGetInfoHandler(t *testing.T) {
	// Set some test values
	version = "v1.0.0"
	gitCommit = "abc123"
	gitBranch = "main"
	buildTime = "2025-01-01T00:00:00Z"
	gitRepository = "https://github.com/choffmann/chat-room"

	req := httptest.NewRequest("GET", "/info", nil)
	w := httptest.NewRecorder()

	getInfoHandler(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var info Info
	if err := json.NewDecoder(w.Body).Decode(&info); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if info.Version != version {
		t.Errorf("expected version %s, got %s", version, info.Version)
	}

	if info.GitCommit != gitCommit {
		t.Errorf("expected gitCommit %s, got %s", gitCommit, info.GitCommit)
	}

	if info.GitBranch != gitBranch {
		t.Errorf("expected gitBranch %s, got %s", gitBranch, info.GitBranch)
	}

	// Check that BuildTime is set
	if info.BuildTime.IsZero() {
		t.Error("expected buildTime to be set")
	}

	if info.GitRepository != gitRepository {
		t.Errorf("expected gitRepository %s, got %s", gitRepository, info.GitRepository)
	}
}

func TestParseRoomID(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		expected  uint
		expectErr bool
	}{
		{
			name:      "Valid room ID",
			input:     "123",
			expected:  123,
			expectErr: false,
		},
		{
			name:      "Zero room ID",
			input:     "0",
			expected:  0,
			expectErr: false,
		},
		{
			name:      "Invalid room ID - not a number",
			input:     "invalid",
			expected:  0,
			expectErr: true,
		},
		{
			name:      "Invalid room ID - negative",
			input:     "-1",
			expected:  0,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseRoomID(tt.input)

			if tt.expectErr {
				if err == nil {
					t.Error("expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("expected %d, got %d", tt.expected, result)
				}
			}
		})
	}
}

func TestGetDisplayName(t *testing.T) {
	tests := []struct {
		name     string
		user     User
		expected string
	}{
		{
			name:     "User with Name field",
			user:     User{Name: "johndoe"},
			expected: "johndoe",
		},
		{
			name:     "User with FirstName and LastName",
			user:     User{FirstName: "John", LastName: "Doe"},
			expected: "John Doe",
		},
		{
			name:     "User with FirstName only",
			user:     User{FirstName: "John"},
			expected: "John",
		},
		{
			name:     "User with no fields",
			user:     User{},
			expected: "Anonymous",
		},
		{
			name:     "User with Name has priority over FirstName/LastName",
			user:     User{Name: "johndoe", FirstName: "John", LastName: "Doe"},
			expected: "johndoe",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getDisplayName(tt.user)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

package model

import (
	"testing"
)

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
			result, err := ParseRoomID(tt.input)

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
			result := GetDisplayName(tt.user)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestShouldStoreMessage(t *testing.T) {
	tests := []struct {
		name     string
		msgType  MessageType
		expected bool
	}{
		{
			name:     "Store system messages",
			msgType:  SystemMessage,
			expected: true,
		},
		{
			name:     "Store user messages",
			msgType:  UserMessage,
			expected: true,
		},
		{
			name:     "Do not store image messages",
			msgType:  ImageMessage,
			expected: false,
		},
		{
			name:     "Do not store user_typing events",
			msgType:  "user_typing",
			expected: false,
		},
		{
			name:     "Do not store message_updated events",
			msgType:  "message_updated",
			expected: false,
		},
		{
			name:     "Do not store custom events",
			msgType:  "custom_event",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldStoreMessage(tt.msgType)
			if result != tt.expected {
				t.Errorf("ShouldStoreMessage(%s) = %v, expected %v", tt.msgType, result, tt.expected)
			}
		})
	}
}

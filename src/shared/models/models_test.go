package models

import "testing"

func TestAPIKeyValidate(t *testing.T) {
	testCases := []struct {
		name          string
		key           APIKey
		requiredScope string
		expected      bool
	}{
		{
			name:          "Test with valid single permission",
			key:           APIKey{Scope: "users:read"},
			requiredScope: "users:read",
			expected:      true,
		},
		{
			name:          "Test with valid multiple permissions",
			key:           APIKey{Scope: "users:read,users:write"},
			requiredScope: "users:read",
			expected:      true,
		},
		{
			name:          "Test with invalid permission",
			key:           APIKey{Scope: "users:read"},
			requiredScope: "users:write",
			expected:      false,
		},
		{
			name:          "Test with wildcard permission",
			key:           APIKey{Scope: "users:*"},
			requiredScope: "users:read",
			expected:      true,
		},
		{
			name:          "Test with multiple scopes",
			key:           APIKey{Scope: "other:read,users:read"},
			requiredScope: "other:read",
			expected:      true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := APIKeyValidate(tc.key, tc.requiredScope)
			if result != tc.expected {
				t.Errorf("Expected '%v', got '%v'", tc.expected, result)
			}
		})
	}
}

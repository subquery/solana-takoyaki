package utils

import (
	"encoding/json"
	"reflect"
	"testing"
)

// Test cases for MarshalWithEmptySlices
func TestMarshalWithEmptySlices(t *testing.T) {
	// Used for bool pointer test
	flag := true

	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{
			name: "Nil slices should be omitted",
			input: struct {
				Numbers []int `json:"numbers,omitempty"`
			}{},
			expected: `{}`,
		},
		{
			name: "Empty slices should serialize as []",
			input: struct {
				Numbers []int `json:"numbers,omitempty"`
			}{Numbers: []int{}},
			expected: `{"numbers":[]}`,
		},
		{
			name: "Populated slices should serialize normally",
			input: struct {
				Numbers []int `json:"numbers,omitempty"`
			}{Numbers: []int{1, 2, 3}},
			expected: `{"numbers":[1,2,3]}`,
		},
		{
			name: "Non-slice fields should remain unchanged",
			input: struct {
				Name    string `json:"name"`
				Numbers []int  `json:"numbers,omitempty"`
			}{Name: "Alice"},
			expected: `{"name":"Alice"}`,
		},
		{
			name: "Struct with no slices should serialize correctly",
			input: struct {
				Name  string `json:"name"`
				Age   int    `json:"age"`
				Admin bool   `json:"admin"`
			}{Name: "Alice", Age: 30, Admin: true},
			expected: `{"name":"Alice","age":30,"admin":true}`,
		},
		{
			name: "Nested struct with empty slice",
			input: struct {
				Name   string `json:"name"`
				Values struct {
					Numbers []int `json:"numbers,omitempty"`
				} `json:"values"`
			}{Name: "Alice", Values: struct {
				Numbers []int `json:"numbers,omitempty"`
			}{Numbers: []int{}}},
			expected: `{"name":"Alice","values":{"numbers":[]}}`,
		},
		{
			name: "Nested struct with nil slice (should omit)",
			input: struct {
				Name   string `json:"name"`
				Values struct {
					Numbers []int `json:"numbers,omitempty"`
				} `json:"values"`
			}{Name: "Alice"},
			expected: `{"name":"Alice","values":{}}`,
		},
		{
			name: "Anonymous struct with multiple slice fields",
			input: struct {
				Numbers []int    `json:"numbers,omitempty"`
				Names   []string `json:"names,omitempty"`
			}{Numbers: []int{}, Names: []string{"Alice"}},
			expected: `{"numbers":[],"names":["Alice"]}`,
		},
		{
			name: "Struct with a map",
			input: struct {
				Records map[string]int `json:"records,omitempty"`
			}{Records: map[string]int{"Alice": 30}},
			expected: `{"records":{"Alice":30}}`,
		},
		{
			name: "Struct with an empty map",
			input: struct {
				Records map[string]int `json:"records,omitempty"`
			}{Records: map[string]int{}},
			expected: `{}`,
		},
		{
			name: "Struct with bool pointer set",
			input: struct {
				Flag *bool `json:"flag,omitempty"`
			}{Flag: &flag},
			expected: `{"flag":true}`,
		},
		{
			name: "Struct with bool pointer NOT set",
			input: struct {
				Flag *bool `json:"flag,omitempty"`
			}{},
			expected: `{}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MarshalWithEmptySlices(tt.input)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Convert result JSON to a normalized form for comparison
			var normalizedResult, normalizedExpected map[string]interface{}
			if err := json.Unmarshal(result, &normalizedResult); err != nil {
				t.Fatalf("Failed to unmarshal result: %v", err)
			}
			if err := json.Unmarshal([]byte(tt.expected), &normalizedExpected); err != nil {
				t.Fatalf("Failed to unmarshal expected: %v", err)
			}

			// Compare JSON outputs
			if !jsonDeepEqual(normalizedResult, normalizedExpected) {
				t.Errorf("Test %q failed.\nExpected: %s\nGot: %s", tt.name, tt.expected, string(result))
			}
		})
	}
}

// jsonDeepEqual compares two JSON objects.
func jsonDeepEqual(a, b map[string]interface{}) bool {
	return len(a) == len(b) && func() bool {
		for k, v := range a {
			if !reflect.DeepEqual(v, b[k]) {
				return false
			}
		}
		return true
	}()
}

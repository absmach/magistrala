// Copyright (c) Abstract Machines
// SPDX-License-Identifier: Apache-2.0

package api

import (
	"sync"
	"testing"
)

func TestNormalizeSerialNumber(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "already normalized",
			input:    "1a:2b:3c:4d",
			expected: "1a:2b:3c:4d",
		},
		{
			name:     "no separators",
			input:    "1a2b3c4d",
			expected: "1a:2b:3c:4d",
		},
		{
			name:     "with spaces",
			input:    "1a 2b 3c 4d",
			expected: "1a:2b:3c:4d",
		},
		{
			name:     "mixed separators",
			input:    "1a:2b 3c:4d",
			expected: "1a:2b:3c:4d",
		},
		{
			name:     "uppercase input",
			input:    "1A:2B:3C:4D",
			expected: "1a:2b:3c:4d",
		},
		{
			name:     "odd length - needs padding",
			input:    "1a2b3",
			expected: "01:a2:b3",
		},
		{
			name:     "single character",
			input:    "a",
			expected: "a",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "long serial number",
			input:    "01:23:45:67:89:ab:cd:ef:12:34:56:78",
			expected: "01:23:45:67:89:ab:cd:ef:12:34:56:78",
		},
		{
			name:     "complex mixed format",
			input:    "01 23:45 67:89AB cd ef",
			expected: "01:23:45:67:89:ab:cd:ef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := NormalizeSerialNumber(tt.input)
			if result != tt.expected {
				t.Errorf("NormalizeSerialNumber(%q) = %q, expected %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNormalizeSerialNumberConcurrent(t *testing.T) {
	input := "1A:2B 3C:4D"
	expected := "1a:2b:3c:4d"

	const numGoroutines = 100
	var wg sync.WaitGroup
	results := make(chan string, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result := NormalizeSerialNumber(input)
			results <- result
		}()
	}

	wg.Wait()
	close(results)

	for result := range results {
		if result != expected {
			t.Errorf("Concurrent execution failed: got %q, expected %q", result, expected)
		}
	}
}

func BenchmarkNormalizeSerialNumber(b *testing.B) {
	testCases := []struct {
		name  string
		input string
	}{
		{"short", "1a2b"},
		{"medium", "1a:2b:3c:4d:5e:6f"},
		{"long", "01:23:45:67:89:ab:cd:ef:12:34:56:78:90:ab:cd:ef"},
		{"mixed_format", "01 23:45 67:89AB cd ef 12:34"},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				NormalizeSerialNumber(tc.input)
			}
		})
	}
}

func BenchmarkNormalizeSerialNumberParallel(b *testing.B) {
	input := "1A:2B 3C:4D:5E:6F:7G:8H"

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			NormalizeSerialNumber(input)
		}
	})
}

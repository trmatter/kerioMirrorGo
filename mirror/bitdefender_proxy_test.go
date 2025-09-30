package mirror

import (
	"testing"
)

func TestShouldCache(t *testing.T) {
	tests := []struct {
		name     string
		filePath string
		expected bool
	}{
		{
			name:     "versions.id should not be cached",
			filePath: "/av64bit/versions.id",
			expected: false,
		},
		{
			name:     "version.txt should not be cached",
			filePath: "/av64bit/version.txt",
			expected: false,
		},
		{
			name:     "cumulative.txt should not be cached",
			filePath: "/av64bit/cumulative.txt",
			expected: false,
		},
		{
			name:     "VERSIONS.ID (uppercase) should not be cached",
			filePath: "/av64bit/VERSIONS.ID",
			expected: false,
		},
		{
			name:     "versions.dat.gz should be cached",
			filePath: "/av64bit/versions.dat.gz",
			expected: true,
		},
		{
			name:     "update.dat should be cached",
			filePath: "/av64bit/update.dat",
			expected: true,
		},
		{
			name:     "some.cvd should be cached",
			filePath: "/av64bit/some.cvd",
			expected: true,
		},
		{
			name:     "double slash path versions.id",
			filePath: "//av64bit/versions.id",
			expected: false,
		},
		{
			name:     "nested path versions.id",
			filePath: "/av64bit/nested/versions.id",
			expected: false,
		},
		{
			name:     "just filename versions.id",
			filePath: "versions.id",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldCache(tt.filePath)
			if result != tt.expected {
				t.Errorf("shouldCache(%q) = %v, want %v", tt.filePath, result, tt.expected)
			}
		})
	}
}

func TestNonCacheableFilesList(t *testing.T) {
	// Verify the non-cacheable files list is not empty
	if len(nonCacheableFiles) == 0 {
		t.Error("nonCacheableFiles list should not be empty")
	}

	// Verify key files are in the list
	expectedFiles := []string{"versions.id", "version.txt", "cumulative.txt"}
	for _, expected := range expectedFiles {
		found := false
		for _, file := range nonCacheableFiles {
			if file == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected file '%s' not found in nonCacheableFiles list", expected)
		}
	}
}

func BenchmarkShouldCache(b *testing.B) {
	paths := []string{
		"/av64bit/versions.id",
		"/av64bit/update.dat",
		"//av64bit/versions.id",
		"/av64bit/nested/path/file.cvd",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		shouldCache(paths[i%len(paths)])
	}
}
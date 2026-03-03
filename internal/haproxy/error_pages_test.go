package haproxy

import (
import "fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerateErrorPages(t *testing.T) {
	tmpDir := t.TempDir()

	originalErrorPagesDir := ErrorPagesDir
	defer func() {
		ErrorPagesDir = originalErrorPagesDir
	}()

	ErrorPagesDir = filepath.Join(tmpDir, "errors")

	err := GenerateErrorPages()
	require.NoError(t, err)

	expectedCodes := []int{400, 403, 408, 500, 502, 503, 504}
	for _, code := range expectedCodes {
		filename := filepath.Join(ErrorPagesDir, fmt.Sprintf("%d.http", code))

		content, err := os.ReadFile(filename)
		require.NoError(t, err, "Error page %d should exist", code)

		assert.Contains(t, string(content), "HTTP/1.0")
		assert.Contains(t, string(content), string(rune(code/100+48)))
	}
}

func TestErrorPagesExist(t *testing.T) {
	tmpDir := t.TempDir()

	originalErrorPagesDir := ErrorPagesDir
	defer func() {
		ErrorPagesDir = originalErrorPagesDir
	}()

	ErrorPagesDir = filepath.Join(tmpDir, "errors")

	assert.False(t, ErrorPagesExist())

	err := GenerateErrorPages()
	require.NoError(t, err)

	assert.True(t, ErrorPagesExist())
}

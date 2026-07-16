package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopReportPath(t *testing.T) {
	t.Run("should target the Desktop when it exists", func(t *testing.T) {
		// Arrange
		home := t.TempDir()
		t.Setenv("HOME", home)
		desktop := filepath.Join(home, "Desktop")
		if err := os.Mkdir(desktop, 0o755); err != nil {
			t.Fatalf("creating Desktop: %v", err)
		}

		// Act
		got := desktopReportPath()

		// Assert
		want := filepath.Join(desktop, "complexity-radar-report.html")
		if got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
	})

	t.Run("should fall back to home when there is no Desktop", func(t *testing.T) {
		// Arrange
		home := t.TempDir()
		t.Setenv("HOME", home)

		// Act
		got := desktopReportPath()

		// Assert
		want := filepath.Join(home, "complexity-radar-report.html")
		if got != want {
			t.Fatalf("path = %q, want %q", got, want)
		}
	})
}

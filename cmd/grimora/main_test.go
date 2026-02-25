package main

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsNewerVersion(t *testing.T) {
	tests := []struct {
		latest  string
		current string
		want    bool
	}{
		{"1.0.0", "1.0.0", false},
		{"1.1.0", "1.0.0", true},
		{"1.0.0", "1.1.0", false},
		{"1.10.0", "1.9.0", true},
		{"2.0.0", "1.99.99", true},
		{"0.0.1", "0.0.2", false},
		{"v1.2.3", "1.2.3", false},
		{"v1.3.0", "v1.2.0", true},
		{"1.0.0", "0.9.0", true},
		{"1.0.1", "1.0.0", true},
		{"1.0.0", "1.0.1", false},
	}
	for _, tt := range tests {
		t.Run(tt.latest+"_vs_"+tt.current, func(t *testing.T) {
			got := isNewerVersion(tt.latest, tt.current)
			if got != tt.want {
				t.Errorf("isNewerVersion(%q, %q) = %v, want %v", tt.latest, tt.current, got, tt.want)
			}
		})
	}
}

// makeTarGz creates a tar.gz file with the given entries.
func makeTarGz(t *testing.T, dest string, entries map[string]string) {
	t.Helper()
	f, err := os.Create(dest)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	tw := tar.NewWriter(gw)

	for name, content := range entries {
		hdr := &tar.Header{
			Name:     name,
			Size:     int64(len(content)),
			Mode:     0755,
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gw.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestExtractBinary(t *testing.T) {
	t.Run("valid tarball", func(t *testing.T) {
		tmpDir := t.TempDir()
		tarPath := filepath.Join(tmpDir, "test.tar.gz")
		makeTarGz(t, tarPath, map[string]string{
			"grimora": "fake-binary-content",
		})

		dest := filepath.Join(tmpDir, "grimora")
		if err := extractBinary(tarPath, dest); err != nil {
			t.Fatalf("extractBinary() error: %v", err)
		}

		data, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("reading extracted binary: %v", err)
		}
		if string(data) != "fake-binary-content" {
			t.Errorf("extracted content = %q, want %q", string(data), "fake-binary-content")
		}
	})

	t.Run("grimora in subdir", func(t *testing.T) {
		tmpDir := t.TempDir()
		tarPath := filepath.Join(tmpDir, "test.tar.gz")
		makeTarGz(t, tarPath, map[string]string{
			"grimora_darwin_arm64/grimora": "subdir-binary",
		})

		dest := filepath.Join(tmpDir, "grimora")
		if err := extractBinary(tarPath, dest); err != nil {
			t.Fatalf("extractBinary() error: %v", err)
		}

		data, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if string(data) != "subdir-binary" {
			t.Errorf("extracted content = %q, want %q", string(data), "subdir-binary")
		}
	})

	t.Run("no matching entry", func(t *testing.T) {
		tmpDir := t.TempDir()
		tarPath := filepath.Join(tmpDir, "test.tar.gz")
		makeTarGz(t, tarPath, map[string]string{
			"other-binary": "content",
		})

		dest := filepath.Join(tmpDir, "grimora")
		err := extractBinary(tarPath, dest)
		if err == nil {
			t.Fatal("expected error for missing grimora entry")
		}
		if err.Error() != "grimora binary not found in tarball" {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty tarball", func(t *testing.T) {
		tmpDir := t.TempDir()
		tarPath := filepath.Join(tmpDir, "test.tar.gz")
		makeTarGz(t, tarPath, map[string]string{})

		dest := filepath.Join(tmpDir, "grimora")
		err := extractBinary(tarPath, dest)
		if err == nil {
			t.Fatal("expected error for empty tarball")
		}
	})
}

func TestVerifyChecksum(t *testing.T) {
	t.Run("matching checksum", func(t *testing.T) {
		tmpDir := t.TempDir()

		content := "test file content"
		filePath := filepath.Join(tmpDir, "grimora_darwin_arm64.tar.gz")
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}

		h := sha256.Sum256([]byte(content))
		checksum := hex.EncodeToString(h[:])

		checksumsPath := filepath.Join(tmpDir, "checksums.txt")
		checksumLine := checksum + "  grimora_darwin_arm64.tar.gz\n"
		if err := os.WriteFile(checksumsPath, []byte(checksumLine), 0644); err != nil {
			t.Fatal(err)
		}

		if err := verifyChecksum(filePath, checksumsPath, "grimora_darwin_arm64.tar.gz"); err != nil {
			t.Fatalf("verifyChecksum() error: %v", err)
		}
	})

	t.Run("mismatched checksum", func(t *testing.T) {
		tmpDir := t.TempDir()

		filePath := filepath.Join(tmpDir, "grimora_darwin_arm64.tar.gz")
		if err := os.WriteFile(filePath, []byte("actual content"), 0644); err != nil {
			t.Fatal(err)
		}

		checksumsPath := filepath.Join(tmpDir, "checksums.txt")
		checksumLine := "deadbeef00000000000000000000000000000000000000000000000000000000  grimora_darwin_arm64.tar.gz\n"
		if err := os.WriteFile(checksumsPath, []byte(checksumLine), 0644); err != nil {
			t.Fatal(err)
		}

		err := verifyChecksum(filePath, checksumsPath, "grimora_darwin_arm64.tar.gz")
		if err == nil {
			t.Fatal("expected error for mismatched checksum")
		}
		if got := err.Error(); !strings.Contains(got,"checksum mismatch") {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("missing entry in checksums", func(t *testing.T) {
		tmpDir := t.TempDir()

		filePath := filepath.Join(tmpDir, "grimora_darwin_arm64.tar.gz")
		if err := os.WriteFile(filePath, []byte("content"), 0644); err != nil {
			t.Fatal(err)
		}

		checksumsPath := filepath.Join(tmpDir, "checksums.txt")
		checksumLine := "abc123  grimora_linux_amd64.tar.gz\n" // different file name
		if err := os.WriteFile(checksumsPath, []byte(checksumLine), 0644); err != nil {
			t.Fatal(err)
		}

		err := verifyChecksum(filePath, checksumsPath, "grimora_darwin_arm64.tar.gz")
		if err == nil {
			t.Fatal("expected error for missing checksum entry")
		}
		if got := err.Error(); !strings.Contains(got,"no checksum found") {
			t.Errorf("unexpected error: %v", err)
		}
	})
}


package main

import (
	"archive/zip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPackageExtensions(t *testing.T) {
	tempOut := t.TempDir()
	srcDir := filepath.Join("..", "..", "extension")

	err := PackageExtensions(srcDir, tempOut)
	if err != nil {
		t.Fatalf("PackageExtensions failed: %v", err)
	}

	chromeZip := filepath.Join(tempOut, "hadesgate-shield-chrome.zip")
	firefoxXpi := filepath.Join(tempOut, "hadesgate-shield-firefox.xpi")
	firefoxZip := filepath.Join(tempOut, "hadesgate-shield-firefox.zip")

	for _, path := range []string{chromeZip, firefoxXpi, firefoxZip} {
		info, err := os.Stat(path)
		if err != nil {
			t.Errorf("expected archive %s to exist: %v", path, err)
			continue
		}
		if info.Size() < 5000 {
			t.Errorf("archive %s is suspiciously small: %d bytes", path, info.Size())
		}
	}

	// Verify Firefox manifest inside firefox XPI
	r, err := zip.OpenReader(firefoxXpi)
	if err != nil {
		t.Fatalf("failed to open firefox zip: %v", err)
	}
	defer r.Close()

	var manifestContent string
	foundFiles := make(map[string]bool)
	for _, f := range r.File {
		foundFiles[f.Name] = true
		if f.Name == "manifest.json" {
			rc, err := f.Open()
			if err != nil {
				t.Fatalf("failed to open manifest.json inside zip: %v", err)
			}
			bytes, _ := io.ReadAll(rc)
			rc.Close()
			manifestContent = string(bytes)
		}
	}

	if !foundFiles["manifest.json"] {
		t.Errorf("manifest.json missing from firefox archive")
	}
	if !foundFiles["background.js"] {
		t.Errorf("background.js missing from firefox archive")
	}
	if !foundFiles["icons/icon128.png"] {
		t.Errorf("icons/icon128.png missing from firefox archive")
	}

	// Must contain gecko ID
	if !strings.Contains(manifestContent, "shield@hadesgate.dev") {
		t.Errorf("firefox manifest must contain shield@hadesgate.dev, got: %s", manifestContent)
	}
	if !strings.Contains(manifestContent, "browser_specific_settings") {
		t.Errorf("firefox manifest must contain browser_specific_settings, got: %s", manifestContent)
	}
}


package main

import (
	"archive/zip"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

var extensionFiles = []string{
	"background.js",
	"content.js",
	"content.css",
	"popup.html",
	"popup.js",
	"popup.css",
	"icons/icon16.png",
	"icons/icon48.png",
	"icons/icon128.png",
}

func main() {
	srcDir := flag.String("src", "extension", "Path to extension directory")
	outDir := flag.String("out", filepath.Join("dist", "extension"), "Output directory for packaged extensions")
	flag.Parse()

	if err := PackageExtensions(*srcDir, *outDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error packaging extensions: %v\n", err)
		os.Exit(1)
	}
}

// PackageExtensions builds both Chromium and Firefox extension packages.
func PackageExtensions(srcDir, outDir string) error {
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// 1. Package Chrome / Chromium (Manifest V3)
	chromeZip := filepath.Join(outDir, "hadesgate-shield-chrome.zip")
	chromeManifest := filepath.Join(srcDir, "manifest.json")
	if err := buildArchive(srcDir, chromeManifest, chromeZip); err != nil {
		return fmt.Errorf("failed to build chrome package: %w", err)
	}
	fmt.Printf("[OK] Packaged Chrome extension: %s\n", chromeZip)

	// 2. Package Firefox Add-on (Gecko / WebExtensions API)
	firefoxXpi := filepath.Join(outDir, "hadesgate-shield-firefox.xpi")
	firefoxZip := filepath.Join(outDir, "hadesgate-shield-firefox.zip")
	firefoxManifest := filepath.Join(srcDir, "manifest.firefox.json")
	if _, err := os.Stat(firefoxManifest); os.IsNotExist(err) {
		// Fallback to main manifest if firefox variant doesn't exist
		firefoxManifest = chromeManifest
	}

	if err := buildArchive(srcDir, firefoxManifest, firefoxXpi); err != nil {
		return fmt.Errorf("failed to build firefox xpi package: %w", err)
	}
	fmt.Printf("[OK] Packaged Firefox extension (XPI): %s\n", firefoxXpi)

	// Also write .zip alias for Firefox Add-ons developer hub
	if err := copyFile(firefoxXpi, firefoxZip); err != nil {
		return fmt.Errorf("failed to write firefox zip alias: %w", err)
	}
	fmt.Printf("[OK] Packaged Firefox extension (ZIP): %s\n", firefoxZip)

	return nil
}

func buildArchive(srcDir, manifestPath, destZipPath string) error {
	outFile, err := os.Create(destZipPath)
	if err != nil {
		return err
	}
	defer outFile.Close()

	w := zip.NewWriter(outFile)
	defer w.Close()

	// 1. Add manifest as manifest.json
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read manifest %s: %w", manifestPath, err)
	}
	manifestWriter, err := w.Create("manifest.json")
	if err != nil {
		return err
	}
	if _, err := manifestWriter.Write(manifestBytes); err != nil {
		return err
	}

	// 2. Add all extension assets
	for _, relPath := range extensionFiles {
		fullPath := filepath.Join(srcDir, filepath.FromSlash(relPath))
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read extension asset %s: %w", fullPath, err)
		}
		// In zip archives, slashes must be forward slashes
		zipEntryName := strings.ReplaceAll(relPath, "\\", "/")
		writer, err := w.Create(zipEntryName)
		if err != nil {
			return err
		}
		if _, err := writer.Write(data); err != nil {
			return err
		}
	}

	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}


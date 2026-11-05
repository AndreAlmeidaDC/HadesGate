package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestBuildZipAndTarArchive(t *testing.T) {
	tempDir := t.TempDir()

	dummyBin := filepath.Join(tempDir, "hadesgate.exe")
	if err := os.WriteFile(dummyBin, []byte("fake-binary-content-12345"), 0755); err != nil {
		t.Fatalf("failed to create dummy binary: %v", err)
	}

	// 1. Test ZIP archive
	zipPath := filepath.Join(tempDir, "release.zip")
	if err := buildZipArchive(dummyBin, "hadesgate.exe", zipPath); err != nil {
		t.Fatalf("buildZipArchive failed: %v", err)
	}

	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("failed to open zip: %v", err)
	}
	defer zr.Close()

	foundBinInZip := false
	for _, f := range zr.File {
		if f.Name == "hadesgate.exe" {
			foundBinInZip = true
		}
	}
	if !foundBinInZip {
		t.Errorf("hadesgate.exe not found in zip archive")
	}

	// 2. Test TarGz archive
	tarPath := filepath.Join(tempDir, "release.tar.gz")
	if err := buildTarGzArchive(dummyBin, "hadesgate", tarPath); err != nil {
		t.Fatalf("buildTarGzArchive failed: %v", err)
	}

	tf, err := os.Open(tarPath)
	if err != nil {
		t.Fatalf("failed to open tar.gz: %v", err)
	}
	defer tf.Close()

	gzr, err := gzip.NewReader(tf)
	if err != nil {
		t.Fatalf("failed to read gzip: %v", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	foundBinInTar := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar read error: %v", err)
		}
		if hdr.Name == "hadesgate" {
			foundBinInTar = true
		}
	}
	if !foundBinInTar {
		t.Errorf("hadesgate not found in tar.gz archive")
	}

	// 3. Test SHA256 calculation
	hash, err := fileSHA256(zipPath)
	if err != nil || len(hash) != 64 {
		t.Errorf("invalid sha256 hash: %s (err: %v)", hash, err)
	}
}


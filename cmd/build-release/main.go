package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

type Target struct {
	OS       string
	Arch     string
	Ext      string
	IsZip    bool
}

var releaseTargets = []Target{
	{OS: "linux", Arch: "amd64", Ext: "", IsZip: false},
	{OS: "linux", Arch: "arm64", Ext: "", IsZip: false},
	{OS: "darwin", Arch: "amd64", Ext: "", IsZip: false},
	{OS: "darwin", Arch: "arm64", Ext: "", IsZip: false},
	{OS: "windows", Arch: "amd64", Ext: ".exe", IsZip: true},
}

func main() {
	versionFlag := flag.String("version", "v2.0.0", "Release version tag (e.g. v2.0.0)")
	outDirFlag := flag.String("out", filepath.Join("dist", "releases"), "Output directory for release archives")
	flag.Parse()

	version := *versionFlag
	outDir := *outDirFlag

	fmt.Println("==================================================")
	fmt.Printf("   HADESGATE // MULTI-PLATFORM RELEASE BUILDER    \n")
	fmt.Printf("   Version: %s | Target: %s\n", version, outDir)
	fmt.Println("==================================================")

	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output dir: %v\n", err)
		os.Exit(1)
	}

	tempDir, err := os.MkdirTemp("", "hadesgate-build-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating temp dir: %v\n", err)
		os.Exit(1)
	}
	defer os.RemoveAll(tempDir)

	var generatedArchives []string

	now := time.Now().UTC().Format(time.RFC3339)
	ldflags := fmt.Sprintf("-s -w -X main.version=%s -X main.buildDate=%s", version, now)

	for _, target := range releaseTargets {
		binName := "hadesgate" + target.Ext
		binPath := filepath.Join(tempDir, fmt.Sprintf("hadesgate-%s-%s%s", target.OS, target.Arch, target.Ext))

		fmt.Printf("[BUILD] Compiling for %s/%s...\n", target.OS, target.Arch)

		cmd := exec.Command("go", "build", "-trimpath", "-ldflags", ldflags, "-o", binPath, "./cmd/hadesgate")
		cmd.Env = append(os.Environ(),
			"CGO_ENABLED=0",
			"GOOS="+target.OS,
			"GOARCH="+target.Arch,
		)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to compile %s/%s: %v\n", target.OS, target.Arch, err)
			os.Exit(1)
		}

		var archiveName string
		var archivePath string

		if target.IsZip {
			archiveName = fmt.Sprintf("hadesgate-%s-%s.zip", target.OS, target.Arch)
			archivePath = filepath.Join(outDir, archiveName)
			if err := buildZipArchive(binPath, binName, archivePath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to build zip %s: %v\n", archivePath, err)
				os.Exit(1)
			}
		} else {
			archiveName = fmt.Sprintf("hadesgate-%s-%s.tar.gz", target.OS, target.Arch)
			archivePath = filepath.Join(outDir, archiveName)
			if err := buildTarGzArchive(binPath, binName, archivePath); err != nil {
				fmt.Fprintf(os.Stderr, "Failed to build tar.gz %s: %v\n", archivePath, err)
				os.Exit(1)
			}
		}

		generatedArchives = append(generatedArchives, archivePath)
		fmt.Printf("[OK] Generated: %s\n", archivePath)
	}

	// Also package browser extensions into release directory if extension dir exists
	extOut := filepath.Join(outDir, "extensions")
	_ = os.MkdirAll(extOut, 0755)
	extPkgCmd := exec.Command("go", "run", "./cmd/package-extension", "-out", outDir)
	extPkgCmd.Stdout = os.Stdout
	extPkgCmd.Stderr = os.Stderr
	if err := extPkgCmd.Run(); err == nil {
		for _, name := range []string{"hadesgate-shield-chrome.zip", "hadesgate-shield-firefox.xpi"} {
			p := filepath.Join(outDir, name)
			if _, err := os.Stat(p); err == nil {
				generatedArchives = append(generatedArchives, p)
			}
		}
	}

	// Compute SHA256SUMS
	checksumPath := filepath.Join(outDir, "SHA256SUMS")
	checksumFile, err := os.Create(checksumPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create SHA256SUMS: %v\n", err)
		os.Exit(1)
	}
	defer checksumFile.Close()

	fmt.Println("\n--- SHA256 CHECKSUMS ---")
	for _, p := range generatedArchives {
		hash, err := fileSHA256(p)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to hash %s: %v\n", p, err)
			continue
		}
		line := fmt.Sprintf("%s  %s\n", hash, filepath.Base(p))
		checksumFile.WriteString(line)
		fmt.Print(line)
	}
	fmt.Printf("\n[SUCCESS] Release build complete! Checksums saved to: %s\n", checksumPath)
}

func buildZipArchive(binPath, binName, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	// Add binary
	if err := addFileToZip(zw, binPath, binName); err != nil {
		return err
	}

	// Add auxiliary files if present
	for _, aux := range []string{"README.md", "LICENSE", ".env.example"} {
		if _, err := os.Stat(aux); err == nil {
			_ = addFileToZip(zw, aux, aux)
		}
	}

	return nil
}

func addFileToZip(zw *zip.Writer, srcPath, entryName string) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = entryName
	header.Method = zip.Deflate

	w, err := zw.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(w, src)
	return err
}

func buildTarGzArchive(binPath, binName, outPath string) error {
	f, err := os.Create(outPath)
	if err != nil {
		return err
	}
	defer f.Close()

	gw := gzip.NewWriter(f)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	// Add binary
	if err := addFileToTar(tw, binPath, binName, 0755); err != nil {
		return err
	}

	// Add auxiliary files
	for _, aux := range []string{"README.md", "LICENSE", ".env.example"} {
		if _, err := os.Stat(aux); err == nil {
			_ = addFileToTar(tw, aux, aux, 0644)
		}
	}

	return nil
}

func addFileToTar(tw *tar.Writer, srcPath, entryName string, mode int64) error {
	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return err
	}

	hdr := &tar.Header{
		Name:    entryName,
		Mode:    mode,
		Size:    info.Size(),
		ModTime: info.ModTime(),
	}

	if err := tw.WriteHeader(hdr); err != nil {
		return err
	}

	_, err = io.Copy(tw, src)
	return err
}

func fileSHA256(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func init() {
	_ = runtime.GOOS
}

package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	libraryPath := flag.String("library", "", "path to the compiled plugin library")
	entryName := flag.String("entry", "", "dynamic library name inside the zip")
	archivePath := flag.String("archive", "", "path to the output zip archive")
	checksumPath := flag.String("checksum", "", "path to the output checksum file")
	verify := flag.Bool("verify", false, "verify an existing archive against its checksum")
	flag.Parse()

	if *verify {
		if *archivePath == "" || *checksumPath == "" || *entryName == "" {
			fatalf("archive, checksum, and entry are required for verification")
		}
		if err := verifyArchive(*archivePath, *checksumPath, *entryName); err != nil {
			fatalf("%v", err)
		}
		fmt.Printf("All checks passed: %s\n", filepath.Base(*archivePath))
		return
	}

	if *libraryPath == "" || *entryName == "" || *archivePath == "" || *checksumPath == "" {
		fatalf("library, entry, archive, and checksum are required")
	}
	if filepath.Base(*entryName) != *entryName {
		fatalf("entry must be a root-level filename")
	}
	archiveData, errPackage := packageLibrary(*libraryPath, *entryName, *archivePath)
	if errPackage != nil {
		fatalf("%v", errPackage)
	}
	checksum := sha256.Sum256(archiveData)
	line := fmt.Sprintf("%s  %s\n", hex.EncodeToString(checksum[:]), filepath.Base(*archivePath))
	if errWrite := os.WriteFile(*checksumPath, []byte(line), 0o644); errWrite != nil {
		fatalf("write checksum: %v", errWrite)
	}
}

func packageLibrary(libraryPath, entryName, archivePath string) ([]byte, error) {
	library, errOpen := os.Open(libraryPath)
	if errOpen != nil {
		return nil, fmt.Errorf("open library: %w", errOpen)
	}
	defer library.Close()

	if errMkdir := os.MkdirAll(filepath.Dir(archivePath), 0o755); errMkdir != nil {
		return nil, fmt.Errorf("create archive directory: %w", errMkdir)
	}
	archive, errCreate := os.Create(archivePath)
	if errCreate != nil {
		return nil, fmt.Errorf("create archive: %w", errCreate)
	}
	archiveClosed := false
	defer func() {
		if !archiveClosed {
			_ = archive.Close()
		}
	}()

	writer := zip.NewWriter(archive)
	header := &zip.FileHeader{Name: entryName, Method: zip.Deflate}
	header.SetMode(0o755)
	entry, errEntry := writer.CreateHeader(header)
	if errEntry != nil {
		return nil, fmt.Errorf("create zip entry: %w", errEntry)
	}
	if _, errCopy := io.Copy(entry, library); errCopy != nil {
		return nil, fmt.Errorf("copy library: %w", errCopy)
	}
	if errClose := writer.Close(); errClose != nil {
		return nil, fmt.Errorf("close zip writer: %w", errClose)
	}
	if errClose := archive.Close(); errClose != nil {
		return nil, fmt.Errorf("close archive: %w", errClose)
	}
	archiveClosed = true

	data, errRead := os.ReadFile(archivePath)
	if errRead != nil {
		return nil, fmt.Errorf("read archive: %w", errRead)
	}
	return data, nil
}

func verifyArchive(archivePath, checksumPath, expectedEntry string) error {
	archiveData, err := os.ReadFile(archivePath)
	if err != nil {
		return fmt.Errorf("read archive: %w", err)
	}
	// Verify SHA-256 checksum.
	checksumData, err := os.ReadFile(checksumPath)
	if err != nil {
		return fmt.Errorf("read checksum file: %w", err)
	}
	parts := strings.Fields(strings.TrimSpace(string(checksumData)))
	if len(parts) < 2 {
		return fmt.Errorf("invalid checksum file format")
	}
	expectedChecksum := parts[0]
	hash := sha256.Sum256(archiveData)
	actualChecksum := hex.EncodeToString(hash[:])
	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: %s != %s", actualChecksum, expectedChecksum)
	}
	// Verify zip contains exactly the expected entry.
	zipReader, err := zip.NewReader(bytes.NewReader(archiveData), int64(len(archiveData)))
	if err != nil {
		return fmt.Errorf("open zip for verification: %w", err)
	}
	if len(zipReader.File) != 1 {
		return fmt.Errorf("expected 1 entry in zip, got %d", len(zipReader.File))
	}
	if zipReader.File[0].Name != expectedEntry {
		return fmt.Errorf("expected entry %q, got %q", expectedEntry, zipReader.File[0].Name)
	}
	return nil
}

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}

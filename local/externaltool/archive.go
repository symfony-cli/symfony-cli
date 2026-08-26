/*
 * Copyright (c) 2026-present Fabien Potencier <fabien@symfony.com>
 *
 * This file is part of Symfony CLI project
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with this program. If not, see <http://www.gnu.org/licenses/>.
 */

package externaltool

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

const (
	maximumExtractedSize = 1 << 30
	maximumArchiveFiles  = 10_000
)

func extractArchive(archivePath, assetName, destination string) error {
	switch {
	case strings.HasSuffix(assetName, ".tar.gz"):
		return extractTarGz(archivePath, destination)
	case strings.HasSuffix(assetName, ".zip"):
		return extractZip(archivePath, destination)
	default:
		return errors.New("unsupported archive format")
	}
}

func extractTarGz(archivePath, destination string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return fmt.Errorf("unable to open archive: %w", err)
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("unable to decompress archive: %w", err)
	}
	defer compressed.Close()

	reader := tar.NewReader(compressed)
	seen := map[string]bool{}
	var extractedSize int64
	files := 0
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("unable to read archive: %w", err)
		}
		if header == nil {
			continue
		}
		path, err := archivePathname(destination, header.Name)
		if err != nil {
			return err
		}
		if seen[path] {
			return fmt.Errorf("archive contains duplicate path %q", header.Name)
		}
		seen[path] = true
		files++
		if files > maximumArchiveFiles {
			return errors.New("archive contains too many files")
		}

		switch header.Typeflag {
		case tar.TypeXHeader, tar.TypeXGlobalHeader:
			continue
		case tar.TypeDir:
			if err := os.MkdirAll(path, os.FileMode(header.Mode)&0777); err != nil {
				return fmt.Errorf("unable to create archive directory: %w", err)
			}
		case tar.TypeReg:
			if header.Size < 0 || extractedSize+header.Size > maximumExtractedSize {
				return errors.New("archive expands beyond the allowed size")
			}
			extractedSize += header.Size
			if err := writeArchiveFile(path, os.FileMode(header.Mode)&0777, reader, header.Size); err != nil {
				return err
			}
		default:
			return fmt.Errorf("archive contains unsupported entry %q", header.Name)
		}
	}
}

func extractZip(archivePath, destination string) error {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return fmt.Errorf("unable to open archive: %w", err)
	}
	defer reader.Close()
	if len(reader.File) > maximumArchiveFiles {
		return errors.New("archive contains too many files")
	}

	seen := map[string]bool{}
	var extractedSize uint64
	for _, file := range reader.File {
		path, err := archivePathname(destination, file.Name)
		if err != nil {
			return err
		}
		if seen[path] {
			return fmt.Errorf("archive contains duplicate path %q", file.Name)
		}
		seen[path] = true
		mode := file.Mode()
		if mode.IsDir() {
			if err := os.MkdirAll(path, mode.Perm()); err != nil {
				return fmt.Errorf("unable to create archive directory: %w", err)
			}
			continue
		}
		if !mode.IsRegular() {
			return fmt.Errorf("archive contains unsupported entry %q", file.Name)
		}
		if extractedSize+file.UncompressedSize64 > maximumExtractedSize {
			return errors.New("archive expands beyond the allowed size")
		}
		extractedSize += file.UncompressedSize64
		contents, err := file.Open()
		if err != nil {
			return fmt.Errorf("unable to read archive entry: %w", err)
		}
		err = writeArchiveFile(path, mode.Perm(), contents, int64(file.UncompressedSize64))
		contents.Close()
		if err != nil {
			return err
		}
	}

	return nil
}

func archivePathname(root, name string) (string, error) {
	if !safeRelativePath(name) {
		return "", fmt.Errorf("archive contains unsafe path %q", name)
	}
	path := filepath.Join(root, filepath.FromSlash(strings.ReplaceAll(name, "\\", "/")))
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("archive contains unsafe path %q", name)
	}

	return path, nil
}

func safeRelativePath(value string) bool {
	value = strings.ReplaceAll(value, "\\", "/")
	if value == "" || strings.HasPrefix(value, "/") || strings.Contains(value, ":") {
		return false
	}
	clean := path.Clean(value)

	return clean != "." && clean != ".." && !strings.HasPrefix(clean, "../")
}

func writeArchiveFile(path string, mode os.FileMode, source io.Reader, size int64) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("unable to create archive directory: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("unable to create archive file: %w", err)
	}
	written, copyErr := io.CopyN(file, source, size)
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("unable to extract archive file: %w", copyErr)
	}
	if written != size {
		return errors.New("archive entry is truncated")
	}

	if closeErr != nil {
		return fmt.Errorf("unable to close archive file: %w", closeErr)
	}

	return nil
}

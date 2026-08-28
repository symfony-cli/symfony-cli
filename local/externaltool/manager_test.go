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
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

type fileVersionRunner struct{}

func (fileVersionRunner) Output(_ context.Context, executable string, _ ...string) ([]byte, error) {
	output, err := os.ReadFile(executable)
	if err != nil {
		return nil, fmt.Errorf("unable to read executable: %w", err)
	}

	return output, nil
}

type releaseFixture struct {
	mu sync.Mutex

	server       *httptest.Server
	version      string
	assetName    string
	archive      []byte
	checksum     string
	archiveDelay time.Duration
	failRelease  bool
	releaseCalls int
	archiveCalls int
}

func newReleaseFixture(t *testing.T, version, assetName string, archive []byte) *releaseFixture {
	t.Helper()
	fixture := &releaseFixture{
		version:   version,
		assetName: assetName,
		archive:   archive,
	}
	fixture.checksum = checksum(archive)
	fixture.server = httptest.NewServer(http.HandlerFunc(fixture.handle))
	t.Cleanup(fixture.server.Close)

	return fixture
}

func (f *releaseFixture) handle(response http.ResponseWriter, request *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch request.URL.Path {
	case "/repos/acme/tool/releases/latest":
		f.releaseCalls++
		if f.failRelease {
			http.Error(response, "offline", http.StatusServiceUnavailable)
			return
		}
		_ = json.NewEncoder(response).Encode(release{
			TagName: f.version,
			Assets: []releaseAsset{
				{Name: f.assetName, URL: f.server.URL + "/archive"},
				{Name: "SHA256SUMS", URL: f.server.URL + "/checksums"},
			},
		})
	case "/checksums":
		fmt.Fprintf(response, "%s  %s\n", f.checksum, f.assetName)
	case "/archive":
		f.archiveCalls++
		if f.archiveDelay > 0 {
			time.Sleep(f.archiveDelay)
		}
		_, _ = response.Write(f.archive)
	default:
		http.NotFound(response, request)
	}
}

func (f *releaseFixture) counts() (int, int) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.releaseCalls, f.archiveCalls
}

func (f *releaseFixture) publish(version string, archive []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.version = version
	f.assetName = fmt.Sprintf("tool-v%s-linux-x64.tar.gz", version)
	f.archive = archive
	f.checksum = checksum(archive)
}

func (f *releaseFixture) setChecksum(value string) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.checksum = value
}

func (f *releaseFixture) setOffline() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failRelease = true
}

func (f *releaseFixture) setArchiveDelay(delay time.Duration) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.archiveDelay = delay
}

func TestManagerRejectsInvalidDefinitions(t *testing.T) {
	manager := NewManager()
	manager.RuntimeOS = "linux"
	manager.RuntimeArch = "amd64"
	valid := testDefinition(t)

	invalidRepository := valid
	invalidRepository.Repository = "../tool"
	invalidVersion := valid
	invalidVersion.MinimumVersion = "invalid"
	definitions := map[string]struct {
		definition Definition
		message    string
	}{
		"incomplete":         {definition: Definition{}, message: "definition is incomplete"},
		"invalid repository": {definition: invalidRepository, message: "repository is invalid"},
		"invalid version":    {definition: invalidVersion, message: "minimum version is invalid"},
	}
	for name, test := range definitions {
		t.Run(name, func(t *testing.T) {
			if _, err := manager.validateDefinition(test.definition); err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected %q error, got %v", test.message, err)
			}
		})
	}

	packages := map[string]Package{
		"unsafe executable":   {OS: "linux", Arch: "amd64", Asset: "tool.tar.gz", Executable: "../tool"},
		"unsafe archive root": {OS: "linux", Arch: "amd64", Asset: "tool.tar.gz", ArchiveRoot: "../root", Executable: "tool"},
		"unsupported archive": {OS: "linux", Arch: "amd64", Asset: "tool.bin", Executable: "tool"},
	}
	for name, pkg := range packages {
		t.Run(name, func(t *testing.T) {
			definition := valid
			definition.Packages = []Package{pkg}
			if _, err := manager.packageFor(definition); err == nil {
				t.Fatal("expected package validation failure")
			}
		})
	}
}

func TestManagerInstallsCompleteDistributionAndReusesIt(t *testing.T) {
	archive := tarGz(t, map[string]archiveFile{
		"tool-v1.2.0/tool":    {contents: "Acme Tool 1.2.0\n", mode: 0755},
		"tool-v1.2.0/LICENSE": {contents: "license", mode: 0644},
		"README":              {contents: "release notes", mode: 0644},
	})
	fixture := newReleaseFixture(t, "v1.2.0", "tool-v1.2.0-linux-x64.tar.gz", archive)
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	manager, definition, stderr := testManager(t, fixture, now)

	first := resolveTool(t, manager, definition)
	if first.Version != "1.2.0" {
		t.Fatalf("unexpected version %q", first.Version)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(first.Executable), "LICENSE")); err != nil {
		t.Fatalf("distribution license was not retained: %v", err)
	}
	if got := stderr.String(); got != "Downloading Acme Tool 1.2.0\n" {
		t.Fatalf("unexpected progress output %q", got)
	}
	orphanDownload := filepath.Join(definition.InstallRoot, ".download-orphan")
	orphanInstall := filepath.Join(definition.InstallRoot, "versions", ".install-orphan")
	if err := os.WriteFile(orphanDownload, []byte("partial"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(orphanInstall, 0700); err != nil {
		t.Fatal(err)
	}
	old := now.Add(-48 * time.Hour)
	if err := os.Chtimes(orphanDownload, old, old); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(orphanInstall, old, old); err != nil {
		t.Fatal(err)
	}

	manager.Now = func() time.Time { return now.Add(time.Hour) }
	second := resolveTool(t, manager, definition)
	if second != first {
		t.Fatalf("installation changed between cached runs: %#v != %#v", second, first)
	}
	if _, err := os.Stat(orphanDownload); !os.IsNotExist(err) {
		t.Fatalf("stale download was not removed: %v", err)
	}
	if _, err := os.Stat(orphanInstall); !os.IsNotExist(err) {
		t.Fatalf("stale installation was not removed: %v", err)
	}
	if releaseCalls, archiveCalls := fixture.counts(); releaseCalls != 1 || archiveCalls != 1 {
		t.Fatalf("unexpected downloads: releases=%d archives=%d", releaseCalls, archiveCalls)
	}
}

func TestManagerChecksAtMostOnceEveryTwentyFourHoursAndFallsBackOffline(t *testing.T) {
	fixture := newToolFixture(t, "1.2.0")
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	manager, definition, _ := testManager(t, fixture, now)
	installed := resolveTool(t, manager, definition)

	fixture.setOffline()
	manager.Now = func() time.Time { return now.Add(25 * time.Hour) }
	fallback := resolveTool(t, manager, definition)
	if fallback != installed {
		t.Fatalf("offline fallback changed the installation: %#v != %#v", fallback, installed)
	}
	manager.Now = func() time.Time { return now.Add(26 * time.Hour) }
	resolveTool(t, manager, definition)
	if releaseCalls, archiveCalls := fixture.counts(); releaseCalls != 2 || archiveCalls != 1 {
		t.Fatalf("unexpected downloads after offline fallback: releases=%d archives=%d", releaseCalls, archiveCalls)
	}
}

func TestManagerReusesManagedInstallationWhenLockCannotBeAcquired(t *testing.T) {
	fixture := newToolFixture(t, "1.2.0")
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	manager, definition, stderr := testManager(t, fixture, now)
	installed := resolveTool(t, manager, definition)
	lockPath := filepath.Join(definition.InstallRoot, "install.lock")
	if err := os.Remove(lockPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(lockPath, 0700); err != nil {
		t.Fatal(err)
	}

	fallback := resolveTool(t, manager, definition)
	if fallback != installed {
		t.Fatalf("lock failure did not reuse the managed installation: %#v != %#v", fallback, installed)
	}
	if !strings.Contains(stderr.String(), "Unable to lock the Acme Tool installation") {
		t.Fatalf("lock failure was not reported: %q", stderr.String())
	}
}

func TestManagerDoesNotTrustFutureUpdateTimestamps(t *testing.T) {
	fixture := newToolFixture(t, "1.2.0")
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	manager, definition, _ := testManager(t, fixture, now)
	installed := resolveTool(t, manager, definition)
	statePath := filepath.Join(definition.InstallRoot, "state.json")
	state := loadState(statePath)
	if state == nil {
		t.Fatal("installation state was not created")
	}
	state.CheckedAt = now.Add(365 * 24 * time.Hour).Unix()
	if err := storeState(statePath, state); err != nil {
		t.Fatal(err)
	}
	fixture.setOffline()
	manager.Now = func() time.Time { return now.Add(time.Hour) }

	fallback := resolveTool(t, manager, definition)
	if fallback != installed {
		t.Fatalf("future timestamp discarded the working installation: %#v != %#v", fallback, installed)
	}
	if releaseCalls, _ := fixture.counts(); releaseCalls != 2 {
		t.Fatalf("future timestamp suppressed the update check: %d", releaseCalls)
	}
}

func TestManagerPreservesWorkingInstallationAfterFailedUpdate(t *testing.T) {
	tests := map[string]struct {
		archive  func(*testing.T) []byte
		checksum string
		message  string
	}{
		"checksum mismatch": {archive: func(t *testing.T) []byte { return toolArchive(t, "1.3.0") }, checksum: "not-a-checksum", message: "checksum mismatch"},
		"corrupted archive": {archive: func(*testing.T) []byte { return []byte("interrupted archive") }, message: "unable to extract"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newToolFixture(t, "1.2.0")
			now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
			manager, definition, stderr := testManager(t, fixture, now)
			installed := resolveTool(t, manager, definition)

			fixture.publish("1.3.0", test.archive(t))
			if test.checksum != "" {
				fixture.setChecksum(test.checksum)
			}
			manager.Now = func() time.Time { return now.Add(25 * time.Hour) }

			fallback := resolveTool(t, manager, definition)
			if fallback != installed {
				t.Fatalf("failed update changed the installation: %#v != %#v", fallback, installed)
			}
			if !strings.Contains(stderr.String(), test.message) {
				t.Fatalf("update failure was not reported: %q", stderr.String())
			}
			if output, err := os.ReadFile(installed.Executable); err != nil || string(output) != "Acme Tool 1.2.0\n" {
				t.Fatalf("working installation was not preserved: output=%q err=%v", output, err)
			}
			state := loadState(filepath.Join(definition.InstallRoot, "state.json"))
			if state == nil || state.Version != "1.2.0" {
				t.Fatalf("unexpected active state %#v", state)
			}
			downloads, err := filepath.Glob(filepath.Join(definition.InstallRoot, ".download-*"))
			if err != nil {
				t.Fatal(err)
			}
			if len(downloads) != 0 {
				t.Fatalf("temporary downloads were not removed: %#v", downloads)
			}
		})
	}
}

func TestManagerPrunesInactiveVersionsAfterGracePeriod(t *testing.T) {
	fixture := newToolFixture(t, "1.2.0")
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	manager, definition, _ := testManager(t, fixture, now)
	initial := resolveTool(t, manager, definition)
	initialDirectory := filepath.Dir(initial.Executable)

	fixture.publish("1.3.0", toolArchive(t, "1.3.0"))
	manager.Now = func() time.Time { return now.Add(25 * time.Hour) }
	updated := resolveTool(t, manager, definition)
	if updated.Version != "1.3.0" {
		t.Fatalf("unexpected updated version %q", updated.Version)
	}
	if _, err := os.Stat(initialDirectory); err != nil {
		t.Fatalf("previous version was removed before the grace period: %v", err)
	}

	manager.Now = func() time.Time { return now.Add(74 * time.Hour) }
	resolveTool(t, manager, definition)
	if _, err := os.Stat(initialDirectory); !os.IsNotExist(err) {
		t.Fatalf("inactive version was not pruned: %v", err)
	}
	if _, err := os.Stat(updated.Executable); err != nil {
		t.Fatalf("active version was pruned: %v", err)
	}
}

func TestManagerRejectsInvalidInstallations(t *testing.T) {
	tests := map[string][]byte{
		"wrong version": tarGz(t, map[string]archiveFile{
			"tool-v1.2.0/tool": {contents: "Acme Tool 1.1.0\n", mode: 0755},
		}),
		"unexpected layout": tarGz(t, map[string]archiveFile{
			"other/tool": {contents: "Acme Tool 1.2.0\n", mode: 0755},
		}),
	}
	for name, archive := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newReleaseFixture(t, "1.2.0", "tool-v1.2.0-linux-x64.tar.gz", archive)
			manager, definition, _ := testManager(t, fixture, time.Now())
			if _, err := manager.Resolve(context.Background(), definition); err == nil {
				t.Fatal("expected installation failure")
			}
			if _, err := os.Stat(filepath.Join(definition.InstallRoot, "state.json")); !os.IsNotExist(err) {
				t.Fatalf("failed installation created an active state: %v", err)
			}
		})
	}
}

func TestExtractTarGzRejectsUnsafePaths(t *testing.T) {
	tests := map[string]string{
		"parent directory":  "../tool",
		"Windows traversal": "..\\tool",
		"Windows drive":     "C:/tool",
	}
	for name, entry := range tests {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			archivePath := filepath.Join(directory, "archive.tar.gz")
			if err := os.WriteFile(archivePath, tarGz(t, map[string]archiveFile{
				entry: {contents: "contents", mode: 0600},
			}), 0600); err != nil {
				t.Fatal(err)
			}

			err := extractTarGz(archivePath, filepath.Join(directory, "extracted"))
			if err == nil || !strings.Contains(err.Error(), "unsafe path") {
				t.Fatalf("expected unsafe path rejection, got %v", err)
			}
			if _, err := os.Stat(filepath.Join(directory, "tool")); !os.IsNotExist(err) {
				t.Fatalf("archive escaped the extraction directory: %v", err)
			}
		})
	}
}

func TestExtractTarGzRejectsOverflowingExpandedSize(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "archive.tar.gz")
	if err := os.WriteFile(archivePath, overflowingTarGz(t), 0600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(directory, "extracted")

	err := extractTarGz(archivePath, destination)
	if err == nil || !strings.Contains(err.Error(), "archive expands beyond the allowed size") {
		t.Fatalf("expected expanded size rejection, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(destination, "huge")); !os.IsNotExist(err) {
		t.Fatalf("oversized archive entry was created: %v", err)
	}
}

func TestExtractTarGzIgnoresGlobalPAXHeaders(t *testing.T) {
	directory := t.TempDir()
	archivePath := filepath.Join(directory, "archive.tar.gz")
	if err := os.WriteFile(archivePath, tarGzWithGlobalPAXHeader(t), 0600); err != nil {
		t.Fatal(err)
	}
	destination := filepath.Join(directory, "extracted")

	if err := extractTarGz(archivePath, destination); err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(filepath.Join(destination, "tool"))
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != "contents" {
		t.Fatalf("unexpected extracted contents %q", contents)
	}
}

func TestManagerInstallsZipAndFlatArchives(t *testing.T) {
	tests := map[string]struct {
		runtimeOS string
		asset     string
		archive   []byte
		pkg       Package
		retained  string
	}{
		"zip": {
			runtimeOS: "windows",
			asset:     "tool-v1.2.0-windows-x64.zip",
			archive: zipArchive(t, map[string]archiveFile{
				"tool-v1.2.0/tool.exe": {contents: "Acme Tool 1.2.0\n", mode: 0644},
				"tool-v1.2.0/NOTICE":   {contents: "notice", mode: 0644},
			}),
			pkg:      Package{OS: "windows", Arch: "amd64", Asset: "tool-v{version}-windows-x64.zip", ArchiveRoot: "tool-v{version}", Executable: "tool.exe"},
			retained: "NOTICE",
		},
		"flat tarball": {
			runtimeOS: "linux",
			asset:     "tool-v1.2.0-linux-x64.tar.gz",
			archive: tarGz(t, map[string]archiveFile{
				"tool":   {contents: "Acme Tool 1.2.0\n", mode: 0755},
				"NOTICE": {contents: "notice", mode: 0644},
			}),
			pkg:      Package{OS: "linux", Arch: "amd64", Asset: "tool-v{version}-linux-x64.tar.gz", Executable: "tool"},
			retained: "NOTICE",
		},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			fixture := newReleaseFixture(t, "1.2.0", test.asset, test.archive)
			manager, definition, _ := testManager(t, fixture, time.Now())
			manager.RuntimeOS = test.runtimeOS
			definition.Packages = []Package{test.pkg}

			installation := resolveTool(t, manager, definition)
			if filepath.Base(installation.Executable) != test.pkg.Executable {
				t.Fatalf("unexpected executable %q", installation.Executable)
			}
			if _, err := os.Stat(filepath.Join(filepath.Dir(installation.Executable), test.retained)); err != nil {
				t.Fatalf("distribution file was not retained: %v", err)
			}
		})
	}
}

func TestManagerUsesASeparateArchiveDownloadTimeout(t *testing.T) {
	fixture := newToolFixture(t, "1.2.0")
	fixture.setArchiveDelay(150 * time.Millisecond)
	manager, definition, _ := testManager(t, fixture, time.Now())
	manager.MetadataTimeout = 50 * time.Millisecond
	manager.ArchiveTimeout = 5 * time.Second

	installation := resolveTool(t, manager, definition)
	if installation.Version != "1.2.0" {
		t.Fatalf("unexpected version %q", installation.Version)
	}
}

func TestManagerFallsBackToLegacyOnUnsupportedPlatform(t *testing.T) {
	fixture := newToolFixture(t, "1.0.0")
	manager, definition, _ := testManager(t, fixture, time.Now())
	manager.RuntimeArch = "386"

	if _, err := manager.Resolve(context.Background(), definition); err == nil || !strings.Contains(err.Error(), "unavailable on linux/386") {
		t.Fatalf("expected unsupported platform failure, got %v", err)
	}
	legacy := legacyExecutable(t, "Acme Tool 1.0.0\n")
	definition.LegacyExecutables = []string{legacy}
	installation := resolveTool(t, manager, definition)
	if installation.Executable != legacy {
		t.Fatalf("unsupported platform did not reuse the legacy executable: %#v", installation)
	}
}

func TestManagerRejectsReleasesBelowTheMinimumVersion(t *testing.T) {
	fixture := newToolFixture(t, "1.0.0")
	manager, definition, _ := testManager(t, fixture, time.Now())
	definition.MinimumVersion = "1.1.0"

	if _, err := manager.Resolve(context.Background(), definition); err == nil || !strings.Contains(err.Error(), "version 1.1.0 or newer is required") {
		t.Fatalf("expected minimum version failure, got %v", err)
	}
}

func TestManagerReusesLegacyExecutableWhenReleaseServiceIsOffline(t *testing.T) {
	fixture := newToolFixture(t, "1.2.0")
	fixture.setOffline()
	now := time.Date(2026, time.August, 26, 12, 0, 0, 0, time.UTC)
	manager, definition, _ := testManager(t, fixture, now)
	legacy := legacyExecutable(t, "Legacy Acme Tool 1.1.0 (wrapped legacy tool 1.0.0)\n")
	definition.LegacyExecutables = []string{legacy}
	definition.LegacyVersionPrefixes = []string{"Legacy Acme Tool "}

	installation := resolveTool(t, manager, definition)
	if installation.Executable != legacy || installation.Version != "1.1.0" {
		t.Fatalf("unexpected legacy installation %#v", installation)
	}
	manager.Now = func() time.Time { return now.Add(time.Hour) }
	resolveTool(t, manager, definition)
	if releaseCalls, archiveCalls := fixture.counts(); releaseCalls != 1 || archiveCalls != 0 {
		t.Fatalf("legacy offline fallback checked too often: releases=%d archives=%d", releaseCalls, archiveCalls)
	}
}

func TestManagerReusesLegacyExecutableWhenCacheCannotBeCreated(t *testing.T) {
	fixture := newToolFixture(t, "1.2.0")
	manager, definition, stderr := testManager(t, fixture, time.Now())
	legacy := legacyExecutable(t, "Acme Tool 1.1.0\n")
	definition.LegacyExecutables = []string{legacy}
	if err := os.MkdirAll(filepath.Dir(definition.InstallRoot), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(definition.InstallRoot, []byte("not a directory"), 0600); err != nil {
		t.Fatal(err)
	}

	installation := resolveTool(t, manager, definition)
	if installation.Executable != legacy {
		t.Fatalf("cache failure did not reuse the legacy executable: %#v", installation)
	}
	if !strings.Contains(stderr.String(), "Unable to create the Acme Tool cache directory") {
		t.Fatalf("cache failure was not reported: %q", stderr.String())
	}
}

func TestManagerMigratesCurrentLegacyExecutableToManagedDistribution(t *testing.T) {
	fixture := newToolFixture(t, "1.2.0")
	manager, definition, _ := testManager(t, fixture, time.Now())
	legacy := legacyExecutable(t, "Acme Tool 1.2.0\n")
	definition.LegacyExecutables = []string{legacy}

	installation := resolveTool(t, manager, definition)
	if installation.Executable == legacy || installation.Version != "1.2.0" {
		t.Fatalf("legacy executable was not migrated: %#v", installation)
	}
	if releaseCalls, archiveCalls := fixture.counts(); releaseCalls != 1 || archiveCalls != 1 {
		t.Fatalf("legacy migration did not install the distribution: releases=%d archives=%d", releaseCalls, archiveCalls)
	}
}

func TestManagerSerializesConcurrentFirstInstall(t *testing.T) {
	fixture := newToolFixture(t, "1.2.0")
	manager, definition, _ := testManager(t, fixture, time.Now())

	results := make(chan Installation, 2)
	errors := make(chan error, 2)
	var workers sync.WaitGroup
	for range 2 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			installation, err := manager.Resolve(context.Background(), definition)
			results <- installation
			errors <- err
		}()
	}
	workers.Wait()
	close(results)
	close(errors)
	for err := range errors {
		if err != nil {
			t.Fatal(err)
		}
	}
	var expected Installation
	for installation := range results {
		if expected == (Installation{}) {
			expected = installation
		} else if installation != expected {
			t.Fatalf("concurrent installations differ: %#v != %#v", installation, expected)
		}
	}
	if releaseCalls, archiveCalls := fixture.counts(); releaseCalls != 1 || archiveCalls != 1 {
		t.Fatalf("concurrent install downloaded more than once: releases=%d archives=%d", releaseCalls, archiveCalls)
	}
}

func legacyExecutable(t *testing.T, versionOutput string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "tool")
	if err := os.WriteFile(path, []byte(versionOutput), 0755); err != nil {
		t.Fatal(err)
	}

	return path
}

func resolveTool(t *testing.T, manager *Manager, definition Definition) Installation {
	t.Helper()
	installation, err := manager.Resolve(t.Context(), definition)
	if err != nil {
		t.Fatal(err)
	}

	return installation
}

func newToolFixture(t *testing.T, version string) *releaseFixture {
	t.Helper()

	return newReleaseFixture(t, version, fmt.Sprintf("tool-v%s-linux-x64.tar.gz", version), toolArchive(t, version))
}

func toolArchive(t *testing.T, version string) []byte {
	t.Helper()

	return tarGz(t, map[string]archiveFile{
		fmt.Sprintf("tool-v%s/tool", version): {contents: fmt.Sprintf("Acme Tool %s\n", version), mode: 0755},
	})
}

func testManager(t *testing.T, fixture *releaseFixture, now time.Time) (*Manager, Definition, *bytes.Buffer) {
	t.Helper()
	stderr := &bytes.Buffer{}
	manager := NewManager()
	manager.Client = fixture.server.Client()
	manager.GitHubAPI = fixture.server.URL
	manager.AllowHTTP = true
	manager.Now = func() time.Time { return now }
	manager.Stderr = stderr
	manager.RuntimeOS = "linux"
	manager.RuntimeArch = "amd64"
	manager.VersionRunner = fileVersionRunner{}

	return manager, testDefinition(t), stderr
}

func testDefinition(t *testing.T) Definition {
	t.Helper()

	return Definition{
		Name:           "Acme Tool",
		Repository:     "acme/tool",
		ChecksumsAsset: "SHA256SUMS",
		MinimumVersion: "1.0.0",
		VersionPrefix:  "Acme Tool ",
		InstallRoot:    filepath.Join(t.TempDir(), "tool-cache"),
		Packages: []Package{
			{OS: "linux", Arch: "amd64", Asset: "tool-v{version}-linux-x64.tar.gz", ArchiveRoot: "tool-v{version}", Executable: "tool"},
		},
	}
}

type archiveFile struct {
	contents string
	mode     os.FileMode
}

func tarGz(t *testing.T, files map[string]archiveFile) []byte {
	t.Helper()
	var archive bytes.Buffer
	compressed := gzip.NewWriter(&archive)
	writer := tar.NewWriter(compressed)
	for name, file := range files {
		if err := writer.WriteHeader(&tar.Header{Name: name, Mode: int64(file.mode), Size: int64(len(file.contents)), Typeflag: tar.TypeReg}); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(writer, file.contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}

	return archive.Bytes()
}

func tarGzWithGlobalPAXHeader(t *testing.T) []byte {
	t.Helper()
	var archive bytes.Buffer
	compressed := gzip.NewWriter(&archive)
	writer := tar.NewWriter(compressed)
	if err := writer.WriteHeader(&tar.Header{
		Name:       "/GlobalHead.1234.1",
		Typeflag:   tar.TypeXGlobalHeader,
		PAXRecords: map[string]string{"comment": "release metadata"},
	}); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteHeader(&tar.Header{Name: "tool", Mode: 0600, Size: int64(len("contents")), Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(writer, "contents"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}

	return archive.Bytes()
}

func overflowingTarGz(t *testing.T) []byte {
	t.Helper()
	var archive bytes.Buffer
	compressed := gzip.NewWriter(&archive)
	writer := tar.NewWriter(compressed)
	if err := writer.WriteHeader(&tar.Header{Name: "small", Mode: 0600, Size: 1, Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(writer, "x"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteHeader(&tar.Header{Name: "huge", Mode: 0600, Size: math.MaxInt64, Typeflag: tar.TypeReg}); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}

	return archive.Bytes()
}

func zipArchive(t *testing.T, files map[string]archiveFile) []byte {
	t.Helper()
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for name, file := range files {
		header := &zip.FileHeader{Name: name, Method: zip.Deflate}
		header.SetMode(file.mode)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(entry, file.contents); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	return archive.Bytes()
}

func checksum(contents []byte) string {
	digest := sha256.Sum256(contents)

	return hex.EncodeToString(digest[:])
}

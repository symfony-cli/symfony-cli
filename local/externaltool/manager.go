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
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
)

const (
	checkInterval      = 24 * time.Hour
	maximumMetadata    = 4 << 20
	maximumArchiveSize = 512 << 20
)

type Package struct {
	OS          string
	Arch        string
	Asset       string
	ArchiveRoot string
	Executable  string
}

type Definition struct {
	Name              string
	Repository        string
	ChecksumsAsset    string
	MinimumVersion    string
	VersionPrefix     string
	InstallRoot       string
	LegacyExecutables []string
	Packages          []Package
}

type Installation struct {
	Executable string
	Version    string
	CacheDir   string
}

type VersionRunner interface {
	Output(ctx context.Context, executable string, arguments ...string) ([]byte, error)
}

type Manager struct {
	Client          *http.Client
	Now             func() time.Time
	Stderr          io.Writer
	RuntimeOS       string
	RuntimeArch     string
	GitHubAPI       string
	AllowHTTP       bool
	Token           string
	MetadataTimeout time.Duration
	ArchiveTimeout  time.Duration
	VersionRunner   VersionRunner
}

type releaseAsset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type release struct {
	TagName    string         `json:"tag_name"`
	Draft      bool           `json:"draft"`
	Prerelease bool           `json:"prerelease"`
	Assets     []releaseAsset `json:"assets"`
}

type installationState struct {
	Version          string `json:"version"`
	CheckedAt        int64  `json:"checkedAt"`
	LegacyExecutable string `json:"legacyExecutable,omitempty"`
}

type nativeVersionRunner struct{}

func NewManager() *Manager {
	return &Manager{
		Client:          defaultHTTPClient(),
		Now:             time.Now,
		Stderr:          os.Stderr,
		RuntimeOS:       runtime.GOOS,
		RuntimeArch:     runtime.GOARCH,
		GitHubAPI:       "https://api.github.com",
		Token:           os.Getenv("GITHUB_TOKEN"),
		MetadataTimeout: 30 * time.Second,
		ArchiveTimeout:  30 * time.Minute,
		VersionRunner:   nativeVersionRunner{},
	}
}

func defaultHTTPClient() *http.Client {
	transport, ok := http.DefaultTransport.(*http.Transport)
	if !ok {
		return &http.Client{}
	}
	clone := transport.Clone()
	clone.ResponseHeaderTimeout = 30 * time.Second

	return &http.Client{Transport: clone}
}

func (nativeVersionRunner) Output(ctx context.Context, executable string, arguments ...string) ([]byte, error) {
	output, err := exec.CommandContext(ctx, executable, arguments...).CombinedOutput()
	if err != nil {
		return output, fmt.Errorf("unable to execute the version command: %w", err)
	}

	return output, nil
}

func (m *Manager) Resolve(ctx context.Context, definition Definition) (Installation, error) {
	minimumVersion, err := m.validateDefinition(definition)
	if err != nil {
		return Installation{}, err
	}
	pkg, err := m.packageFor(definition)
	if err != nil {
		if legacy, ok := m.legacy(ctx, definition, minimumVersion); ok {
			return legacy, nil
		}

		return Installation{}, err
	}
	if err := os.MkdirAll(definition.InstallRoot, 0755); err != nil {
		if legacy, ok := m.legacy(ctx, definition, minimumVersion); ok {
			m.warning("Unable to create the %s cache directory: %v; using version %s.\n", definition.Name, err, legacy.Version)
			return legacy, nil
		}

		return Installation{}, fmt.Errorf("unable to create the %s cache directory: %w", definition.Name, err)
	}

	unlock, err := acquireLock(filepath.Join(definition.InstallRoot, "install.lock"))
	if err != nil {
		state := loadState(filepath.Join(definition.InstallRoot, "state.json"))
		if current, ok := m.installed(ctx, definition, pkg, state, minimumVersion); ok {
			m.warning("Unable to lock the %s installation: %v; using version %s.\n", definition.Name, err, current.Version)
			return current, nil
		}
		if legacy, ok := m.legacy(ctx, definition, minimumVersion); ok {
			m.warning("Unable to lock the %s installation: %v; using version %s.\n", definition.Name, err, legacy.Version)
			return legacy, nil
		}

		return Installation{}, fmt.Errorf("unable to lock the %s installation: %w", definition.Name, err)
	}
	defer unlock()

	now := m.Now().UTC()
	statePath := filepath.Join(definition.InstallRoot, "state.json")
	state := loadState(statePath)
	activeVersion := ""
	if state != nil && state.LegacyExecutable == "" {
		activeVersion = state.Version
	}
	m.cleanupCache(definition.InstallRoot, activeVersion, now.Add(-2*checkInterval))
	if state != nil && state.CheckedAt > now.Unix() {
		state.CheckedAt = 0
	}
	current, currentOK := m.installed(ctx, definition, pkg, state, minimumVersion)
	currentIsLegacy := currentOK && state.LegacyExecutable != ""
	if !currentOK {
		if legacy, ok := m.legacy(ctx, definition, minimumVersion); ok {
			current = legacy
			currentOK = true
			currentIsLegacy = true
			state = &installationState{Version: legacy.Version, LegacyExecutable: legacy.Executable}
		}
	}
	if currentOK && !currentIsLegacy {
		_ = os.Chtimes(filepath.Join(definition.InstallRoot, "versions", current.Version), now, now)
	}

	if currentOK && state.CheckedAt > now.Add(-checkInterval).Unix() {
		return current, nil
	}
	if currentOK {
		state.CheckedAt = now.Unix()
		if err := storeState(statePath, state); err != nil {
			m.warning("Unable to record the %s update check: %v\n", definition.Name, err)
		}
	}

	latest, err := m.latestRelease(ctx, definition.Repository)
	if err != nil {
		if currentOK {
			return current, nil
		}

		return Installation{}, fmt.Errorf("unable to resolve the latest stable %s release: %w", definition.Name, err)
	}
	latestVersion, err := normalizedVersion(latest.TagName)
	if err != nil {
		return m.fallback(definition, current, currentOK, fmt.Errorf("the latest stable release has an invalid version: %w", err))
	}
	if latestVersion.LessThan(minimumVersion) {
		err := fmt.Errorf("the latest stable release is %s, but version %s or newer is required", latestVersion, minimumVersion)

		return m.fallback(definition, current, currentOK, err)
	}
	if currentOK {
		currentVersion, _ := version.NewVersion(current.Version)
		if !latestVersion.GreaterThan(currentVersion) && (!currentIsLegacy || !latestVersion.Equal(currentVersion)) {
			return current, nil
		}
	}

	assetName := expandVersion(pkg.Asset, latestVersion.String())
	archiveAsset, ok := findAsset(latest.Assets, assetName)
	if !ok {
		return m.fallback(definition, current, currentOK, fmt.Errorf("the latest stable release has no %s asset", assetName))
	}
	checksumAsset, ok := findAsset(latest.Assets, definition.ChecksumsAsset)
	if !ok {
		return m.fallback(definition, current, currentOK, fmt.Errorf("the latest stable release has no %s asset", definition.ChecksumsAsset))
	}

	checksumContents, err := m.downloadBytes(ctx, checksumAsset.URL, maximumMetadata)
	if err != nil {
		return m.fallback(definition, current, currentOK, fmt.Errorf("unable to download the checksum manifest: %w", err))
	}
	expectedChecksum, err := checksumFor(checksumContents, assetName)
	if err != nil {
		return m.fallback(definition, current, currentOK, fmt.Errorf("unable to verify the release metadata: %w", err))
	}

	if m.Stderr != nil {
		fmt.Fprintf(m.Stderr, "Downloading %s %s\n", definition.Name, latestVersion)
	}
	archivePath, err := m.downloadArchive(ctx, definition.InstallRoot, archiveAsset.URL, expectedChecksum)
	if err != nil {
		return m.fallback(definition, current, currentOK, fmt.Errorf("unable to download version %s: %w", latestVersion, err))
	}
	defer os.Remove(archivePath)

	installation, err := m.install(ctx, definition, pkg, latestVersion.String(), archivePath)
	if err != nil {
		return m.fallback(definition, current, currentOK, err)
	}
	state = &installationState{Version: installation.Version, CheckedAt: now.Unix()}
	if err := storeState(statePath, state); err != nil {
		return m.fallback(definition, current, currentOK, fmt.Errorf("unable to activate version %s: %w", installation.Version, err))
	}

	return installation, nil
}

func (m *Manager) validateDefinition(definition Definition) (*version.Version, error) {
	if definition.Name == "" || definition.Repository == "" || definition.ChecksumsAsset == "" || definition.MinimumVersion == "" || definition.VersionPrefix == "" || definition.InstallRoot == "" {
		return nil, errors.New("external tool definition is incomplete")
	}
	if strings.Count(definition.Repository, "/") != 1 || strings.Contains(definition.Repository, "..") {
		return nil, errors.New("external tool repository is invalid")
	}
	minimumVersion, err := normalizedVersion(definition.MinimumVersion)
	if err != nil {
		return nil, fmt.Errorf("external tool minimum version is invalid: %w", err)
	}

	return minimumVersion, nil
}

func (m *Manager) packageFor(definition Definition) (Package, error) {
	for _, pkg := range definition.Packages {
		if pkg.OS != m.RuntimeOS || pkg.Arch != m.RuntimeArch {
			continue
		}
		if pkg.Asset == "" || pkg.Executable == "" || !safeRelativePath(pkg.Executable) || (pkg.ArchiveRoot != "" && !safeRelativePath(pkg.ArchiveRoot)) {
			return Package{}, errors.New("external tool package definition is invalid")
		}
		if !strings.HasSuffix(pkg.Asset, ".tar.gz") && !strings.HasSuffix(pkg.Asset, ".zip") {
			return Package{}, errors.New("external tool package archive is unsupported")
		}

		return pkg, nil
	}

	return Package{}, fmt.Errorf("%s is unavailable on %s/%s", definition.Name, m.RuntimeOS, m.RuntimeArch)
}

func (m *Manager) installed(ctx context.Context, definition Definition, pkg Package, state *installationState, minimumVersion *version.Version) (Installation, bool) {
	if state == nil {
		return Installation{}, false
	}
	installedVersion, err := normalizedVersion(state.Version)
	if err != nil || installedVersion.LessThan(minimumVersion) {
		return Installation{}, false
	}
	executable := filepath.Join(definition.InstallRoot, "versions", installedVersion.String(), pkg.Executable)
	if state.LegacyExecutable != "" {
		if !contains(definition.LegacyExecutables, state.LegacyExecutable) {
			return Installation{}, false
		}
		executable = state.LegacyExecutable
	}
	if err := m.verifyVersion(ctx, executable, definition.VersionPrefix, installedVersion); err != nil {
		return Installation{}, false
	}

	return Installation{Executable: executable, Version: installedVersion.String(), CacheDir: definition.InstallRoot}, true
}

func (m *Manager) legacy(ctx context.Context, definition Definition, minimumVersion *version.Version) (Installation, bool) {
	for _, executable := range definition.LegacyExecutables {
		output, err := m.VersionRunner.Output(ctx, executable, "--version")
		if err != nil {
			continue
		}
		legacyVersion, err := parseVersionOutput(output, definition.VersionPrefix)
		if err != nil || legacyVersion.LessThan(minimumVersion) {
			continue
		}

		return Installation{Executable: executable, Version: legacyVersion.String(), CacheDir: definition.InstallRoot}, true
	}

	return Installation{}, false
}

func (m *Manager) latestRelease(ctx context.Context, repository string) (*release, error) {
	endpoint := strings.TrimRight(m.GitHubAPI, "/") + "/repos/" + repository + "/releases/latest"
	contents, err := m.downloadBytes(ctx, endpoint, maximumMetadata)
	if err != nil {
		return nil, err
	}
	var latest release
	if err := json.Unmarshal(contents, &latest); err != nil {
		return nil, fmt.Errorf("invalid release metadata: %w", err)
	}
	if latest.Draft || latest.Prerelease || latest.TagName == "" {
		return nil, errors.New("release metadata does not describe a stable release")
	}

	return &latest, nil
}

func (m *Manager) downloadBytes(ctx context.Context, address string, maximumSize int64) ([]byte, error) {
	if !m.validDownloadURL(address) {
		return nil, errors.New("download URL is invalid")
	}
	requestContext, cancel := timeoutContext(ctx, m.MetadataTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestContext, http.MethodGet, address, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to create download request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "Symfony CLI")
	if m.Token != "" {
		req.Header.Set("Authorization", "Bearer "+m.Token)
	}
	resp, err := m.Client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if !m.validDownloadURL(resp.Request.URL.String()) {
		return nil, errors.New("download redirected to an invalid URL")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("download returned HTTP status %d", resp.StatusCode)
	}
	contents, err := io.ReadAll(io.LimitReader(resp.Body, maximumSize+1))
	if err != nil {
		return nil, fmt.Errorf("unable to read download: %w", err)
	}
	if int64(len(contents)) > maximumSize {
		return nil, errors.New("download exceeds the allowed size")
	}

	return contents, nil
}

func (m *Manager) downloadArchive(ctx context.Context, directory, address, expectedChecksum string) (string, error) {
	if !m.validDownloadURL(address) {
		return "", errors.New("download URL is invalid")
	}
	requestContext, cancel := timeoutContext(ctx, m.ArchiveTimeout)
	defer cancel()
	req, err := http.NewRequestWithContext(requestContext, http.MethodGet, address, nil)
	if err != nil {
		return "", fmt.Errorf("unable to create download request: %w", err)
	}
	req.Header.Set("User-Agent", "Symfony CLI")
	if m.Token != "" {
		req.Header.Set("Authorization", "Bearer "+m.Token)
	}
	resp, err := m.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()
	if !m.validDownloadURL(resp.Request.URL.String()) {
		return "", errors.New("download redirected to an invalid URL")
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return "", fmt.Errorf("download returned HTTP status %d", resp.StatusCode)
	}

	file, err := os.CreateTemp(directory, ".download-*")
	if err != nil {
		return "", fmt.Errorf("unable to create temporary download: %w", err)
	}
	path := file.Name()
	remove := true
	defer func() {
		file.Close()
		if remove {
			os.Remove(path)
		}
	}()

	hasher := sha256.New()
	written, err := io.Copy(io.MultiWriter(file, hasher), io.LimitReader(resp.Body, maximumArchiveSize+1))
	if err != nil {
		return "", fmt.Errorf("unable to save download: %w", err)
	}
	if written > maximumArchiveSize {
		return "", errors.New("download exceeds the allowed size")
	}
	if err := file.Sync(); err != nil {
		return "", fmt.Errorf("unable to sync download: %w", err)
	}
	if err := file.Close(); err != nil {
		return "", fmt.Errorf("unable to close download: %w", err)
	}
	actualChecksum := hex.EncodeToString(hasher.Sum(nil))
	if actualChecksum != expectedChecksum {
		return "", fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}
	remove = false

	return path, nil
}

func (m *Manager) validDownloadURL(address string) bool {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Host == "" {
		return false
	}

	return parsed.Scheme == "https" || m.AllowHTTP && parsed.Scheme == "http"
}

func (m *Manager) fallback(definition Definition, current Installation, currentOK bool, cause error) (Installation, error) {
	if !currentOK {
		return Installation{}, fmt.Errorf("unable to install %s: %w", definition.Name, cause)
	}
	m.warning("Unable to update %s: %v; using version %s.\n", definition.Name, cause, current.Version)

	return current, nil
}

func (m *Manager) warning(format string, arguments ...any) {
	if m.Stderr != nil {
		fmt.Fprintf(m.Stderr, format, arguments...)
	}
}

func (m *Manager) cleanupCache(root, activeVersion string, cutoff time.Time) {
	removeOldEntries(root, []string{".download-", ".state-"}, cutoff)
	versionsDirectory := filepath.Join(root, "versions")
	entries, err := os.ReadDir(versionsDirectory)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.Name() == activeVersion {
			continue
		}
		temporary := strings.HasPrefix(entry.Name(), ".install-")
		if !temporary {
			if _, err := normalizedVersion(entry.Name()); err != nil || !entry.IsDir() {
				continue
			}
		}
		info, err := entry.Info()
		if err == nil && info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(filepath.Join(versionsDirectory, entry.Name()))
		}
	}
}

func removeOldEntries(directory string, prefixes []string, cutoff time.Time) {
	entries, err := os.ReadDir(directory)
	if err != nil {
		return
	}
	for _, entry := range entries {
		matched := false
		for _, prefix := range prefixes {
			if strings.HasPrefix(entry.Name(), prefix) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		info, err := entry.Info()
		if err == nil && info.ModTime().Before(cutoff) {
			_ = os.RemoveAll(filepath.Join(directory, entry.Name()))
		}
	}
}

func timeoutContext(ctx context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if timeout <= 0 {
		return context.WithCancel(ctx)
	}

	return context.WithTimeout(ctx, timeout)
}

func (m *Manager) install(ctx context.Context, definition Definition, pkg Package, releaseVersion, archivePath string) (Installation, error) {
	versionsDirectory := filepath.Join(definition.InstallRoot, "versions")
	if err := os.MkdirAll(versionsDirectory, 0755); err != nil {
		return Installation{}, fmt.Errorf("unable to create the %s versions directory: %w", definition.Name, err)
	}
	temporary, err := os.MkdirTemp(versionsDirectory, ".install-*")
	if err != nil {
		return Installation{}, fmt.Errorf("unable to prepare the %s installation: %w", definition.Name, err)
	}
	defer os.RemoveAll(temporary)

	if err := extractArchive(archivePath, pkg.Asset, temporary); err != nil {
		return Installation{}, fmt.Errorf("unable to extract %s %s: %w", definition.Name, releaseVersion, err)
	}
	distribution := temporary
	if pkg.ArchiveRoot != "" {
		root := expandVersion(pkg.ArchiveRoot, releaseVersion)
		entries, err := os.ReadDir(temporary)
		if err != nil || len(entries) != 1 || entries[0].Name() != root || !entries[0].IsDir() {
			return Installation{}, fmt.Errorf("the %s archive has an unexpected layout", definition.Name)
		}
		distribution = filepath.Join(temporary, root)
	}
	executable := filepath.Join(distribution, pkg.Executable)
	info, err := os.Stat(executable)
	if err != nil || !info.Mode().IsRegular() {
		return Installation{}, fmt.Errorf("the %s archive does not contain %s", definition.Name, pkg.Executable)
	}
	if m.RuntimeOS != "windows" && info.Mode().Perm()&0111 == 0 {
		return Installation{}, fmt.Errorf("the %s archive contains a non-executable %s", definition.Name, pkg.Executable)
	}
	expectedVersion, _ := normalizedVersion(releaseVersion)
	if err := m.verifyVersion(ctx, executable, definition.VersionPrefix, expectedVersion); err != nil {
		return Installation{}, fmt.Errorf("the %s archive failed version verification: %w", definition.Name, err)
	}

	finalDirectory := filepath.Join(versionsDirectory, releaseVersion)
	if _, err := os.Stat(finalDirectory); err == nil {
		finalExecutable := filepath.Join(finalDirectory, pkg.Executable)
		if err := m.verifyVersion(ctx, finalExecutable, definition.VersionPrefix, expectedVersion); err == nil {
			return Installation{Executable: finalExecutable, Version: releaseVersion, CacheDir: definition.InstallRoot}, nil
		}
		if err := os.RemoveAll(finalDirectory); err != nil {
			return Installation{}, fmt.Errorf("unable to replace the invalid %s installation: %w", definition.Name, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Installation{}, fmt.Errorf("unable to inspect the %s installation: %w", definition.Name, err)
	}
	if err := os.Rename(distribution, finalDirectory); err != nil {
		return Installation{}, fmt.Errorf("unable to activate %s %s: %w", definition.Name, releaseVersion, err)
	}

	return Installation{
		Executable: filepath.Join(finalDirectory, pkg.Executable),
		Version:    releaseVersion,
		CacheDir:   definition.InstallRoot,
	}, nil
}

func (m *Manager) verifyVersion(ctx context.Context, executable, prefix string, expected *version.Version) error {
	output, err := m.VersionRunner.Output(ctx, executable, "--version")
	if err != nil {
		return err
	}
	actual, err := parseVersionOutput(output, prefix)
	if err != nil {
		return err
	}
	if !actual.Equal(expected) {
		return fmt.Errorf("reported version %s instead of %s", actual, expected)
	}

	return nil
}

func parseVersionOutput(output []byte, prefix string) (*version.Version, error) {
	value := strings.TrimSpace(string(output))
	if !strings.HasPrefix(value, prefix) {
		return nil, errors.New("version output is invalid")
	}
	value = strings.TrimSpace(strings.TrimPrefix(value, prefix))
	if value == "" || strings.ContainsAny(value, "\r\n\t ") {
		return nil, errors.New("version output is invalid")
	}

	return normalizedVersion(value)
}

func normalizedVersion(value string) (*version.Version, error) {
	value = strings.TrimPrefix(strings.TrimSpace(value), "v")
	parsed, err := version.NewVersion(value)
	if err != nil {
		return nil, fmt.Errorf("unable to parse version: %w", err)
	}
	if parsed.String() != value {
		return nil, fmt.Errorf("version %q is not normalized", value)
	}

	return parsed, nil
}

func contains(values []string, searched string) bool {
	for _, value := range values {
		if value == searched {
			return true
		}
	}

	return false
}

func findAsset(assets []releaseAsset, name string) (releaseAsset, bool) {
	for _, asset := range assets {
		if asset.Name == name && asset.URL != "" {
			return asset, true
		}
	}

	return releaseAsset{}, false
}

func checksumFor(contents []byte, filename string) (string, error) {
	checksum := ""
	for _, line := range strings.Split(strings.ReplaceAll(string(contents), "\r\n", "\n"), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 || fields[1] != filename {
			continue
		}
		if len(fields[0]) != sha256.Size*2 {
			return "", fmt.Errorf("invalid checksum for %s", filename)
		}
		if _, err := hex.DecodeString(fields[0]); err != nil {
			return "", fmt.Errorf("invalid checksum for %s", filename)
		}
		if checksum != "" {
			return "", fmt.Errorf("duplicate checksum for %s", filename)
		}
		checksum = strings.ToLower(fields[0])
	}
	if checksum == "" {
		return "", fmt.Errorf("missing checksum for %s", filename)
	}

	return checksum, nil
}

func loadState(path string) *installationState {
	contents, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var state installationState
	if err := json.Unmarshal(contents, &state); err != nil || state.Version == "" || state.CheckedAt < 0 {
		return nil
	}

	return &state
}

func storeState(path string, state *installationState) error {
	contents, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("unable to encode state: %w", err)
	}
	file, err := os.CreateTemp(filepath.Dir(path), ".state-*")
	if err != nil {
		return fmt.Errorf("unable to create temporary state: %w", err)
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0644); err != nil {
		file.Close()
		return fmt.Errorf("unable to set state permissions: %w", err)
	}
	if _, err := file.Write(contents); err != nil {
		file.Close()
		return fmt.Errorf("unable to write state: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("unable to sync state: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("unable to close state: %w", err)
	}

	return replaceFile(temporary, path)
}

func expandVersion(template, releaseVersion string) string {
	return strings.ReplaceAll(template, "{version}", releaseVersion)
}

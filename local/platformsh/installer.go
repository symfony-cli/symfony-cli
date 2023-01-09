/*
 * Copyright (c) 2023-present Fabien Potencier <fabien@symfony.com>
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

package platformsh

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/blackfireio/osinfo"
	"github.com/pkg/errors"
	"github.com/schollz/progressbar/v3"
	"github.com/symfony-cli/terminal"
)

type githubAsset struct {
	Name    string
	URL     string `json:"browser_download_url"`
	version string
}

type versionCheck struct {
	CurrentVersion string
	Timestamp      int64
}

// BinaryPath returns the cloud binary path.
func BinaryPath(home string) string {
	return filepath.Join(home, ".platformsh", "bin", "platform")
}

// Install installs or updates the Platform.sh CLI tool.
func Install(home string) (string, error) {
	binPath := BinaryPath(home)
	versionCheckPath := binPath + ".json"

	// do we already have the binary?
	binExists := false
	if _, err := os.Stat(binPath); err == nil {
		binExists = true
		versionCheck := loadVersionCheck(versionCheckPath)
		if versionCheck == nil {
			// we need to download the bin again as we don't have the version info anymore, so it will never be updated!
			goto download
		}
		// have we checked recently for a new version?
		if versionCheck.Timestamp > time.Now().Add(-24*time.Hour).Unix() {
			return binPath, nil
		}
		// don't check for the next 24 hours
		versionCheck.store(versionCheckPath)
		if asset, err := getLatestVersion(); err == nil {
			// no new version
			if asset.version == string(versionCheck.CurrentVersion) {
				return binPath, nil
			}
		}
	}

download:
	asset, err := getLatestVersion()
	if err != nil {
		if binExists {
			// unable to get the latest version, but we already have a bin, use it
			return binPath, nil
		}
		return "", err
	}
	if err := downloadAndExtract(asset, binPath); err != nil {
		return "", err
	}

	versionCheck := versionCheck{CurrentVersion: asset.version}
	if err := versionCheck.store(versionCheckPath); err != nil {
		return "", err
	}
	return binPath, nil
}

func getLatestVersion() (*githubAsset, error) {
	spinner := terminal.NewSpinner(terminal.Stderr)
	spinner.Start()
	defer spinner.Stop()

	resp, err := http.Get("https://api.github.com/repos/platformsh/cli/releases/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return nil, errors.New(http.StatusText(resp.StatusCode))
	}
	manifestBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var manifest struct {
		Name   string
		Assets []*githubAsset
	}
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		return nil, err
	}

	info, err := osinfo.GetOSInfo()
	if err != nil {
		return nil, err
	}

	var asset *githubAsset
	for _, a := range manifest.Assets {
		if !strings.HasSuffix(a.Name, ".gz") && !strings.HasSuffix(a.Name, ".zip") {
			continue
		}
		if !strings.Contains(a.Name, "platform") {
			continue
		}
		if (strings.Contains(a.Name, info.Architecture) && strings.Contains(a.Name, info.Family)) ||
			(strings.Contains(a.Name, "all") && info.Family == "darwin") {
			asset = a
			break
		}
	}
	if asset == nil {
		return nil, errors.New(fmt.Sprintf("unable to find a suitable Platform.sh CLI tool for your machine (%s/%s)", info.Family, info.Architecture))
	}
	asset.version = manifest.Name

	return asset, nil
}

func downloadAndExtract(asset *githubAsset, binPath string) error {
	resp, err := http.Get(asset.URL)
	if err != nil {
		return err
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		return errors.New(http.StatusText(resp.StatusCode))
	}

	pr, pw := io.Pipe()
	errs := make(chan error, 1)
	go func() {
		bar := progressbar.DefaultBytes(resp.ContentLength, fmt.Sprintf("Downloading Platform.sh CLI version %s", asset.version))
		if _, err := io.Copy(io.MultiWriter(pw, bar), resp.Body); err != nil {
			errs <- err
		}
		_ = bar.Close()
		errs <- pw.Close()
	}()

	gzr, err := gzip.NewReader(pr)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		select {
		case err := <-errs:
			return err
		default:
			header, err := tr.Next()
			switch {
			case err == io.EOF:
				return nil
			case err != nil:
				return err
			case header == nil:
				continue
			default:
				if header.Typeflag != tar.TypeReg {
					continue
				}
				if header.Name != "platform" {
					continue
				}
				if _, err := os.Stat(filepath.Dir(binPath)); os.IsNotExist(err) {
					if err := os.MkdirAll(filepath.Dir(binPath), 0755); err != nil {
						return err
					}
				}
				out, err := os.OpenFile(binPath, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
				if err != nil {
					return err
				}
				if _, err := io.Copy(out, tr); err != nil {
					out.Close()
					return err
				}
				return out.Close()
			}
		}
	}
}

func loadVersionCheck(path string) *versionCheck {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var versionCheck versionCheck
	if err := json.Unmarshal(data, &versionCheck); err != nil {
		_ = os.Remove(path)
		return nil
	}
	return &versionCheck
}

func (versionCheck *versionCheck) store(path string) error {
	versionCheck.Timestamp = time.Now().Unix()
	data, err := json.Marshal(versionCheck)
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

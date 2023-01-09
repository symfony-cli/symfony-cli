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
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/blackfireio/osinfo"
	"github.com/pkg/errors"
	"github.com/schollz/progressbar/v3"
	"github.com/symfony-cli/terminal"
)

type githubAsset struct {
	Name string
	URL  string `json:"browser_download_url"`
}

func InstallBin(home string) error {
	dir := filepath.Join(home, ".platformsh", "bin")
	if _, err := os.Stat(filepath.Join(dir, "platform")); err == nil {
		return nil
	}

	asset, err := getLatestVersionURL()
	if err != nil {
		return err
	}

	return downloadAndExtract(asset, home)
}

func getLatestVersionURL() (*githubAsset, error) {
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
	manifestBody, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var manifest struct {
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
		if strings.Contains(a.Name, info.Architecture) && strings.Contains(a.Name, info.Family) {
			asset = a
			break
		}
	}
	if asset == nil {
		return nil, errors.New(fmt.Sprintf("unable to find a suitable Platform.sh CLI tool for your machine (%s/%s)", info.Family, info.Architecture))
	}
	return asset, nil
}

func downloadAndExtract(asset *githubAsset, home string) error {
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
		bar := progressbar.DefaultBytes(resp.ContentLength, "Downloading Platform.sh CLI tool")
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
				path := filepath.Join(home, ".platformsh", "bin", "platform")
				if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
					_ = os.MkdirAll(filepath.Dir(path), 0755)
				}
				out, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
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

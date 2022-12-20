/*
 * Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
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

package updater

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"
	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
	"github.com/rs/zerolog"
	"github.com/symfony-cli/console"
	"github.com/symfony-cli/symfony-cli/util"
)

func NewUpdater(cacheDir string, output io.Writer, debug bool) Updater {
	roundTripper := &http.Transport{
		Proxy:        http.ProxyFromEnvironment,
		MaxIdleConns: 100,
	}

	logger := zerolog.Nop()
	if debug {
		logger = zerolog.New(zerolog.ConsoleWriter{Out: output})
	}

	return Updater{
		CacheDir:   cacheDir,
		HTTPClient: &http.Client{Transport: CacheTransport(roundTripper, diskcache.New(cacheDir))},
		Output:     output,
		Timeout:    time.Second,
		logger:     logger,
	}
}

type Updater struct {
	CacheDir   string
	HTTPClient *http.Client
	Output     io.Writer
	Timeout    time.Duration

	logger zerolog.Logger
	timer  *time.Timer
}

// CheckForNewVersion does a simple check once (within the Updater.Timeout
// timeframe) for new version available and display a warning if
// a new version is found.
func (updater *Updater) CheckForNewVersion(currentVersionStr string) {
	if util.IsGoRun() {
		return
	}

	currentVersion, err := version.NewVersion(currentVersionStr)
	if err != nil {
		return
	}

	newVersionCh := make(chan *version.Version)
	go func() {
		version := updater.check(currentVersion, true)
		select {
		case newVersionCh <- version:
		default:
		}
	}()

	updater.timer = time.NewTimer(updater.Timeout)
	defer updater.timer.Stop()
	select {
	case <-updater.timer.C:
		updater.logger.Printf("<comment>Checking for updates timeout expired</>")
	case newVersionFound := <-newVersionCh:
		if newVersionFound == nil {
			updater.silence()
			return
		}
		fmt.Fprintf(updater.Output, "\n<error> INFO </> <info>A new Symfony CLI version is available</> (<info>%s</>, currently running <info>%s</>).\n\n", newVersionFound, currentVersion)
		fmt.Fprintf(updater.Output, "       If you installed the Symfony CLI via a package manager, updates are going to be automatic.\n")
		fmt.Fprintf(updater.Output, "       If not, upgrade by downloading the new version at <href=https://github.com/symfony-cli/symfony-cli/releases>https://github.com/symfony-cli/symfony-cli/releases</>\n")
		fmt.Fprintf(updater.Output, "       And replace the current binary (<info>%s</>) by the new one.\n\n", console.CurrentBinaryName())
	}
}

func (updater *Updater) silence() {
	filename := filepath.Join(updater.CacheDir, "silence")
	if f, _ := os.OpenFile(filename, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0700); f != nil {
		f.Close()
	}
}

func (updater *Updater) check(currentVersion *version.Version, enableCache bool) *version.Version {
	if err := updater.createCacheDir(); err != nil {
		fmt.Fprintf(updater.Output, "<comment>Disabling update check: %q</>\n", err)
		return nil
	}
	if silenceInfo, _ := os.Stat(filepath.Join(updater.CacheDir, "silence")); enableCache && silenceInfo != nil {
		if time.Since(silenceInfo.ModTime()) < 24*time.Hour {
			return nil
		}
	}

	increaseTimeOut := false
	var manifestBody []byte
	manifestCachePath := filepath.Join(updater.CacheDir, "manifest.json")
	manifestFile, manifestFileErr := os.Open(manifestCachePath)
	if enableCache && manifestFileErr == nil {
		if stat, err := manifestFile.Stat(); err == nil {
			if time.Since(stat.ModTime()) < 1*time.Hour {
				if manifestCacheBody, manifestCacheErr := io.ReadAll(manifestFile); manifestCacheErr == nil {
					manifestBody = manifestCacheBody
				}
			} else {
				increaseTimeOut = time.Since(stat.ModTime()) > 7*24*time.Hour
			}
		}
		manifestFile.Close()
	} else {
		increaseTimeOut = os.IsNotExist(manifestFileErr)
	}

	if increaseTimeOut && updater.timer != nil {
		updater.timer.Stop()
		updater.Timeout = 4 * updater.Timeout
		updater.logger.Printf("We didn't manage to check version for a long time, increasing timeout to %s", updater.Timeout)
		updater.timer.Reset(updater.Timeout)
	}

	updater.logger.Printf("Checking for updates (current version: <info>%s</>)", currentVersion)
	if manifestBody == nil {
		req, err := http.NewRequest(http.MethodGet, "https://api.github.com/repos/symfony-cli/symfony-cli/releases/latest", nil)
		if err != nil {
			updater.logger.Err(err).Msg("")
			return nil
		}

		resp, err := updater.HTTPClient.Do(req)
		if resp != nil {
			defer resp.Body.Close()
		}
		if err == nil && (resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest) {
			err = errors.New(http.StatusText(resp.StatusCode))
		}
		if err != nil {
			updater.logger.Err(err).Msg("")
			return nil
		}

		manifestBody, err = io.ReadAll(resp.Body)
		if err != nil {
			updater.logger.Err(err).Msg("")
			return nil
		}

		if err := os.WriteFile(manifestCachePath, manifestBody, 0644); err != nil {
			updater.logger.Err(err).Msg("")
			return nil
		}
	}

	var manifest struct {
		Name string
	}
	if err := json.Unmarshal(manifestBody, &manifest); err != nil {
		updater.logger.Err(err).Msg("")
		return nil
	}

	latestVersion, err := version.NewVersion(manifest.Name)
	if err != nil {
		updater.logger.Err(err).Msg("")
		return nil
	}
	if latestVersion.GreaterThan(currentVersion) {
		return latestVersion
	}
	return nil
}

func (updater *Updater) createCacheDir() error {
	if directoryInfo, directoryErr := os.Stat(updater.CacheDir); os.IsNotExist(directoryErr) {
		return errors.WithStack(os.MkdirAll(updater.CacheDir, 0750))
	} else if directoryErr != nil {
		return errors.WithStack(directoryErr)
	} else if !directoryInfo.IsDir() {
		return errors.Errorf("%q already exists and is not a directory", updater.CacheDir)
	}

	return nil
}

func CacheTransport(tripper http.RoundTripper, cache httpcache.Cache) http.RoundTripper {
	return &httpcache.Transport{
		Transport: &cacheInnerTransport{tripper},
		Cache:     cache,
	}
}

// cacheInnerTransport is a http.RoundTripper that cleanup cache
// headers from HTTP responses with a status code outside the 200-399 range
// (used to prevent caching error responses).
type cacheInnerTransport struct {
	http.RoundTripper
}

func (rt *cacheInnerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := rt.RoundTripper.RoundTrip(req)
	if resp == nil {
		return resp, err
	}

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusBadRequest {
		resp.Header.Del("date")
		resp.Header.Del("expires")
		resp.Header.Del("etag")
		resp.Header.Del("last-modified")
		resp.Header.Set("cache-control", "no-cache")
	}

	return resp, err
}

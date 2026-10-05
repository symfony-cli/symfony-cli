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

package upsun

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/hashicorp/go-version"
	"github.com/pkg/errors"
)

const metaRegistryURL = "https://meta.upsun.com"

type MetaRegistry struct {
	BaseURL   string
	Client    *http.Client
	UserAgent string

	images     map[string]metaImage
	extensions map[string]metaExtension
}

type metaImage struct {
	Service  bool `json:"service"`
	Versions map[string]struct {
		Upsun struct {
			Status string `json:"status"`
		} `json:"upsun"`
	} `json:"versions"`
}

type metaExtension struct {
	Versions map[string]struct {
		Status string `json:"status"`
	} `json:"versions"`
}

func NewMetaRegistry(appVersion string) *MetaRegistry {
	return &MetaRegistry{
		BaseURL:   metaRegistryURL,
		Client:    &http.Client{Timeout: 30 * time.Second},
		UserAgent: "symfony-cli/" + appVersion,
	}
}

// ServiceVersion returns the Upsun version of a service type to use for the
// wanted one (like a Docker image version), and whether it matches it:
//
//  1. the newest supported or deprecated version matching the wanted one, like
//     16 for 16.4 or 8.8 for 8;
//  2. otherwise, the oldest supported version newer than the wanted one, so
//     that a retired version is upgraded as little as possible;
//  3. otherwise (or when wanted is empty), the newest supported version.
//
// Deprecated versions are only used in steps 2 and 3 when no version is
// supported. It returns an empty string for unknown service types.
func (r *MetaRegistry) ServiceVersion(serviceType, wanted string) (string, bool, error) {
	supported, deprecated, err := r.serviceVersions(serviceType)
	if err != nil {
		return "", false, err
	}

	if wanted != "" {
		var match *version.Version
		for _, v := range slices.Concat(supported, deprecated) {
			if versionMatches(v.Original(), wanted) && (match == nil || v.GreaterThan(match)) {
				match = v
			}
		}
		if match != nil {
			return match.Original(), true, nil
		}
	}

	candidates := supported
	if len(candidates) == 0 {
		candidates = deprecated
	}
	if len(candidates) == 0 {
		return "", false, nil
	}
	if w, err := version.NewVersion(wanted); err == nil {
		for _, v := range candidates {
			if v.GreaterThan(w) {
				return v.Original(), false, nil
			}
		}
	}
	return candidates[len(candidates)-1].Original(), false, nil
}

// serviceVersions returns the supported and deprecated versions of a service
// type, sorted from the oldest to the newest.
func (r *MetaRegistry) serviceVersions(serviceType string) ([]*version.Version, []*version.Version, error) {
	if r.images == nil {
		if err := r.fetch("/images", &r.images); err != nil {
			return nil, nil, err
		}
	}
	image, ok := r.images[serviceType]
	if !ok || !image.Service {
		return nil, nil, nil
	}
	var supported, deprecated []*version.Version
	for raw, info := range image.Versions {
		status := info.Upsun.Status
		if status != "supported" && status != "deprecated" {
			continue
		}
		v, err := version.NewVersion(raw)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "unable to parse version %q of service %q", raw, serviceType)
		}
		if status == "supported" {
			supported = append(supported, v)
		} else {
			deprecated = append(deprecated, v)
		}
	}
	sort.Sort(version.Collection(supported))
	sort.Sort(version.Collection(deprecated))
	return supported, deprecated, nil
}

// versionMatches reports whether two versions are equal on the segments they
// both define, like 16 and 16.4.
func versionMatches(a, b string) bool {
	as, bs := strings.Split(a, "."), strings.Split(b, ".")
	for i := range min(len(as), len(bs)) {
		if as[i] != bs[i] {
			return false
		}
	}
	return true
}

func (r *MetaRegistry) IsPHPExtensionAvailable(ext, phpVersion string) (bool, error) {
	if r.extensions == nil {
		var extensions map[string]metaExtension
		if err := r.fetch("/extensions/php/cloud", &extensions); err != nil {
			return false, err
		}
		r.extensions = make(map[string]metaExtension, len(extensions))
		for name, e := range extensions {
			r.extensions[strings.ToLower(name)] = e
		}
	}
	info, ok := r.extensions[strings.ToLower(ext)].Versions[phpVersion]
	if !ok {
		return false, nil
	}
	switch info.Status {
	case "available", "default", "deprecated":
		return true, nil
	}
	return false, nil
}

func (r *MetaRegistry) fetch(path string, v any) error {
	url := r.BaseURL + path
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return errors.Wrapf(err, "unable to fetch %s", url)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", r.UserAgent)
	resp, err := r.Client.Do(req)
	if err != nil {
		return errors.Wrapf(err, "unable to fetch %s", url)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unable to fetch %s: %s", url, resp.Status)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "unable to read %s", url)
	}
	if err := json.Unmarshal(body, v); err != nil {
		return errors.Wrapf(err, "unable to decode %s", url)
	}
	return nil
}

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

// ServiceLastVersion returns an empty string for unknown service types.
func (r *MetaRegistry) ServiceLastVersion(serviceType string) (string, error) {
	if r.images == nil {
		if err := r.fetch("/images", &r.images); err != nil {
			return "", err
		}
	}
	image, ok := r.images[serviceType]
	if !ok || !image.Service {
		return "", nil
	}
	latest := map[string]*version.Version{}
	for raw, info := range image.Versions {
		status := info.Upsun.Status
		if status != "supported" && status != "deprecated" {
			continue
		}
		v, err := version.NewVersion(raw)
		if err != nil {
			return "", errors.Wrapf(err, "unable to parse version %q of service %q", raw, serviceType)
		}
		if l := latest[status]; l == nil || v.GreaterThan(l) {
			latest[status] = v
		}
	}
	for _, status := range []string{"supported", "deprecated"} {
		if v := latest[status]; v != nil {
			return v.Original(), nil
		}
	}
	return "", nil
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

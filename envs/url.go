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

package envs

import (
	"bytes"
	"encoding/json"
	"net/url"

	"github.com/pkg/errors"
)

var end = errors.New("invalid end of array or object")

type URLSlice []URL

// UnmarshalJSON for map slice.
func (s *URLSlice) UnmarshalJSON(b []byte) error {
	var v map[string]URL
	var keys []string

	if err := errors.WithStack(json.Unmarshal(b, &v)); err != nil {
		return errors.WithStack(err)
	}

	d := json.NewDecoder(bytes.NewReader(b))
	t, err := d.Token()
	if err != nil {
		return errors.WithStack(err)
	}
	if t != json.Delim('{') {
		return errors.New("expected start of object")
	}

	for {
		t, err := d.Token()
		if err != nil {
			return errors.WithStack(err)
		}
		if t == json.Delim('}') {
			break
		}
		keys = append(keys, t.(string))
		if err := errors.WithStack(skipValue(d)); err != nil {
			return errors.WithStack(err)
		}
	}

	slice := make([]URL, 0, len(keys))

	for _, key := range keys {
		value := v[key]
		value.Key = key
		slice = append(slice, value)
	}

	*s = slice
	return nil
}

func skipValue(d *json.Decoder) error {
	t, err := d.Token()
	if err != nil {
		return errors.WithStack(err)
	}
	switch t {
	case json.Delim('['), json.Delim('{'):
		for {
			if err := skipValue(d); err != nil {
				if err == end {
					break
				}
				return errors.WithStack(err)
			}
		}
	case json.Delim(']'), json.Delim('}'):
		return end
	}
	return nil
}

type URL struct {
	Key string `json:"-"`

	Kind        string `json:"type"`
	To          string
	Upstream    string
	OriginalURL string `json:"original_url"`

	url *url.URL
}

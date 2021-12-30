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
		return err
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
			return err
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
				return err
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

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

package humanlog

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pkg/errors"
)

// [12-Aug-2020 16:31:33] WARNING: [pool web] child 312 said into stdout: "[2020-08-12T18:31:33.470956+02:00] console.DEBUG: www {"xxx":"yyy","code":1} []"
var PHPFPMLogLineRegexp = regexp.MustCompile(`^\[\d+\-[^\-]+\-\d+ \d+\:\d+\:[\d\.]+\] WARNING\: \[pool [^\]]+\] child \d+ said into std(?:err|out)\: "(.*)"\s*$`)

type Options struct {
	SkipUnchanged bool
	LightBg       bool
	WithSource    bool
}

type Handler struct {
	opts *Options

	mu       sync.Mutex
	lastLine *line
}

type line struct {
	level   string
	time    time.Time
	source  string
	message string
	fields  map[string]string
}

func NewHandler(opts *Options) *Handler {
	return &Handler{
		opts: opts,
	}
}

func (h *Handler) Simplify(in []byte) []byte {
	var line *line
	var err error
	h.mu.Lock()
	defer func() {
		h.lastLine = line
		h.mu.Unlock()
	}()

	// remove the end newline
	in = bytes.TrimRight(in, "\n")

	// is it a PHP FPM line? (strip the first (irrelevant) part)
	in = PHPFPMLogLineRegexp.ReplaceAll(in, []byte("$1"))

	line, err = convertPHPLog(in)
	if err != nil || line == nil {
		line, err = convertPHPFPMLog(in)
		if err != nil || line == nil {
			// is it a Symfony log line?
			line, err = convertSymfonyLog(in)
			if err != nil {
				return in
			}
			if line == nil {
				if !bytes.Contains(in, []byte(`"time":`)) && !bytes.Contains(in, []byte(`"ts":`)) {
					return in
				}
				line, err = unmarshal(in)
				if err != nil {
					return in
				}
			}
		}
	}

	tweakHTTPLog(line)

	var buf bytes.Buffer

	buf.WriteString(line.message)
	buf.WriteString(" ")
	buf.WriteString(strings.Join(h.joinKVs(line), " "))

	return buf.Bytes()
}

func (h *Handler) Prettify(in []byte) []byte {
	var line *line
	var err error
	h.mu.Lock()
	defer func() {
		h.lastLine = line
		h.mu.Unlock()
	}()

	// remove the end newline
	in = bytes.TrimRight(in, "\n")

	// is it a PHP FPM line? (strip the first (irrelevant) part)
	in = PHPFPMLogLineRegexp.ReplaceAll(in, []byte("$1"))

	line, err = convertPHPLog(in)
	if err != nil || line == nil {
		line, err = convertPHPFPMLog(in)
		if err != nil || line == nil {
			// is it a Symfony log line?
			line, err = convertSymfonyLog(in)
			if err != nil {
				return in
			}
			if line == nil {
				if !bytes.Contains(in, []byte(`"time":`)) && !bytes.Contains(in, []byte(`"ts":`)) {
					return in
				}
				line, err = unmarshal(in)
				if err != nil {
					return in
				}
			}
		}
	}

	tweakHTTPLog(line)

	var buf bytes.Buffer
	buf.WriteString(line.time.Format(time.Stamp))

	buf.WriteString(" |")
	lvl := strings.ToUpper(line.level)
	if len(lvl) > 7 {
		lvl = lvl[:7]
	} else {
		lvl += strings.Repeat(" ", 7-len(lvl))
	}

	switch line.level {
	case "notice", "warn", "warning":
		buf.WriteString("<warning>")
	case "error", "fatal", "panic", "critical", "emergency":
		buf.WriteString("<error>")
	}
	buf.WriteString(lvl)
	switch line.level {
	case "notice", "warn", "warning", "error", "fatal", "panic", "critical", "emergency":
		buf.WriteString("</>")
	}
	buf.WriteString("| ")

	if h.opts.WithSource {
		buf.WriteString("<comment>")
		if line.source == "" {
			line.source = "       "
		}
		source := strings.ToUpper(line.source)
		if len(source) > 6 {
			source = source[:6]
		} else if len(source) < 6 {
			source = source + strings.Repeat(" ", 6-len(source))
		}
		buf.WriteString(source)
		buf.WriteString("</> ")
	}
	buf.WriteString(line.message)
	buf.WriteString(" ")
	buf.WriteString(strings.Join(h.joinKVs(line), " "))

	return buf.Bytes()
}

func (h *Handler) joinKVs(line *line) []string {
	kv := make([]string, 0, len(line.fields))
	for k, v := range line.fields {
		if h.opts.SkipUnchanged && h.lastLine != nil {
			if lastV, ok := h.lastLine.fields[k]; ok && lastV == v {
				continue
			}
		}
		if k == "err" || k == "error" || k == "exception" {
			kv = append(kv, "<error>"+k+"</>="+v)
		} else {
			kv = append(kv, "<fg=cyan>"+k+"</>="+v)
		}
	}
	sort.Strings(kv)
	return kv
}

func unmarshal(data []byte) (*line, error) {
	raw := make(map[string]interface{})
	err := errors.WithStack(json.Unmarshal(data, &raw))
	if err != nil {
		return nil, errors.WithStack(err)
	}
	line := &line{
		fields: make(map[string]string),
	}
	timeStr, ok := raw["time"].(string)
	if ok {
		delete(raw, "time")
	} else {
		timeStr, ok = raw["ts"].(string)
		if ok {
			delete(raw, "ts")
		}
	}
	if ok {
		line.time, ok = tryParseTime(timeStr)
		if !ok {
			return nil, errors.Errorf("field time is not a known timestamp: %v", timeStr)
		}
	} else if i, iOk := raw["ts"].(int64); iOk {
		line.time = time.Unix(i, 0)
		delete(raw, "ts")
	} else if f, fOk := raw["ts"].(float64); fOk {
		line.time = time.Unix(int64(f), int64((f-float64(int64(f)))*1000000000))
		delete(raw, "ts")
	}

	if line.source, ok = raw["source"].(string); ok {
		delete(raw, "source")
	}

	if line.message, ok = raw["msg"].(string); ok {
		delete(raw, "msg")
	} else if line.message, ok = raw["message"].(string); ok {
		delete(raw, "message")
	}

	line.level, ok = raw["level"].(string)
	if !ok {
		line.level, ok = raw["lvl"].(string)
		delete(raw, "lvl")
		if !ok {
			line.level = "????"
		}
	} else {
		delete(raw, "level")
	}

	for key, val := range raw {
		line.fields[key] = convertAnyVal(val)
	}
	return line, nil
}

func convertAnyVal(val interface{}) string {
	switch v := val.(type) {
	case float64:
		if v-math.Floor(v) < 0.000001 && v < 1e9 {
			// not too large integer
			return fmt.Sprintf("%d", int(v))
		}
		return fmt.Sprintf("%g", v)
	case string:
		return fmt.Sprintf("%q", v)
	default:
		ret, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(ret)
	}
}

func tweakHTTPLog(l *line) {
	// is it an HTTP log line?
	status, ok := l.fields["status"]
	if !ok {
		return
	}
	method, ok := l.fields["method"]
	if !ok {
		return
	}
	method = method[1 : len(method)-1]

	url := l.message
	scheme, okScheme := l.fields["scheme"]
	host, okHost := l.fields["host"]
	if method == "GET" && okScheme && okHost {
		url = fmt.Sprintf("<href=%s://%s%s>%s</>", scheme[1:len(scheme)-1], host[1:len(host)-1], l.message, l.message)
		delete(l.fields, "scheme")
		delete(l.fields, "host")
	}

	delete(l.fields, "status")
	delete(l.fields, "method")
	l.message = fmt.Sprintf("%-4s (%s) <fg=cyan>%s</>", method, status, url)
}

const RFC3339_EXTENDED = "2006-01-02T15:04:05.999999999-0700"

var formats = []string{
	"2006-01-02 15:04:05.999999999 -0700 MST",
	"2006-01-2 15:04:05",
	"2006-01-2 15:04",
	"2006-01-2 15:04:05 -0700",
	"2006-01-2 15:04 -0700",
	"2006-01-2 15:04:05 -07:00",
	"2006-01-2 15:04 -07:00",
	"2006-01-2 15:04:05 MST",
	"2006-01-2 15:04 MST",
	time.RFC3339,
	RFC3339_EXTENDED,
	time.RFC3339Nano,
	time.RFC822,
	time.RFC822Z,
	time.RFC850,
	time.RFC1123,
	time.RFC1123Z,
	time.UnixDate,
	time.RubyDate,
	time.ANSIC,
	time.Kitchen,
	time.Stamp,
	time.StampMilli,
	time.StampMicro,
	time.StampNano,
}

// tries to parse time using a couple of formats before giving up
func tryParseTime(value string) (time.Time, bool) {
	var t time.Time
	var err error
	for _, layout := range formats {
		t, err = time.Parse(layout, value)
		if err == nil {
			return t, true
		}
	}

	return t, false
}

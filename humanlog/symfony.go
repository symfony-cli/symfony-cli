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
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// [2018-11-19 12:52:00] console.DEBUG: www {"xxx":"yyy","code":1} []
// or [2019-11-13T07:16:50.260544+01:00] console.DEBUG: www {"xxx":"yyy","code":1} []
var symfonyLogLineRegexp = regexp.MustCompile("^\\[(\\d{4}\\-\\d{2}\\-\\d{2} \\d{2}\\:\\d{2}\\:\\d{2}|\\d{4}\\-\\d{2}\\-\\d{2}T\\d{2}\\:\\d{2}\\:\\d{2}\\.\\d+\\+\\d{2}\\:\\d{2})\\] ([^\\.]+)\\.([^\\:]+)\\: (.+) (\\[.*?\\]|{.*?}) (\\[.*?\\]|{.*?})\\s*$")

func convertSymfonyLog(in []byte) (*line, error) {
	allMatches := symfonyLogLineRegexp.FindAllSubmatch(in, -1)
	if allMatches == nil {
		return nil, nil
	}
	line := &line{
		fields: make(map[string]string),
	}
	var err error
	matches := allMatches[0]
	for i, m := range matches {
		if i == 1 {
			// convert date (2018-11-19 13:32:00)
			line.time, err = time.Parse(`2006-01-02 15:04:05`, string(m))
			if err != nil {
				// convert date (2019-11-13T07:16:50.260544+01:00)
				line.time, err = time.Parse(time.RFC3339Nano, string(m))
				if err != nil {
					return nil, errors.WithStack(err)
				}
			}
		} else if i == 2 {
			line.source = string(m)
		} else if i == 3 {
			line.level = strings.ToLower(string(m))
		} else if i == 4 {
			line.message = string(m)
			// unfortunately, we cannot really parse log lines with a regexp
			// one problem is for very nested hash maps like exceptions
			// this hack works because we don't want exception traces in logs anyway
			if idx := strings.Index(line.message, ` {"exception":`); idx != -1 {
				line.message = line.message[:idx]
			}
		} else if i == 5 || i == 6 {
			args := make(map[string]interface{})
			if m[0] == '[' {
				var values []interface{}
				if err := errors.WithStack(json.Unmarshal(m, &values)); err != nil {
					continue
				}
				for i, v := range values {
					args[strconv.Itoa(i)] = v
				}
			} else {
				if err := errors.WithStack(json.Unmarshal(m, &args)); err != nil {
					continue
				}
			}
			for k, v := range args {
				if k == "exception" {
					continue
				}
				line.fields[k] = convertAnyVal(v)
			}
		}
	}
	return line, nil
}

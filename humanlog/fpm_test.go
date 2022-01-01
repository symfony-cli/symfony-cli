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
	"time"

	. "gopkg.in/check.v1"
)

func (s *HumanlogSuite) TestFPMLogConverter(c *C) {
	ins := []string{
		`[17-Sep-2020 12:20:03] NOTICE: fpm is running, pid 83827`,
		`[17-Sep-2020 12:20:03] NOTICE: ready to handle connections`,
		`[17-Sep-2020 12:20:26] NOTICE: Terminating ...`,
		`[17-Sep-2020 12:20:26] NOTICE: exiting, bye-bye!`,
		`[17-Sep-2020 12:25:28] NOTICE: PHP message: PHP Warning:  PHP Startup: Unable to load dynamic library '/app/blackfire-20190902-zts.so' (tried: /app/blackfire-20190902-zts.so (dlopen(/app/blackfire-20190902-zts.so, 9): image not found), /usr/local/lib/php/pecl/20190902//app/blackfire-20190902-zts.so.so (dlopen(/usr/local/lib/php/pecl/20190902//app/blackfire-20190902-zts.so.so, 9): image not found)) in Unknown on line 0`,
		`[17-Sep-2020 12:25:48] NOTICE: PHP message: PHP Warning:  PHP Startup: failed to open stream: No such file or directory in Unknown on line 0`,
		`[17-Sep-2020 12:25:48] NOTICE: PHP message: PHP Fatal error:  PHP Startup: Failed opening required 'foo.php' (include_path='.:/usr/local/Cellar/php/7.4.10/share/php/pear') in Unknown on line 0`,
	}
	expected := []*line{
		{
			time:    time.Date(2020, 9, 17, 12, 20, 03, 0, time.UTC),
			level:   "notice",
			source:  "FPM",
			message: "fpm is running, pid 83827",
			fields:  map[string]string{},
		},
		{
			time:    time.Date(2020, 9, 17, 12, 20, 03, 0, time.UTC),
			level:   "notice",
			source:  "FPM",
			message: "ready to handle connections",
			fields:  map[string]string{},
		},
		{
			time:    time.Date(2020, 9, 17, 12, 20, 26, 0, time.UTC),
			level:   "notice",
			source:  "FPM",
			message: "Terminating ...",
			fields:  map[string]string{},
		},
		{
			time:    time.Date(2020, 9, 17, 12, 20, 26, 0, time.UTC),
			level:   "notice",
			source:  "FPM",
			message: "exiting, bye-bye!",
			fields:  map[string]string{},
		},
		{
			time:    time.Date(2020, 9, 17, 12, 25, 28, 0, time.UTC),
			level:   "warning",
			source:  "FPM",
			message: `Unable to load dynamic library '/app/blackfire-20190902-zts.so' (tried: /app/blackfire-20190902-zts.so (dlopen(/app/blackfire-20190902-zts.so, 9): image not found), /usr/local/lib/php/pecl/20190902//app/blackfire-20190902-zts.so.so (dlopen(/usr/local/lib/php/pecl/20190902//app/blackfire-20190902-zts.so.so, 9): image not found)) in Unknown on line 0`,
			fields:  map[string]string{},
		},
		{
			time:    time.Date(2020, 9, 17, 12, 25, 48, 0, time.UTC),
			level:   "warning",
			source:  "FPM",
			message: `failed to open stream: No such file or directory in Unknown on line 0`,
			fields:  map[string]string{},
		},
		{
			time:    time.Date(2020, 9, 17, 12, 25, 48, 0, time.UTC),
			level:   "fatal",
			source:  "FPM",
			message: `Failed opening required 'foo.php' (include_path='.:/usr/local/Cellar/php/7.4.10/share/php/pear') in Unknown on line 0`,
			fields:  map[string]string{},
		},
	}
	for i, in := range ins {
		out, err := convertPHPFPMLog([]byte(in))
		c.Assert(err, Equals, nil)
		c.Check(out, DeepEquals, expected[i])
	}
}

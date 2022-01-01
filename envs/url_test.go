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
	"encoding/json"

	. "gopkg.in/check.v1"
)

type URLSuite struct{}

var _ = Suite(&URLSuite{})

func (s *URLSuite) TestUnmarshalJSON(c *C) {
	{
		slice := URLSlice{}
		value := []byte(`{"https://market.43zqastgx4-c6yyhsnvmdojk.eu.s5y.io/": {"primary": false, "id": null, "attributes": {}, "type": "upstream", "upstream": "app", "original_url": "https://market.{all}/"}, "https://43zqastgx4-c6yyhsnvmdojk.eu.s5y.io/": {"primary": true, "id": null, "attributes": {}, "type": "upstream", "upstream": "app", "original_url": "https://{all}/"}, "https://admin.43zqastgx4-c6yyhsnvmdojk.eu.s5y.io/": {"primary": false, "id": null, "attributes": {}, "type": "upstream", "upstream": "app", "original_url": "https://admin.{all}/"}, "http://market.43zqastgx4-c6yyhsnvmdojk.eu.s5y.io/": {"to": "https://market.43zqastgx4-c6yyhsnvmdojk.eu.s5y.io/", "original_url": "http://market.{all}/", "type": "redirect", "primary": false, "id": null}, "http://admin.43zqastgx4-c6yyhsnvmdojk.eu.s5y.io/": {"to": "https://admin.43zqastgx4-c6yyhsnvmdojk.eu.s5y.io/", "original_url": "http://admin.{all}/", "type": "redirect", "primary": false, "id": null}, "http://43zqastgx4-c6yyhsnvmdojk.eu.s5y.io/": {"to": "https://43zqastgx4-c6yyhsnvmdojk.eu.s5y.io/", "original_url": "http://{all}/", "type": "redirect", "primary": false, "id": null}}`)
		err := json.Unmarshal(value, &slice)
		c.Check(err, IsNil)

		c.Check(len(slice), Equals, 6)
		c.Check(slice[0].Key, Equals, "https://market.43zqastgx4-c6yyhsnvmdojk.eu.s5y.io/")
		c.Check(slice[1].Key, Equals, "https://43zqastgx4-c6yyhsnvmdojk.eu.s5y.io/")
	}

	{
		slice := URLSlice{}
		value := []byte(`"foo"`)
		err := json.Unmarshal(value, &slice)
		c.Check(err, NotNil)
	}
}

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

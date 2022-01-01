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

package php

import (
	"bytes"

	"github.com/symfony-cli/symfony-cli/envs"
)

type PHPValues []keyVal

type keyVal struct {
	Key string
	Val string
}

func GetPHPINISettings(dir string) *PHPValues {
	values := &PHPValues{}
	values.addBlackfireSocket(dir)
	return values
}

func (v *PHPValues) Bytes() []byte {
	strs := [][]byte{}
	for _, value := range *v {
		strs = append(strs, []byte(value.Key+"="+value.Val))
	}
	return bytes.Join(strs, []byte{'\n'})
}

func (v *PHPValues) merge(m map[string]string) {
	for key, val := range m {
		*v = append(*v, keyVal{Key: key, Val: val})
	}
}

func (v *PHPValues) addBlackfireSocket(dir string) {
	local, err := envs.NewLocal(dir, false)
	if err != nil {
		return
	}
	if local.FindRelationshipPrefix("blackfire", "tcp") == "" {
		return
	}
	if blackfireSocket := envs.AsMap(local)["BLACKFIRE_URL"]; blackfireSocket != "" {
		v.merge(map[string]string{
			"blackfire.agent_socket": blackfireSocket,
		})
	}
}

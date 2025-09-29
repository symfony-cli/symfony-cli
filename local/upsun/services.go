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

type service struct {
	Type     string
	Versions serviceVersions
}

type serviceVersions struct {
	Deprecated []string
	Supported  []string
}

func ServiceLastVersion(name string) string {
	for _, s := range availableServices {
		if s.Type == name {
			versions := s.Versions.Supported
			if len(versions) == 0 {
				versions = s.Versions.Deprecated
			}
			if len(versions) > 0 {
				return versions[len(versions)-1]
			}
		}
	}
	return ""
}

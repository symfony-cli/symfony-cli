/*
 * Copyright (c) 2026-present Fabien Potencier <fabien@symfony.com>
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
	"path/filepath"
	"slices"
	"testing"
)

func TestManagedToolDefinitionsPreserveUpsunReleaseContracts(t *testing.T) {
	tests := []struct {
		name          string
		product       CloudProduct
		definition    string
		versionPrefix string
		assets        []string
	}{
		{
			name:          "Flex",
			product:       Flex,
			definition:    "Upsun Flex CLI",
			versionPrefix: "Upsun CLI ",
			assets: []string{
				"upsun_{version}_linux_amd64.tar.gz",
				"upsun_{version}_linux_arm64.tar.gz",
				"upsun_{version}_darwin_all.tar.gz",
				"upsun_{version}_darwin_all.tar.gz",
				"upsun_{version}_windows_amd64.zip",
			},
		},
		{
			name:          "Fixed",
			product:       Fixed,
			definition:    "Upsun Fixed CLI",
			versionPrefix: "Upsun CLI (Platform.sh compatibility) ",
			assets: []string{
				"platform_{version}_linux_amd64.tar.gz",
				"platform_{version}_linux_arm64.tar.gz",
				"platform_{version}_darwin_all.tar.gz",
				"platform_{version}_darwin_all.tar.gz",
				"platform_{version}_windows_amd64.zip",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			definition := toolDefinition("/home/user", test.product)
			if definition.Name != test.definition || definition.VersionPrefix != test.versionPrefix || definition.ChecksumsAsset != "checksums.txt" {
				t.Fatalf("unexpected definition %#v", definition)
			}
			if definition.InstallRoot != filepath.Join("/home/user", test.product.CLIConfigPath, "tools", test.product.BinName) {
				t.Fatalf("unexpected install root %q", definition.InstallRoot)
			}
			if !slices.Equal(definition.LegacyExecutables, []string{
				filepath.Join("/home/user", test.product.BinaryPath()),
				filepath.Join("/home/user", test.product.BinaryPath()) + ".exe",
			}) {
				t.Fatalf("unexpected legacy paths %#v", definition.LegacyExecutables)
			}
			assets := make([]string, 0, len(definition.Packages))
			for _, pkg := range definition.Packages {
				assets = append(assets, pkg.Asset)
			}
			if !slices.Equal(assets, test.assets) {
				t.Fatalf("unexpected assets %#v", assets)
			}
		})
	}
}

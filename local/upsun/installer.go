/*
 * Copyright (c) 2023-present Fabien Potencier <fabien@symfony.com>
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
	"context"
	"path/filepath"

	"github.com/symfony-cli/symfony-cli/local/externaltool"
	"github.com/symfony-cli/terminal"
)

func Install(home string, product CloudProduct) (string, error) {
	manager := externaltool.NewManager()
	manager.Stderr = terminal.Stderr
	installation, err := manager.Resolve(context.Background(), toolDefinition(home, product))
	if err != nil {
		return "", err
	}

	return installation.Executable, nil
}

func toolDefinition(home string, product CloudProduct) externaltool.Definition {
	assetPrefix := "platform"
	name := product.Name
	if name == "" {
		name = "Platform.sh/Upsun"
	}
	versionPrefix := "Upsun CLI (Platform.sh compatibility) "
	legacyVersionPrefixes := []string{"Platform.sh CLI "}
	if product == Flex {
		assetPrefix = "upsun"
		versionPrefix = "Upsun CLI "
		legacyVersionPrefixes = nil
	}
	legacyExecutable := filepath.Join(home, product.BinaryPath())

	return externaltool.Definition{
		Name:                  name + " CLI",
		Repository:            "platformsh/cli",
		ChecksumsAsset:        "checksums.txt",
		MinimumVersion:        "0.0.0",
		VersionPrefix:         versionPrefix,
		InstallRoot:           filepath.Join(home, product.CLIConfigPath, "tools", product.BinName),
		LegacyExecutables:     []string{legacyExecutable, legacyExecutable + ".exe"},
		LegacyVersionPrefixes: legacyVersionPrefixes,
		Packages: []externaltool.Package{
			{OS: "linux", Arch: "amd64", Asset: assetPrefix + "_{version}_linux_amd64.tar.gz", Executable: product.BinName},
			{OS: "linux", Arch: "arm64", Asset: assetPrefix + "_{version}_linux_arm64.tar.gz", Executable: product.BinName},
			{OS: "darwin", Arch: "amd64", Asset: assetPrefix + "_{version}_darwin_all.tar.gz", Executable: product.BinName},
			{OS: "darwin", Arch: "arm64", Asset: assetPrefix + "_{version}_darwin_all.tar.gz", Executable: product.BinName},
			{OS: "windows", Arch: "amd64", Asset: assetPrefix + "_{version}_windows_amd64.zip", Executable: product.BinName + ".exe"},
		},
	}
}

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

package util

import (
	"fmt"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"sync"

	"github.com/pkg/errors"
	"github.com/symfony-cli/terminal"
)

const ConfigurationDirectory = "symfony-cli"
const LegacyConfigurationDirectory = ".symfony5"

var (
	legacyPathWarning sync.Once
)

func GetHomeDir() string {
	if InCloud() {
		u, err := user.Current()
		if err != nil {
			return filepath.Join(os.TempDir(), ConfigurationDirectory)
		}
		return filepath.Join(os.TempDir(), u.Username, ConfigurationDirectory)
	}

	configurationPath := filepath.Join(".", "."+ConfigurationDirectory)
	if userCfg, err := os.UserConfigDir(); err != nil {
		terminal.Logger.Warn().Err(err).Msg("Could not determine user config dir, using local directory instead")
	} else {
		configurationPath = filepath.Join(userCfg, ConfigurationDirectory)
	}

	// use the legacy path if it exists already
	if home, err := os.UserHomeDir(); err == nil {
		legacyPath := filepath.Join(home, LegacyConfigurationDirectory)
		if _, err := os.Stat(legacyPath); !errors.Is(err, fs.ErrNotExist) {
			terminal.Logger.Warn().Str("directory", legacyPath).Err(err).Msg("Legacy configuration directory detected")
			legacyPathWarning.Do(func() {
				terminal.SymfonyStyle(terminal.Stdout, terminal.Stdin).Warning(fmt.Sprintf("The configuration location for the Symfony CLI has changed in v5.12.0.\nHaving the configuration stored in \"$HOME/.symfony5\" is deprecated and will not be supported in next major versions.Please migrate the \"%s\" directory manually to \"%s\" at your earliest convenience after stopping the proxy and every instances running.", legacyPath, configurationPath))
			})
			return legacyPath
		}
	}

	return configurationPath
}

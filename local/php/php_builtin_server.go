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
	"fmt"
	"os"
	"path/filepath"
)

var phprouter = []byte(`<?php
/*
 * This file is part of the Symfony package.
 *
 * (c) Fabien Potencier <fabien@symfony.com>
 *
 * For the full copyright and license information, please view the LICENSE
 * file that was distributed with this source code.
 */

// Workaround https://bugs.php.net/64566
if (ini_get('auto_prepend_file') && !in_array(realpath(ini_get('auto_prepend_file')), get_included_files(), true)) {
	require ini_get('auto_prepend_file');
}

if (isset($_SERVER['HTTP___SYMFONY_LOCAL_REQUEST_ID__'])) {
	if (file_exists($envFile = __FILE__.'-'.$_SERVER['HTTP___SYMFONY_LOCAL_REQUEST_ID__'].'-env')) {
		require $envFile;
	}
	unset($_SERVER['HTTP___SYMFONY_LOCAL_REQUEST_ID__']);
}

$_SERVER = array_merge($_SERVER, $_ENV);

if (is_file($_SERVER['DOCUMENT_ROOT'].DIRECTORY_SEPARATOR.$_SERVER['SCRIPT_NAME'])) {
	return false;
}

$script = $_ENV['APP_FRONT_CONTROLLER'];
$_SERVER['SCRIPT_FILENAME'] = $_SERVER['DOCUMENT_ROOT'].DIRECTORY_SEPARATOR.$script;
$_SERVER['SCRIPT_NAME'] = DIRECTORY_SEPARATOR.$script;
$_SERVER['PHP_SELF'] = DIRECTORY_SEPARATOR.$script;

require $script;
`)

func (p *Server) phpRouterFile() string {
	path := filepath.Join(p.homeDir, fmt.Sprintf("php/%s-router.php", name(p.projectDir)))
	if _, err := os.Stat(filepath.Dir(path)); os.IsNotExist(err) {
		_ = os.MkdirAll(filepath.Dir(path), 0755)
	}
	return path
}

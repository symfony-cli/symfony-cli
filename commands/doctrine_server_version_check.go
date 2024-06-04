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

package commands

import (
	"fmt"

	"github.com/symfony-cli/symfony-cli/local/platformsh"
)

// checkDoctrineServerVersionSetting checks that project has a DB and that server version is set properly
func checkDoctrineServerVersionSetting(projectDir string) error {
	if len(platformsh.FindLocalApplications(projectDir)) > 1 {
		// not implemented yet as more complex
		return nil
	}

	configFile, dbName, dbVersion := platformsh.ReadDBVersionFromPlatformServiceYAML(projectDir)
	if dbName == "" {
		// no DB
		return nil
	}

	errorTpl := fmt.Sprintf(`
 The "%s" file defines
 a "%s" version %s database service
 but %%s.
 
 Before deploying, fix the version mismatch.
 `, configFile, dbName, dbVersion)

	dotEnvVersion, err := platformsh.ReadDBVersionFromDotEnv(projectDir)
	if err != nil {
		return nil
	}
	if platformsh.DatabaseVersiondUnsynced(dotEnvVersion, dbVersion) {
		return fmt.Errorf(errorTpl, fmt.Sprintf("the \".env\" file requires version %s", dotEnvVersion))
	}

	doctrineConfigVersion, err := platformsh.ReadDBVersionFromDoctrineConfigYAML(projectDir)
	if err != nil {
		return nil
	}
	if platformsh.DatabaseVersiondUnsynced(doctrineConfigVersion, dbVersion) {
		return fmt.Errorf(errorTpl, fmt.Sprintf("the \"config/packages/doctrine.yaml\" file requires version %s", doctrineConfigVersion))
	}

	if dotEnvVersion == "" && doctrineConfigVersion == "" {
		return fmt.Errorf(`
 The "%s" file defines a "%s" database service.
 
 When deploying, Doctrine needs to know the database version to determine the supported SQL syntax.
 
 As the database is not available when Doctrine is warming up its cache on Platform.sh,
 you need to explicitly set the database version in the ".env" or "config/packages/doctrine.yaml" file.
 
 Set the "server_version" parameter to "%s" in "config/packages/doctrine.yaml".
 `, configFile, dbName, dbVersion)
	}

	fmt.Println("✅ Doctrine server version is set properly.")

	return nil
}

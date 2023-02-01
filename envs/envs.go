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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/symfony-cli/symfony-cli/util"
)

func GetEnv(dir string, debug bool) (Environment, error) {
	// are we remote or local?
	if util.InCloud() {
		return &Remote{Debug: debug}, nil
	}

	return NewLocal(dir, debug)
}

// Environment knows how to extract env vars (local or remote)
type Environment interface {
	Path() string
	Mailer() Envs
	Language() string
	Relationships() Relationships
	Extra() Envs
	Local() bool
}

type Relationships map[string][]map[string]interface{}

type Envs map[string]string

// AsSlice returns the extracted environment variables
func AsSlice(env Environment) []string {
	envs := []string{}
	for key, value := range AsMap(env) {
		envs = append(envs, fmt.Sprintf("%s=%s", key, value))
	}

	return envs
}

// AsString returns a string representation of the environment variables
func AsString(env Environment) string {
	return strings.Join(AsSlice(env), " ")
}

// AsMap returns the extracted environment variables
func AsMap(env Environment) map[string]string {
	envs := Envs{}
	appID := appID(env.Path())
	if appID != "" {
		envs["APP_ID"] = appID
	}
	for k, v := range extractRelationshipsEnvs(env) {
		envs[k] = v
	}
	for k, v := range env.Mailer() {
		envs[k] = v
	}
	for k, v := range env.Extra() {
		envs[k] = v
	}
	return envs
}

// appID returns the Symfony project's ID from composer.json
func appID(path string) string {
	content, err := ioutil.ReadFile(filepath.Join(path, "composer.json"))
	if err != nil {
		return ""
	}
	var composer struct {
		Extra struct {
			Symfony struct {
				ID string `json:"id"`
			} `json:"symfony"`
		} `json:"extra"`
	}
	if err := json.Unmarshal(content, &composer); err != nil {
		return ""
	}
	return composer.Extra.Symfony.ID
}

func extractRelationshipsEnvs(env Environment) Envs {
	values := Envs{}
	for key, allValues := range env.Relationships() {
		key = strings.ToUpper(key)

		for i, endpoint := range allValues {
			scheme := endpoint["scheme"]
			rel := endpoint["rel"]

			prefix := fmt.Sprintf("%s_", key)
			if i != 0 {
				prefix = fmt.Sprintf("%s_%d_", key, i)
			}
			prefix = strings.Replace(prefix, "-", "_", -1)

			if scheme == "pgsql" || scheme == "mysql" {
				if !isMaster(endpoint) {
					continue
				}
				if scheme == "pgsql" {
					// works for both Doctrine and Go
					endpoint["scheme"] = "postgres"
				}
				url := fmt.Sprintf("%s://", endpoint["scheme"].(string))
				if username, ok := endpoint["username"].(string); ok && username != "" {
					url += username
					values[fmt.Sprintf("%sUSER", prefix)] = username
					values[fmt.Sprintf("%sUSERNAME", prefix)] = username

					if password, ok := endpoint["password"].(string); ok && password != "" {
						url += fmt.Sprintf(":%s", password)
						values[fmt.Sprintf("%sPASSWORD", prefix)] = password
					}
					url += "@"
				}

				path := "main"
				if p, ok := endpoint["path"].(string); ok && p != "" {
					path = p
				}
				url += fmt.Sprintf("%s:%s/%s?sslmode=disable", endpoint["host"].(string), formatInt(endpoint["port"]), path)
				values[fmt.Sprintf("%sURL", prefix)] = url
				if env.Language() != "golang" {
					charset := "utf8"
					if envCharset := os.Getenv(fmt.Sprintf("%sCHARSET", prefix)); envCharset != "" {
						charset = envCharset
					} else if scheme == "mysql" {
						charset = "utf8mb4"
					}
					values[fmt.Sprintf("%sURL", prefix)] = values[fmt.Sprintf("%sURL", prefix)] + "&charset=" + charset
				}
				if env.Language() == "php" {
					if v, ok := endpoint["type"]; ok {
						versionKey := fmt.Sprintf("%sVERSION", prefix)
						if version, hasVersionInEnv := os.LookupEnv(versionKey); hasVersionInEnv {
							values[versionKey] = version
							values[fmt.Sprintf("%sURL", prefix)] = values[fmt.Sprintf("%sURL", prefix)] + "&serverVersion=" + values[versionKey]
						} else if strings.Contains(v.(string), ":") {
							version := strings.SplitN(v.(string), ":", 2)[1]

							// we actually provide mariadb not mysql
							if endpoint["scheme"].(string) == "mysql" {
								minor := 0
								if version == "10.2" {
									minor = 7
								}
								version = fmt.Sprintf("mariadb-%s.%d", version, minor)
							}

							values[versionKey] = version
							values[fmt.Sprintf("%sURL", prefix)] = values[fmt.Sprintf("%sURL", prefix)] + "&serverVersion=" + values[versionKey]
						}
					}
				}
				values[fmt.Sprintf("%sSERVER", prefix)] = formatServer(endpoint)
				values[fmt.Sprintf("%sDRIVER", prefix)] = endpoint["scheme"].(string)
				values[fmt.Sprintf("%sHOST", prefix)] = endpoint["host"].(string)
				values[fmt.Sprintf("%sPORT", prefix)] = formatInt(endpoint["port"])
				values[fmt.Sprintf("%sNAME", prefix)] = path
				values[fmt.Sprintf("%sDATABASE", prefix)] = path

				if env.Local() {
					if scheme == "pgsql" {
						values["PGHOST"] = endpoint["host"].(string)
						values["PGPORT"] = formatInt(endpoint["port"])
						values["PGDATABASE"] = path
						values["PGUSER"] = endpoint["username"].(string)
						values["PGPASSWORD"] = endpoint["password"].(string)
					} else if scheme == "mysql" {
						values["MYSQL_HOST"] = endpoint["host"].(string)
						values["MYSQL_TCP_PORT"] = formatInt(endpoint["port"])
					}
				}
			} else if scheme == "redis" {
				values[fmt.Sprintf("%sURL", prefix)] = fmt.Sprintf("redis://%s:%s", endpoint["host"].(string), formatInt(endpoint["port"]))
				values[fmt.Sprintf("%sHOST", prefix)] = endpoint["host"].(string)
				values[fmt.Sprintf("%sPORT", prefix)] = formatInt(endpoint["port"])
				values[fmt.Sprintf("%sSCHEME", prefix)] = endpoint["scheme"].(string)
			} else if scheme == "solr" {
				values[fmt.Sprintf("%sHOST", prefix)] = endpoint["host"].(string)
				values[fmt.Sprintf("%sPORT", prefix)] = formatInt(endpoint["port"])
				values[fmt.Sprintf("%sNAME", prefix)] = endpoint["path"].(string)
				values[fmt.Sprintf("%sDATABASE", prefix)] = endpoint["path"].(string)
			} else if rel == "elasticsearch" {
				path, hasPath := endpoint["path"]
				if !hasPath || path == nil {
					path = ""
				}
				values[fmt.Sprintf("%sURL", prefix)] = fmt.Sprintf("%s://%s:%s%s", endpoint["scheme"].(string), endpoint["host"].(string), formatInt(endpoint["port"]), path)
				values[fmt.Sprintf("%sHOST", prefix)] = endpoint["host"].(string)
				values[fmt.Sprintf("%sPORT", prefix)] = formatInt(endpoint["port"])
				values[fmt.Sprintf("%sSCHEME", prefix)] = endpoint["scheme"].(string)
			} else if scheme == "mongodb" {
				if !isMaster(endpoint) {
					continue
				}
				values[fmt.Sprintf("%sSERVER", prefix)] = formatServer(endpoint)
				values[fmt.Sprintf("%sHOST", prefix)] = endpoint["host"].(string)
				values[fmt.Sprintf("%sPORT", prefix)] = formatInt(endpoint["port"])
				values[fmt.Sprintf("%sSCHEME", prefix)] = endpoint["scheme"].(string)
				values[fmt.Sprintf("%sNAME", prefix)] = endpoint["path"].(string)
				values[fmt.Sprintf("%sDATABASE", prefix)] = endpoint["path"].(string)
				values[fmt.Sprintf("%sUSER", prefix)] = endpoint["username"].(string)
				values[fmt.Sprintf("%sUSERNAME", prefix)] = endpoint["username"].(string)
				values[fmt.Sprintf("%sPASSWORD", prefix)] = endpoint["password"].(string)
			} else if scheme == "amqp" {
				values[fmt.Sprintf("%sURL", prefix)] = fmt.Sprintf("%s://%s:%s@%s:%s", endpoint["scheme"].(string), endpoint["username"].(string), endpoint["password"].(string), endpoint["host"].(string), formatInt(endpoint["port"]))
				values[fmt.Sprintf("%sDSN", prefix)] = fmt.Sprintf("%s://%s:%s@%s:%s", endpoint["scheme"].(string), endpoint["username"].(string), endpoint["password"].(string), endpoint["host"].(string), formatInt(endpoint["port"]))
				values[fmt.Sprintf("%sSERVER", prefix)] = formatServer(endpoint)
				values[fmt.Sprintf("%sHOST", prefix)] = endpoint["host"].(string)
				values[fmt.Sprintf("%sPORT", prefix)] = formatInt(endpoint["port"])
				values[fmt.Sprintf("%sSCHEME", prefix)] = endpoint["scheme"].(string)
				values[fmt.Sprintf("%sUSER", prefix)] = endpoint["username"].(string)
				values[fmt.Sprintf("%sUSERNAME", prefix)] = endpoint["username"].(string)
				values[fmt.Sprintf("%sPASSWORD", prefix)] = endpoint["password"].(string)
			} else if scheme == "memcached" {
				values[fmt.Sprintf("%sHOST", prefix)] = endpoint["host"].(string)
				values[fmt.Sprintf("%sPORT", prefix)] = formatInt(endpoint["port"])
				values[fmt.Sprintf("%sIP", prefix)] = endpoint["ip"].(string)
			} else if rel == "influxdb" {
				values[fmt.Sprintf("%sSCHEME", prefix)] = endpoint["scheme"].(string)
				values[fmt.Sprintf("%sHOST", prefix)] = endpoint["host"].(string)
				values[fmt.Sprintf("%sPORT", prefix)] = formatInt(endpoint["port"])
				values[fmt.Sprintf("%sIP", prefix)] = endpoint["ip"].(string)
			} else if scheme == "kafka" {
				values[fmt.Sprintf("%sURL", prefix)] = fmt.Sprintf("%s://%s:%s", endpoint["scheme"].(string), endpoint["host"].(string), formatInt(endpoint["port"]))
				values[fmt.Sprintf("%sSCHEME", prefix)] = endpoint["scheme"].(string)
				values[fmt.Sprintf("%sHOST", prefix)] = endpoint["host"].(string)
				values[fmt.Sprintf("%sPORT", prefix)] = formatInt(endpoint["port"])
				values[fmt.Sprintf("%sIP", prefix)] = endpoint["ip"].(string)
			} else if scheme == "tcp" {
				values[fmt.Sprintf("%sURL", prefix)] = formatServer(endpoint)
				values[fmt.Sprintf("%sIP", prefix)] = endpoint["ip"].(string)
				values[fmt.Sprintf("%sPORT", prefix)] = formatInt(endpoint["port"])
				values[fmt.Sprintf("%sSCHEME", prefix)] = endpoint["scheme"].(string)
				values[fmt.Sprintf("%sHOST", prefix)] = endpoint["host"].(string)
				if rel == "blackfire" {
					values["BLACKFIRE_AGENT_SOCKET"] = values[fmt.Sprintf("%sURL", prefix)]
				}
			} else if rel == "mercure" {
				values["MERCURE_URL"] = fmt.Sprintf("%s://%s:%s/.well-known/mercure", endpoint["scheme"].(string), endpoint["host"].(string), formatInt(endpoint["port"]))
				values["MERCURE_PUBLIC_URL"] = values["MERCURE_URL"]
			} else if scheme == "http" || scheme == "https" {
				username, hasUsername := endpoint["username"].(string)
				password, hasPassword := endpoint["password"].(string)
				if hasUsername || hasPassword {
					values[fmt.Sprintf("%sURL", prefix)] = fmt.Sprintf("%s://%s:%s@%s:%s", endpoint["scheme"].(string), username, password, endpoint["host"].(string), formatInt(endpoint["port"]))
				} else {
					values[fmt.Sprintf("%sURL", prefix)] = fmt.Sprintf("%s://%s:%s", endpoint["scheme"].(string), endpoint["host"].(string), formatInt(endpoint["port"]))
				}
				values[fmt.Sprintf("%sSERVER", prefix)] = formatServer(endpoint)
				values[fmt.Sprintf("%sIP", prefix)] = endpoint["ip"].(string)
				values[fmt.Sprintf("%sPORT", prefix)] = formatInt(endpoint["port"])
				values[fmt.Sprintf("%sSCHEME", prefix)] = endpoint["scheme"].(string)
				values[fmt.Sprintf("%sHOST", prefix)] = endpoint["host"].(string)
				if hasUsername {
					values[fmt.Sprintf("%sUSER", prefix)] = endpoint["username"].(string)
					values[fmt.Sprintf("%sUSERNAME", prefix)] = endpoint["username"].(string)
				}
				if hasPassword {
					values[fmt.Sprintf("%sPASSWORD", prefix)] = endpoint["password"].(string)
				}
			} else if scheme == "smtp" {
				values["MAILER_CATCHER"] = "1"

				// for Laravel Swiftmailer, use a MAIL prefix
				// for Swiftmailer, use a MAILER prefix
				values[fmt.Sprintf("%sDRIVER", prefix)] = endpoint["scheme"].(string)
				values[fmt.Sprintf("%sHOST", prefix)] = endpoint["host"].(string)
				values[fmt.Sprintf("%sPORT", prefix)] = formatInt(endpoint["port"])
				values[fmt.Sprintf("%sUSERNAME", prefix)] = ""
				values[fmt.Sprintf("%sPASSWORD", prefix)] = ""
				values[fmt.Sprintf("%sAUTH_MODE", prefix)] = ""
				values[fmt.Sprintf("%sURL", prefix)] = fmt.Sprintf("%s://%s:%s", endpoint["scheme"].(string), endpoint["host"].(string), formatInt(endpoint["port"]))

				// for Symfony Mailer, use a MAILER prefix
				values[fmt.Sprintf("%sDSN", prefix)] = fmt.Sprintf("%s://%s:%s", endpoint["scheme"].(string), endpoint["host"].(string), formatInt(endpoint["port"]))
			} else if rel == "simple" {
				values[fmt.Sprintf("%sIP", prefix)] = endpoint["ip"].(string)
				values[fmt.Sprintf("%sPORT", prefix)] = formatInt(endpoint["port"])
				values[fmt.Sprintf("%sHOST", prefix)] = endpoint["host"].(string)
			}
		}
	}
	return values
}

func formatServer(endpoint map[string]interface{}) string {
	return fmt.Sprintf("%s://%s:%s", endpoint["scheme"].(string), endpoint["host"].(string), formatInt(endpoint["port"]))
}

func formatInt(val interface{}) string {
	if s, ok := val.(string); ok {
		return s
	}
	if i, ok := val.(int); ok {
		return strconv.Itoa(i)
	}
	return strconv.FormatInt(int64(val.(float64)), 10)
}

// isMaster determines if the given relationship is Master or Slave in the
// context of a Master-Slave database configuration. Defaults to true if the
// relationship can not be in a Master-Slave configuration or if the status can
// not be determined.
func isMaster(endpoint map[string]interface{}) bool {
	if val, ok := endpoint["query"]; ok {
		if query, ok := val.(map[string]interface{}); ok {
			if isMaster, ok := query["is_master"].(bool); ok {
				return isMaster
			}
		}
	}

	return true
}

func isMailerDefined() bool {
	if _, ok := os.LookupEnv("MAILER_URL"); ok {
		return true
	}
	if _, ok := os.LookupEnv("MAILER_DSN"); ok {
		return true
	}
	if _, ok := os.LookupEnv("MAILER_HOST"); ok {
		return true
	}
	return false
}

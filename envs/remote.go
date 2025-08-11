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
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strings"
)

// Remote represents the Symfony platform
type Remote struct {
	Debug bool
}

// Path returns the project's path
func (r *Remote) Path() string {
	if dir := os.Getenv("PLATFORM_APP_DIR"); dir != "" {
		return dir
	}
	return "/app"
}

// Local returns true if the command is used on a local machine
func (r *Remote) Local() bool {
	return false
}

// Relationships returns envs from Symfony relationships
func (r *Remote) Relationships() Relationships {
	var data []byte
	relationships, exists := os.LookupEnv("PLATFORM_RELATIONSHIPS")
	if !exists {
		// we are probably during the build phase
		if r.Debug {
			fmt.Fprint(os.Stderr, "PLATFORM_RELATIONSHIPS env var does not exist\n")
		}
		return nil
	}

	var err error
	data, err = base64.StdEncoding.DecodeString(relationships)
	if err != nil {
		if r.Debug {
			fmt.Fprintf(os.Stderr, "unable to decode PLATFORM_RELATIONSHIPS: %s\n", err)
		}
		return nil
	}

	var res Relationships
	if err := json.Unmarshal(data, &res); err != nil {
		if r.Debug {
			fmt.Fprintf(os.Stderr, "unable to unmarshal PLATFORM_RELATIONSHIPS: %s\n", err)
		}
		return nil
	}

	for name, allValues := range res {
		for _, endpoint := range allValues {
			scheme := endpoint["scheme"]
			if scheme != "amqp" {
				continue
			}

			name = name + "-management"

			newEndpoint := map[string]interface{}{}
			for k, v := range endpoint {
				newEndpoint[k] = v
			}
			newEndpoint["port"] = "15672"
			newEndpoint["scheme"] = "http"

			res[name] = []map[string]interface{}{newEndpoint}
		}
	}

	return res
}

// Mailer returns MAILER_* env vars
func (r *Remote) Mailer() Envs {
	if isMailerDefined() {
		return Envs{
			"MAILER_ENABLED": "1",
			//"MAILER_CATCHER":   "0",
		}
	}

	v := Envs{
		"MAILER_ENABLED": "1",
		//"MAILER_CATCHER":   "0",
		"MAILER_PORT":      "25",
		"MAILER_TRANSPORT": "smtp",
		"MAILER_AUTH_MODE": "plain",
		"MAILER_USER":      "",
		"MAILER_PASSWORD":  "",
	}
	host := os.Getenv("PLATFORM_SMTP_HOST")
	if r.Debug {
		fmt.Fprintf(os.Stderr, "reading PLATFORM_SMTP_HOST: %s\n", host)
	}
	if host == "" {
		v["MAILER_ENABLED"] = "0"
		v["MAILER_URL"] = "null://localhost"
		v["MAILER_DSN"] = "null://localhost"
		v["MAILER_HOST"] = "localhost"
		return v
	}
	host = strings.TrimSuffix(host, ":"+v["MAILER_PORT"])

	v["MAILER_URL"] = fmt.Sprintf("smtp://%s:%s?verify_peer=0", host, v["MAILER_PORT"])
	v["MAILER_DSN"] = fmt.Sprintf("smtp://%s:%s?verify_peer=0", host, v["MAILER_PORT"])
	v["MAILER_HOST"] = host
	return v
}

// Extra adds some env specific env vars
func (r *Remote) Extra() Envs {
	v := Envs{}
	// define default variables if not already defined (Symfony ones)
	if env, ok := os.LookupEnv("APP_ENV"); ok {
		if r.Debug {
			fmt.Fprintf(os.Stderr, "adding SYMFONY_ENV: %v (from APP_ENV)\n", env)
		}
		v["SYMFONY_ENV"] = env
	} else if env, ok := os.LookupEnv("SYMFONY_ENV"); ok {
		if r.Debug {
			fmt.Fprintf(os.Stderr, "adding APP_ENV: %v (from SYMFONY_ENV)\n", env)
		}
		v["APP_ENV"] = env
	} else {
		if r.Debug {
			fmt.Fprint(os.Stderr, "adding APP_ENV and SYMFONY_ENV: prod\n")
		}
		v["APP_ENV"] = "prod"
		v["SYMFONY_ENV"] = "prod"
	}

	if debug, ok := os.LookupEnv("APP_DEBUG"); ok {
		if r.Debug {
			fmt.Fprintf(os.Stderr, "adding SYMFONY_DEBUG: %v (from APP_DEBUG)\n", debug)
		}
		v["SYMFONY_DEBUG"] = debug
	} else if debug, ok := os.LookupEnv("SYMFONY_DEBUG"); ok {
		if r.Debug {
			fmt.Fprintf(os.Stderr, "adding APP_DEBUG: %v (from SYMFONY_DEBUG)\n", debug)
		}
		v["APP_DEBUG"] = debug
	} else {
		if r.Debug {
			fmt.Fprint(os.Stderr, "adding APP_DEBUG and SYMFONY_DEBUG: 0\n")
		}
		v["APP_DEBUG"] = "0"
		v["SYMFONY_DEBUG"] = "0"
	}

	if _, ok := os.LookupEnv("APP_SECRET"); !ok {
		if entropy, ok := os.LookupEnv("PLATFORM_PROJECT_ENTROPY"); ok {
			if r.Debug {
				fmt.Fprintf(os.Stderr, "adding APP_SECRET: %s\n", v["PLATFORM_PROJECT_ENTROPY"])
			}
			v["APP_SECRET"] = entropy
		}
	}
	if value := r.extractProjectDefaultUrl(); value != nil {
		port := value.Port()
		if port == "" {
			if value.Scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}

		for _, prefix := range []string{"SYMFONY_PROJECT_DEFAULT_ROUTE_", "SYMFONY_DEFAULT_ROUTE_"} {
			v[fmt.Sprintf("%sURL", prefix)] = value.String()
			v[fmt.Sprintf("%sHOST", prefix)] = value.Host
			v[fmt.Sprintf("%sSCHEME", prefix)] = value.Scheme
			v[fmt.Sprintf("%sPATH", prefix)] = value.Path
			v[fmt.Sprintf("%sPORT", prefix)] = port
		}

		v["DEFAULT_URI"] = value.String()
	}
	if value := r.extractApplicationDefaultUrl(); value != nil {
		port := value.Port()
		if port == "" {
			if value.Scheme == "https" {
				port = "443"
			} else {
				port = "80"
			}
		}

		prefix := "SYMFONY_APPLICATION_DEFAULT_ROUTE_"
		v[fmt.Sprintf("%sURL", prefix)] = value.String()
		v[fmt.Sprintf("%sHOST", prefix)] = value.Host
		v[fmt.Sprintf("%sSCHEME", prefix)] = value.Scheme
		v[fmt.Sprintf("%sPATH", prefix)] = value.Path
		v[fmt.Sprintf("%sPORT", prefix)] = port
	}
	if application, exists := os.LookupEnv("PLATFORM_APPLICATION_NAME"); exists {
		if strings.Contains(application, "--") {
			v["SYMFONY_IS_WORKER"] = "1"
		} else {
			v["SYMFONY_IS_WORKER"] = "0"
		}
	}
	// add some default vars from crons
	if _, ok := os.LookupEnv("MAILFROM"); !ok {
		if from := os.Getenv("PLATFORM_PROJECT"); from != "" {
			if env := os.Getenv("PLATFORM_BRANCH"); env != "master" {
				from += "+"
				from += env
			} else if env := os.Getenv("PLATFORM_ENVIRONMENT"); !strings.HasPrefix(env, "master-") {
				from += "+"
				from += env
			}
			v["MAILFROM"] = fmt.Sprintf("%s@cron.noreply.s5y.io", from)
		}
	}
	return v
}

func (r *Remote) extractProjectDefaultUrl() *url.URL {
	application, _ := os.LookupEnv("PLATFORM_APPLICATION_NAME")
	if index := strings.Index(application, "--"); index != -1 {
		application = application[0:index]
	}

	routes, _ := os.LookupEnv("PLATFORM_ROUTES")

	data, err := base64.StdEncoding.DecodeString(routes)
	if err != nil {
		if r.Debug {
			fmt.Fprintf(os.Stderr, "unable to decode PLATFORM_ROUTES: %s\n", err)
		}
		return nil
	}

	var urls URLSlice
	if err := json.Unmarshal(data, &urls); err != nil {
		if r.Debug {
			fmt.Fprintf(os.Stderr, "unable to unmarshal PLATFORM_ROUTES: %s\n", err)
		}
		return nil
	}

	possibleUrls := make([]URL, 0, len(urls))

	// Filtering pass
	for _, value := range urls {
		// only routes with upstream (no redirect)
		if value.Kind != "upstream" {
			continue
		}

		// with a valid URL
		value.url, err = url.Parse(value.Key)
		if err != nil {
			continue
		}

		possibleUrls = append(possibleUrls, value)
	}

	if len(possibleUrls) == 0 {
		return nil
	}

	// first match on the main {default} or {all} route, then match on a generic www. subdomain
	for _, match := range []string{"://{default}/", "://{all}/", "://www.{default}/", "://www.{all}/"} {
		for _, value := range possibleUrls {
			if strings.HasSuffix(value.OriginalURL, match) {
				return value.url
			}
		}
	}

	// then match on a generic www. subdomain, then {default} or {all} and a path within the app
	for _, fallback := range []string{"://{default}/", "://{all}/", "://www.{default}/", "://www.{all}/"} {
		for _, value := range possibleUrls {
			if value.Upstream == application && strings.Contains(value.OriginalURL, fallback) {
				return value.url
			}
		}
	}

	// Fallback case: we take the first one (in the declared order) for current app
	for _, value := range possibleUrls {
		if value.Upstream == application {
			return value.url
		}
	}

	// Last resort: we take the first one (in the declared order) for the project
	return possibleUrls[0].url
}

func (r *Remote) extractApplicationDefaultUrl() *url.URL {
	application, _ := os.LookupEnv("PLATFORM_APPLICATION_NAME")
	if index := strings.Index(application, "--"); index != -1 {
		application = application[0:index]
	}

	routes, _ := os.LookupEnv("PLATFORM_ROUTES")
	data, err := base64.StdEncoding.DecodeString(routes)
	if err != nil {
		if r.Debug {
			fmt.Fprintf(os.Stderr, "unable to decode PLATFORM_ROUTES: %s\n", err)
		}
		return nil
	}

	var urls URLSlice
	if err := json.Unmarshal(data, &urls); err != nil {
		if r.Debug {
			fmt.Fprintf(os.Stderr, "unable to unmarshal PLATFORM_ROUTES: %s\n", err)
		}
		return nil
	}

	possibleUrls := make([]URL, 0, len(urls))

	// Filtering pass
	for _, value := range urls {
		// only routes with upstream (no redirect)
		if value.Kind != "upstream" {
			continue
		}

		// only routes for the current app
		if value.Upstream != application {
			continue
		}

		// with a valid URL
		value.url, err = url.Parse(value.Key)
		if err != nil {
			continue
		}

		possibleUrls = append(possibleUrls, value)
	}

	if len(possibleUrls) == 0 {
		return nil
	}

	// first match on the main {default} or {all} route, then match on a generic www. subdomain
	for _, match := range []string{"://{default}/", "://{all}/", "://www.{default}/", "://www.{all}/"} {
		for _, value := range possibleUrls {
			if strings.HasSuffix(value.OriginalURL, match) {
				return value.url
			}
		}
	}

	// then match on a generic www. subdomain, then {default} or {all} and a path within the app
	for _, fallback := range []string{"://{default}/", "://{all}/", "://www.{default}/", "://www.{all}/"} {
		for _, value := range possibleUrls {
			if strings.Contains(value.OriginalURL, fallback) {
				return value.url
			}
		}
	}

	// Last resort: we take the first one (in the declared order)
	return possibleUrls[0].url
}

func (r *Remote) Language() string {
	application, _ := os.LookupEnv("PLATFORM_APPLICATION")
	data, err := base64.StdEncoding.DecodeString(application)
	if err != nil {
		if r.Debug {
			fmt.Fprintf(os.Stderr, "unable to decode PLATFORM_APPLICATION: %s\n", err)
		}
		return "php"
	}
	var app struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(data, &app); err != nil {
		if r.Debug {
			fmt.Fprintf(os.Stderr, "unable to unmarshal PLATFORM_APPLICATION: %s\n", err)
		}
		return "php"
	}
	parts := strings.Split(app.Type, ":")
	return parts[0]
}

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

package project

import "github.com/symfony-cli/console"

var ConfigurationFlags = []console.Flag{
	&console.BoolFlag{Name: "allow-http", Usage: "Prevent auto-redirection from HTTP to HTTPS"},
	&console.StringFlag{Name: "document-root", Usage: "Project document root (auto-configured by default)"},
	&console.StringFlag{Name: "passthru", Usage: "Project passthru index (auto-configured by default)"},
	&console.IntFlag{Name: "port", DefaultValue: 8000, Usage: "Preferred HTTP port"},
	&console.StringFlag{Name: "listen-ip", DefaultValue: "127.0.0.1", Usage: "The IP on which the CLI should listen"},
	&console.BoolFlag{Name: "allow-all-ip", Usage: "Listen on all the available interfaces"},
	&console.BoolFlag{Name: "daemon", Aliases: []string{"d"}, Usage: "Run the server in the background"},
	&console.StringFlag{Name: "p12", Usage: "Name of the file containing the TLS certificate to use in p12 format"},
	&console.BoolFlag{Name: "no-tls", Usage: "Use HTTP instead of HTTPS"},
	&console.BoolFlag{Name: "use-gzip", Usage: "Use GZIP"},
	&console.StringFlag{
		Name:  "tls-key-log-file",
		Usage: "Destination for TLS master secrets in NSS key log format",
		// If 'SSLKEYLOGFILE' environment variable is set, uses this as a
		// destination of TLS key log. In this context, the name
		// 'SSLKEYLOGFILE' is common, so using 'SSL' instead of 'TLS' name.
		// This environment variable is preferred than the key log file
		// from the console argument.
		EnvVars: []string{"SSLKEYLOGFILE"},
	},
	&console.BoolFlag{Name: "no-workers", Usage: "Do not start workers"},
	&console.BoolFlag{Name: "allow-cors", Usage: "Allow Cross-origin resource sharing (CORS) requests"},
}

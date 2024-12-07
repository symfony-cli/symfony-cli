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

package http

import (
	"net/http"

	"github.com/rs/zerolog"
)

func corsWrapper(h http.Handler, logger zerolog.Logger) http.Handler {
	var corsHeaders = []string{"Access-Control-Allow-Origin", "Access-Control-Allow-Methods", "Access-Control-Allow-Headers"}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		for _, corsHeader := range corsHeaders {
			w.Header().Set(corsHeader, "*")
		}

		h.ServeHTTP(w, r)

		for _, corsHeader := range corsHeaders {
			if headers, exists := w.Header()[corsHeader]; !exists || len(headers) < 2 {
				continue
			}

			logger.Warn().Msgf(`Multiple entries detected for header "%s". Only one should be set: you should enable CORS handling in the CLI only if the application does not handle them.`, corsHeader)
		}
	})
}

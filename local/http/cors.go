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
)

func corsWrapper(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)

		if r.Method != http.MethodOptions {
			return
		}

		if w.Header().Get("Access-Control-Allow-Origin") == "" {
			w.Header().Add("Access-Control-Allow-Origin", "*")
		}

		if w.Header().Get("Access-Control-Allow-Methods") == "" {
			w.Header().Add("Access-Control-Allow-Methods", "*")
		}

		if w.Header().Get("Access-Control-Allow-Headers") == "" {
			w.Header().Add("Access-Control-Allow-Headers", "*")
		}
	})
}

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
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"regexp"

	"github.com/pkg/errors"
	"github.com/symfony-cli/symfony-cli/envs"
	"github.com/symfony-cli/terminal"
)

func (p *Server) tweakToolbar(body io.ReadCloser, env map[string]string) (io.ReadCloser, error) {
	// CGI adds a \n at the start of the toolbar code
	bn := bytes.Repeat([]byte{' '}, 1)
	n, err := body.Read(bn)
	// if body is empty, return immediately
	if n == 0 && err == io.EOF {
		return ioutil.NopCloser(bytes.NewReader([]byte{})), nil
	}
	if n == len(bn) && err != nil {
		return nil, errors.WithStack(err)
	}
	if bn[0] != '\n' && bn[0] != '<' {
		return ioutil.NopCloser(io.MultiReader(bytes.NewReader(bn), body)), nil
	}

	toolbarHint := []byte("<!-- START of Symfony Web Debug Toolbar -->")
	if bn[0] == '<' {
		toolbarHint = toolbarHint[1:]
	}
	start := bytes.Repeat([]byte{' '}, len(toolbarHint))
	n, err = body.Read(start)
	if n == len(start) && err != nil {
		return nil, errors.WithStack(err)
	}
	if n != len(toolbarHint) || !bytes.Equal(start, toolbarHint) {
		return ioutil.NopCloser(io.MultiReader(bytes.NewReader(bn), bytes.NewReader(start), body)), nil
	}

	logoBg := "sf-toolbar-status-normal"
	tunnel := `<span class="sf-toolbar-status sf-toolbar-status-red">Down</span>`
	docker := `<span class="sf-toolbar-status sf-toolbar-status-red">Down</span>`
	envVars := `<span class="sf-toolbar-status sf-toolbar-status-red">None</span>`
	if env["SYMFONY_TUNNEL"] != "" {
		tunnel = fmt.Sprintf(`<span class="sf-toolbar-status sf-toolbar-status-green">Up (%s)</span>`, env["SYMFONY_TUNNEL"])
		if env["SYMFONY_TUNNEL_ENV"] != "" {
			envVars = `<span class="sf-toolbar-status sf-toolbar-status-green">from Platform.sh</span>`
			logoBg = "sf-toolbar-status-green"
		} else {
			logoBg = "sf-toolbar-status-yellow"
		}
	}

	if env["SYMFONY_DOCKER_ENV"] == "1" {
		docker = `<span class="sf-toolbar-status sf-toolbar-status-green">Up</span>`
		logoBg = "sf-toolbar-status-green"
		if env["SYMFONY_TUNNEL_ENV"] == "" {
			envVars = `<span class="sf-toolbar-status sf-toolbar-status-green">from Docker</span>`
		}
	}

	webmail := `<span class="sf-toolbar-status sf-toolbar-status-red">Down</span>`
	rabbitmqui := `<span class="sf-toolbar-status sf-toolbar-status-red">Down</span>`
	blackfire := `<span class="sf-toolbar-status sf-toolbar-status-red">Down</span>`
	if env, err := envs.NewLocal(p.projectDir, terminal.IsDebug()); err == nil {
		values := envs.AsMap(env)
		if prefix := env.FindRelationshipPrefix("mailer", "http"); prefix != "" {
			if url, exists := values[prefix+"URL"]; exists {
				webmail = fmt.Sprintf(`<span class="sf-toolbar-status sf-toolbar-status-green">Up</span>&nbsp;&nbsp;&nbsp;<a class="sf-cli-webmail" href="%s" target="_blank">Open</a>`, url)
			}
		}
		if prefix := env.FindRelationshipPrefix("amqp", "http"); prefix != "" {
			if url, exists := values[prefix+"URL"]; exists {
				rabbitmqui = fmt.Sprintf(`<span class="sf-toolbar-status sf-toolbar-status-green">Up</span>&nbsp;&nbsp;&nbsp;<a class="sf-cli-rabbitmq" href="%s" target="_blank">Open</a>`, url)
			}
		}
		if prefix := env.FindRelationshipPrefix("blackfire", "tcp"); prefix != "" {
			blackfire = `<span class="sf-toolbar-status sf-toolbar-status-green">Up</span>&nbsp;&nbsp;&nbsp;<a class="sf-cli-blackfire" href="https://blackfire.io/" target="_blank">Open</a>`
		}
	}

	b, err := ioutil.ReadAll(body)
	if err != nil {
		return body, errors.WithStack(err)
	}
	content := []byte(`
<div class="sf-cli sf-toolbar-block sf-toolbar-block-sf-cli ` + logoBg + ` sf-toolbar-block-right">
	<div class="sf-toolbar-icon">
		<span class="sf-toolbar-label">
			<svg xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink" x="0px" y="0px" width="24" height="24" viewBox="0 0 24 24" enable-background="new 0 0 24 24" version="1.1">
				<path style="stroke:none;fill-rule:evenodd;fill:white;fill-opacity:1;" d="M 24 2.398438 C 24 1.074219 22.925781 0 21.601562 0 L 2.398438 0 C 1.074219 0 0 1.074219 0 2.398438 L 0 21.601562 C 0 22.925781 1.074219 24 2.398438 24 L 21.601562 24 C 22.925781 24 24 22.925781 24 21.601562 Z M 24 2.398438 "/>
				<path style="stroke:none;fill-rule:nonzero;fill:black;fill-opacity:1;" d="M 18.078125 3.109375 C 16.742188 3.15625 15.578125 3.894531 14.710938 4.910156 C 13.75 6.027344 13.109375 7.351562 12.648438 8.703125 C 11.824219 8.027344 11.191406 7.152344 9.867188 6.773438 C 8.847656 6.480469 7.773438 6.601562 6.785156 7.335938 C 6.320312 7.683594 5.996094 8.210938 5.84375 8.710938 C 5.449219 9.996094 6.261719 11.144531 6.628906 11.558594 L 7.4375 12.421875 C 7.605469 12.59375 8.007812 13.035156 7.808594 13.667969 C 7.597656 14.359375 6.765625 14.804688 5.914062 14.542969 C 5.53125 14.425781 4.984375 14.144531 5.105469 13.742188 C 5.15625 13.578125 5.273438 13.457031 5.335938 13.316406 C 5.394531 13.195312 5.421875 13.105469 5.441406 13.050781 C 5.597656 12.542969 5.382812 11.878906 4.835938 11.710938 C 4.328125 11.554688 3.808594 11.679688 3.605469 12.332031 C 3.378906 13.078125 3.734375 14.425781 5.640625 15.015625 C 7.875 15.703125 9.765625 14.484375 10.035156 12.898438 C 10.203125 11.90625 9.753906 11.167969 8.933594 10.21875 L 8.261719 9.476562 C 7.855469 9.070312 7.71875 8.378906 8.136719 7.847656 C 8.492188 7.402344 8.996094 7.210938 9.824219 7.433594 C 11.03125 7.761719 11.566406 8.597656 12.464844 9.273438 C 12.09375 10.492188 11.851562 11.710938 11.632812 12.804688 L 11.5 13.621094 C 10.855469 16.984375 10.367188 18.832031 9.09375 19.890625 C 8.839844 20.074219 8.472656 20.347656 7.917969 20.367188 C 7.628906 20.375 7.535156 20.175781 7.53125 20.089844 C 7.527344 19.886719 7.695312 19.792969 7.808594 19.703125 C 7.980469 19.609375 8.238281 19.457031 8.21875 18.960938 C 8.203125 18.378906 7.71875 17.875 7.023438 17.898438 C 6.5 17.917969 5.703125 18.40625 5.734375 19.308594 C 5.765625 20.238281 6.628906 20.933594 7.9375 20.890625 C 8.636719 20.867188 10.195312 20.582031 11.730469 18.753906 C 13.519531 16.660156 14.019531 14.261719 14.394531 12.503906 L 14.816406 10.183594 C 15.050781 10.210938 15.300781 10.230469 15.570312 10.238281 C 17.796875 10.285156 18.910156 9.132812 18.929688 8.292969 C 18.941406 7.785156 18.597656 7.285156 18.113281 7.296875 C 17.769531 7.304688 17.335938 7.535156 17.230469 8.011719 C 17.128906 8.480469 17.941406 8.902344 17.308594 9.3125 C 16.855469 9.605469 16.050781 9.808594 14.914062 9.644531 L 15.121094 8.5 C 15.542969 6.335938 16.0625 3.671875 18.035156 3.609375 C 18.179688 3.601562 18.703125 3.613281 18.71875 3.960938 C 18.722656 4.078125 18.691406 4.109375 18.554688 4.375 C 18.417969 4.582031 18.363281 4.757812 18.371094 4.960938 C 18.390625 5.511719 18.8125 5.875 19.417969 5.855469 C 20.234375 5.828125 20.46875 5.035156 20.453125 4.628906 C 20.421875 3.671875 19.410156 3.066406 18.078125 3.109375 Z M 18.078125 3.109375 "/>
			</svg>
		</span>
		<span class="sf-toolbar-value">Server</span>
	</div>
	<div class="sf-toolbar-info" style="left: 0px;">
		<div class="sf-toolbar-info-piece">
			<b>Server</b>` + p.Version.ServerTypeName() + ` ` + p.Version.Version + `
		</div>
		<div class="sf-toolbar-info-piece">
			<b>Tunnel</b>` + tunnel + `
		</div>
		<div class="sf-toolbar-info-piece">
			<b>Docker Compose</b>` + docker + `
		</div>
		<div class="sf-toolbar-info-piece">
			<b>Env Vars</b>` + envVars + `
		</div>
		<div class="sf-toolbar-info-piece">
			<b>RabbitMQ UI</b>` + rabbitmqui + `
		</div>
		<div class="sf-toolbar-info-piece">
			<b>Webmail</b>` + webmail + `
		</div>
		<div class="sf-toolbar-info-piece">
			<b>Blackfire.io Agent</b>` + blackfire + `
		</div>
	</div>
	<div></div>
</div>
$1`)

	re := regexp.MustCompile(`(<(?:a|button)[^"]+?class="hide-button")`)
	b = re.ReplaceAll(b, content)

	return ioutil.NopCloser(io.MultiReader(bytes.NewReader(bn), bytes.NewReader(start), bytes.NewReader(b))), nil
}

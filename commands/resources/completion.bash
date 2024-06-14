# Copyright (c) 2021-present Fabien Potencier <fabien@symfony.com>
#
# This file is part of Symfony CLI project
#
# This program is free software: you can redistribute it and/or modify
# it under the terms of the GNU Affero General Public License as
# published by the Free Software Foundation, either version 3 of the
# License, or (at your option) any later version.
#
# This program is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
# GNU Affero General Public License for more details.
#
# You should have received a copy of the GNU Affero General Public License
# along with this program. If not, see <http://www.gnu.org/licenses/>.
#
# Bash completions for the CLI binary
#
# References:
#   - https://github.com/symfony/symfony/blob/6.4/src/Symfony/Component/Console/Resources/completion.bash
#   - https://github.com/posener/complete/blob/master/install/bash.go
#   - https://github.com/scop/bash-completion/blob/master/completions/sudo
#

# this wrapper function allows us to let Symfony knows how to call the
# `bin/console` using the Symfony CLI binary (to ensure the right env and PHP
# versions are used)
_{{ .App.HelpName }}_console() {
  # shellcheck disable=SC2068
  {{ .CurrentBinaryInvocation }} console $@
}

_complete_{{ .App.HelpName }}() {

    # Use the default completion for shell redirect operators.
    for w in '>' '>>' '&>' '<'; do
        if [[ $w = "${COMP_WORDS[COMP_CWORD-1]}" ]]; then
            compopt -o filenames
            COMPREPLY=($(compgen -f -- "${COMP_WORDS[COMP_CWORD]}"))
            return 0
        fi
    done

    for (( i=1; i <= COMP_CWORD; i++ )); do
        if [[ "${COMP_WORDS[i]}" != -* ]]; then
            case "${COMP_WORDS[i]}" in
                console)
                    _SF_CMD="_{{ .App.HelpName }}_console" _command_offset $i
                    return
                    ;;
                composer{{range $name := (.App.Command "php").Names }}|{{$name}}{{end}}{{range $name := (.App.Command "run").Names }}|{{$name}}{{end}})
                    _command_offset $i
                    return
                    ;;
            esac;
        fi
    done

    IFS=$'\n' COMPREPLY=( $(COMP_LINE="${COMP_LINE}" COMP_POINT="${COMP_POINT}" COMP_DEBUG="$COMP_DEBUG" {{ .CurrentBinaryPath }} self:autocomplete) )
}

complete -F _complete_{{ .App.HelpName }} {{ .App.HelpName }}

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
                {{range $i, $name := (.App.Command "php").Names }}{{if $i}}|{{end}}{{$name}}{{end}}{{range $name := (.App.Command "run").Names }}|{{$name}}{{end}})
                    _command_offset $i
                    return
                    ;;
            esac;
        fi
    done

    # Use newline as only separator to allow space in completion values
    local IFS=$'\n'

    local cur prev words cword
    _get_comp_words_by_ref -n := cur prev words cword

    local sfcomplete
    if sfcomplete=$(COMP_LINE="${COMP_LINE}" COMP_POINT="${COMP_POINT}" COMP_DEBUG="$COMP_DEBUG" CURRENT="$cword" {{ .CurrentBinaryPath }} self:autocomplete 2>&1); then
        local quote suggestions
        quote=${cur:0:1}

        # Use single quotes by default if suggestions contains backslash (FQCN)
        if [ "$quote" == '' ] && [[ "$sfcomplete" =~ \\ ]]; then
            quote=\'
        fi

        if [ "$quote" == \' ]; then
            # single quotes: no additional escaping (does not accept ' in values)
            suggestions=$(for s in $sfcomplete; do printf $'%q%q%q\n' "$quote" "$s" "$quote"; done)
        elif [ "$quote" == \" ]; then
            # double quotes: double escaping for \ $ ` "
            suggestions=$(for s in $sfcomplete; do
                s=${s//\\/\\\\}
                s=${s//\$/\\\$}
                s=${s//\`/\\\`}
                s=${s//\"/\\\"}
                printf $'%q%q%q\n' "$quote" "$s" "$quote";
            done)
        else
            # no quotes: double escaping
            suggestions=$(for s in $sfcomplete; do printf $'%q\n' $(printf '%q' "$s"); done)
        fi
        COMPREPLY=($(IFS=$'\n' compgen -W "$suggestions" -- $(printf -- "%q" "$cur")))
        __ltrim_colon_completions "$cur"
    else
        if [[ "$sfcomplete" != *"Command \"_complete\" is not defined."* ]]; then
            >&2 echo
            >&2 echo $sfcomplete
        fi

        return 1
    fi
}

complete -F _complete_{{ .App.HelpName }} {{ .App.HelpName }}

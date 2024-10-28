_completion_devcontainer_vim(){
    local prev cur cword
    _get_comp_words_by_ref -n : cur prev cword

    local commands="run templates start stop down config vimrc runargs tool clean index self-update help"
    local subcommands_run=""
    local subcommands_templates="apply"
    local subcommands_tool="vim devcontainer clipboard-data-receiver"
    local subcommands_tool_vim="download"
    local subcommands_tool_devcontainer="download"
    local subcommands_tool_clipboard_data_receiver="download"
    local subcommands_index="update"

    if [[ ${cword} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "${commands}" -- "${cur}") )
    else
        case "${prev}" in
            run)
                COMPREPLY=( $(compgen -W "${subcommands_run}" -- "${cur}") )
                ;;
            templates)
                COMPREPLY=( $(compgen -W "${subcommands_templates}" -- "${cur}") )
                ;;
            tool)
                COMPREPLY=( $(compgen -W "${subcommands_tool}" -- "${cur}") )
                ;;
            vim)
                COMPREPLY=( $(compgen -W "${subcommands_tool_vim}" -- "${cur}") )
                ;;
            devcontainer)
                COMPREPLY=( $(compgen -W "${subcommands_tool_devcontainer}" -- "${cur}") )
                ;;
            clipboard-data-receiver)
                COMPREPLY=( $(compgen -W "${subcommands_tool_clipboard_data_receiver}" -- "${cur}") )
                ;;
            index)
                COMPREPLY=( $(compgen -W "${subcommands_index}" -- "${cur}") )
                ;;
            *)
                COMPREPLY=()
                ;;
        esac
    fi
} &&
complete -F _completion_devcontainer_vim devcontainer.vim bash-complete-func

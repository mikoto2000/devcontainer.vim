_completion_devcontainer_vim(){
    local prev cur cword
    _get_comp_words_by_ref -n : cur prev cword
    opts="run templates start stop down config vimrc runargs tool clean index self-update help"
    COMPREPLY=( $(compgen -W "${opts}" -- "${cur}") )
} &&
complete -F _completion_devcontainer_vim devcontainer.vim bash-complete-func


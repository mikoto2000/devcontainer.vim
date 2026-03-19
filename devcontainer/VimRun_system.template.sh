#!/bin/sh

{{ .TmuxCommand }} -u set-option -g status off \; new-session -s "devcontainer.vim" -A {{ .VimFileName }} --cmd "let g:devcontainer_vim = v:true" -S /{{ .SendToTcp }} -S /vimrc $*

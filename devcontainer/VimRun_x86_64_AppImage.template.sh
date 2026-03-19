#!/bin/sh

cd ~
/{{ .VimFileName }} --appimage-extract > /dev/null

cd -
{{ .TmuxCommand }} -u set-option -g status off \; new-session -s "devcontainer.vim" -A ~/squashfs-root/AppRun --cmd "let g:devcontainer_vim = v:true" -S /{{ .SendToTcp }} -S /vimrc $*

#!/bin/sh

cd ~
/{{ .VimFileName }} --appimage-extract > /dev/null

cd -
~/squashfs-root/AppRun --cmd "let g:devcontainer_vim = v:true" -S /{{ .SendToTcp }} -S /vimrc $*


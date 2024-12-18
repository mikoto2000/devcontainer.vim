#!/bin/sh

{{ .VimFileName }} --cmd "let g:devcontainer_vim = v:true" -S /{{ .SendToTcp }} -S /vimrc $*


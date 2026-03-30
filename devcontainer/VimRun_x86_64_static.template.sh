#!/bin/sh

cd /
tar zxf ./{{ .VimFileName }} -C ~/ > /dev/null

cd ~
chmod -R +w ~/vim-static
rm -rf ~/vim-static
mv $(ls -d ~/vim-*-x86_64) ~/vim-static
{{- if .UseTmux }}
{{ .TmuxCommand }} -u set-option -g prefix None \; unbind-key C-b \; set-option -g status off \; set-option -g set-clipboard on \; new-session -s "devcontainer.vim" -A ~/vim-static/AppRun --cmd "let g:devcontainer_vim = v:true" -S /{{ .SendToTcp }} -S /vimrc
{{- else }}
~/vim-static/AppRun --cmd "let g:devcontainer_vim = v:true" -S /{{ .SendToTcp }} -S /vimrc
{{- end }}

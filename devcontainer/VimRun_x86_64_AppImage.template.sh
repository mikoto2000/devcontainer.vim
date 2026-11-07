#!/bin/sh

cd ~
{{ shellQuote (print "/" .VimFileName) }} --appimage-extract > /dev/null

cd -
{{- if .UseTmux }}
{{ shellQuote .TmuxCommand }} -u set-option -g prefix None \; unbind-key C-b \; set-option -g status off \; set-option -g set-clipboard on \; new-session -s "devcontainer.vim" -A ~/squashfs-root/AppRun --cmd "let g:devcontainer_vim = v:true" -S {{ shellQuote (print "/" .SendToTcp) }} -S /vimrc $*
{{- else }}
~/squashfs-root/AppRun --cmd "let g:devcontainer_vim = v:true" -S {{ shellQuote (print "/" .SendToTcp) }} -S /vimrc $*
{{- end }}

#!/bin/sh

{{- if .UseTmux }}
{{ shellQuote .TmuxCommand }} -u set-option -g prefix None \; unbind-key C-b \; set-option -g status off \; set-option -g set-clipboard on \; new-session -s "devcontainer.vim" -A {{ shellQuote .VimFileName }} --cmd "let g:devcontainer_vim = v:true" -S {{ shellQuote (print "/" .SendToTcp) }} -S /vimrc $*
{{- else }}
{{ shellQuote .VimFileName }} --cmd "let g:devcontainer_vim = v:true" -S {{ shellQuote (print "/" .SendToTcp) }} -S /vimrc $*
{{- end }}

#!/bin/sh

{{- if .UseTmux }}
{{ .TmuxCommand }} -u set-option -g prefix None \; unbind-key C-b \; set-option -g status off \; set-option -g set-clipboard on \; new-session -s "devcontainer.vim" -A {{ .VimFileName }} --cmd "let g:devcontainer_vim = v:true" -S /{{ .SendToTcp }} -S /vimrc $*
{{- else }}
{{ .VimFileName }} --cmd "let g:devcontainer_vim = v:true" -S /{{ .SendToTcp }} -S /vimrc $*
{{- end }}

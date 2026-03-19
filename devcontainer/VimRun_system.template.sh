#!/bin/sh
set -eu

if [ "${1:-}" = "--inside-tmux" ]; then
  shift
  exec {{ .VimFileName }} --cmd "let g:devcontainer_vim = v:true" -S /{{ .SendToTcp }} -S /vimrc "$@"
fi

if [ -n "${TMUX:-}" ]; then
  exec {{ .VimFileName }} --cmd "let g:devcontainer_vim = v:true" -S /{{ .SendToTcp }} -S /vimrc "$@"
fi

tmux_args=""
for arg in "$@"; do
  escaped_arg=$(printf "%s" "$arg" | sed "s/'/'\\\\''/g")
  tmux_args="${tmux_args} '${escaped_arg}'"
done

exec {{ .TmuxCommand }} new-session -A -s devcontainer.vim "/VimRun.sh --inside-tmux${tmux_args}"

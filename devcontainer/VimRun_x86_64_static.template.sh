#!/bin/sh
set -eu

run_vim() {
  cd /
  tar zxf ./{{ .VimFileName }} -C ~/ > /dev/null

  cd ~
  chmod -R +w ~/vim-static
  rm -rf ~/vim-static
  mv $(ls -d ~/vim-*-x86_64) ~/vim-static
  exec ~/vim-static/AppRun --cmd "let g:devcontainer_vim = v:true" -S /{{ .SendToTcp }} -S /vimrc "$@"
}

if [ "${1:-}" = "--inside-tmux" ]; then
  shift
  run_vim "$@"
fi

if [ -n "${TMUX:-}" ]; then
  run_vim "$@"
fi

tmux_args=""
for arg in "$@"; do
  escaped_arg=$(printf "%s" "$arg" | sed "s/'/'\\\\''/g")
  tmux_args="${tmux_args} '${escaped_arg}'"
done

exec {{ .TmuxCommand }} new-session -A -s devcontainer.vim "/VimRun.sh --inside-tmux${tmux_args}"

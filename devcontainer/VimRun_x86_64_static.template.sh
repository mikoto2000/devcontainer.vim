#!/bin/sh

cd /
tar zxf ./{{ .VimFileName }} -C ~/ > /dev/null

cd ~
chmod -R +w ~/vim-static
rm -rf ~/vim-static
mv $(ls -d ~/vim-*-x86_64) ~/vim-static
~/vim-static/AppRun --cmd "let g:devcontainer_vim = v:true" -S /{{ .SendToTcp }} -S /vimrc


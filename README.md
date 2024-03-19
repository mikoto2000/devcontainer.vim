# devcontainer.vim

コンテナ上で Vim を使った開発をするためのツール。

# Usage:

```sh
# 以下 docker コマンド相当の環境でコンテナを立ち上げる場合
# docker run -it --rm -v "$(pwd):/work" --workdir /work -v "$HOME/.vim:/root/.vim" --name golang golang:1.22.1-bookworm
# ※ 現段階では devcontainer.json 対応をしていないので、諸々を明示的に指定する必要がある
devcontainer.vim -v "$(pwd):/work" --workdir /work -v "$HOME/.vim:/root/.vim" --name golang golang:1.22.1-bookworm
```


# Requirements:

以下コマンドがインストール済みで、PATH が通っていること。

- docker


# Install:

## binary download

TODO

## go install

```sh
go install github.com/mikoto2000/devcontainer.vim@latest
```


# Uninstall:

## Windows

Delete executable file, config directory(`~/AppData/Roaming/devcontainer.vim`), and cache directory(`~/AppData/Local/devcontainer.vim`).

```sh
Remove-Item PATH_TO/devcontainer.vim.exe
Remove-Item -Recurse ~/AppData/Roaming/devcontainer.vim
Remove-Item -Recurse ~/AppData/Local/devcontainer.vim
```

## Linux

Delete executable file, config directory(`~/.config/devcontainer.vim`), and cache directory(`~/.cache/devcontainer.vim`).

```sh
rm PATH_TO/devcontainer.vim
rm -rf ~/.config/devcontainer.vim
rm -rf ~/.cache/devcontainer.vim
```

## MacOS

TODO:


# TODO:

- [x] : v0.1.0
    - [x] : docker run 対応
        - [x] : コンテナの起動
        - [x] : AppImage 版 Vim のダウンロードとコンテナへの転送
    - [ ] : `devcontainer.vim` への引数と `docker` への引数を指定できるようにする
        - `--` より前を `devcontainer.vim` への引数とする
    - [ ] : リリーススクリプト・リリースワークフローを作る
- [ ] : v0.2.0
    - [ ] : クリップボード転送機能追加
        1. TCP でテキストを待ち受け、受信したテキストをクリップボードへ反映するプログラムを作る
        2. TCP ソケット通信する関数、ヤンク処理時にテキスト送信をするマッピングを実装したスクリプトを作る
            - `docker cp` で `/SendToTcp.vim` にコピーし、 `-c "source /SendToTcp.vim` する
        3. `devcontainer.vim` 起動時に「1.」のプログラムを実行
            - 多重起動防止のために既にプログラムが実行済みかどうかを確認する必要がある
            - 終了時にも、「他の `devcontainer.vim` が存在するか」を確認して終了させるか判定
- [ ] : v0.3.0
    - [ ] : `devcontainer.json` の `nonComposeBase` 対応
        - どこまで対応するかは未検討...
- [ ] : v0.4.0
    - [ ] : `devcontainer.json` の `composeContainer` 対応
        - どこまで対応するかは未検討...
- [ ] : v0.5.0
    - [ ] : キャッシュクリアコマンド
    - [ ] : アンインストールコマンド
    - [ ] : Vim アップデートコマンド


# License:

Copyright (C) 2024 mikoto2000

This software is released under the MIT License, see LICENSE

このソフトウェアは MIT ライセンスの下で公開されています。 LICENSE を参照してください。


# Author:

mikoto2000 <mikoto2000@gmail.com>



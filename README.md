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


# TODO:

- [ ] : v0.1.0
    - [ ] : docker run 対応
        - [ ] : コンテナの起動
        - [ ] : AppImage 版 Vim のダウンロードとコンテナへの転送
- [ ] : v0.2.0
    - [ ] : `devcontainer.json` 対応
        - どこまで対応するかは未検討...
        - [ ] : docker compose 対応はしたい


# License:

Copyright (C) 2024 mikoto2000

This software is released under the MIT License, see LICENSE

このソフトウェアは MIT ライセンスの下で公開されています。 LICENSE を参照してください。


# Author:

mikoto2000 <mikoto2000@gmail.com>



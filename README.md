# devcontainer.vim

コンテナ上で Vim を使った開発をするためのツール。

## Usage:

### `devcontainer.json` が存在しないプロジェクトで、ワンショットで環境を立ち上げる

```sh
devcontainer.vim run [DOCKER_OPTIONS] [DOCKER_ARGS]
```

`docker run -it --rm -v "$(pwd):/work" --workdir /work -v "$HOME/.vim:/root/.vim" --name golang golang:1.22.1-bookworm` コマンド相当の環境でコンテナを立ち上げる場合の例:

```sh
devcontainer.vim run -v "$(pwd):/work" --workdir /work -v "$HOME/.vim:/root/.vim" --name golang golang:1.22.1-bookworm
```

### `devcontainer.json` が存在する場合

カレントディレクトリから `devcontainer.json` を検索し、読み込み、環境を立ち上げ、Vim を転送し、起動する。

```sh
devcontainer.vim start .
```

`devcontainer` への引数を(`--workspace-folder` 以外は) そのまま利用できるため、
`.vim` をバインドしたい場合、以下のように指定する。

```sh
devcontainer.vim start --mount "type=bind,source=$HOME/.vim,target=/root/.vim" ./
```


## Requirements:

以下コマンドがインストール済みで、PATH が通っていること。

- docker


## Features:

TODO


## Install:

### binary download

[Latest version](https://github.com/mikoto2000/devcontainer.vim/releases/latest)


### go install

```sh
go install github.com/mikoto2000/devcontainer.vim@latest
```


## Uninstall:

### Windows

Delete executable file, config directory(`~/AppData/Roaming/devcontainer.vim`), and cache directory(`~/AppData/Local/devcontainer.vim`).

```sh
Remove-Item PATH_TO/devcontainer.vim.exe
Remove-Item -Recurse ~/AppData/Roaming/devcontainer.vim
Remove-Item -Recurse ~/AppData/Local/devcontainer.vim
```

### Linux

Delete executable file, config directory(`~/.config/devcontainer.vim`), and cache directory(`~/.cache/devcontainer.vim`).

```sh
rm PATH_TO/devcontainer.vim
rm -rf ~/.config/devcontainer.vim
rm -rf ~/.cache/devcontainer.vim
```

### MacOS

TODO:


## TODO:

- [x] : v0.1.0
    - [x] : docker run 対応
        - [x] : コンテナの起動
        - [x] : AppImage 版 Vim のダウンロードとコンテナへの転送
    - [x] : `devcontainer.vim` への引数と `docker` への引数を指定できるようにする
        - [x] : `run` コマンドとして実現
- [x] : v0.2.0
    - [x] : `devcontainer.json` の `composeContainer` 対応
        - [x] : dockerComposeFile
        - [x] : service
        - [x] : workspaceFolder
        - [x] : remoteUser
- [x] : v0.3.0
    - [x] : `devcontainer.json` の `nonComposeBase` 対応
- [ ] : v0.4.0
    - [ ] : down コマンドの実装
        - [ ] : `composeContainer` と `nonComposeBase` の判定
            - [ ] : `devcontainer read-configuration` の結果に `dockerComposeFile` が含まれているかで判定
        - [ ] : `composeContainer` の場合
            - `docker compose ps --format json` して `Project` の値を取得し、 `docker compose -p ${PROJECT_NAME} down` する
        - [ ] : `nonComposeBase` の場合
            - `docker ps --format json` して `Labels` 内に `devcontainer.local_folder=xxx` が含まれており、 `xxx` が現在のディレクトリと一致するものを探し、そいつの ID で `docker rm -f ${CONTAINER_ID}` する
- [ ] : v0.5.0
    - [ ] : 暗黙の docker option を追加できるようにする
        - [ ] : 設定ファイルに暗黙のオプション設定を追加し、 `run` サブコマンド実行時にそれを読み込む
        - [ ] : 設定ファイルを開くサブコマンドを追加
            - [ ] : 関連付けられているファイルで開けるなら開く、そうでなければパスを表示
    - [ ] : リリーススクリプト・リリースワークフローを作る
- [ ] : v0.6.0
    - [ ] : クリップボード転送機能追加
        1. TCP でテキストを待ち受け、受信したテキストをクリップボードへ反映するプログラムを作る
        2. TCP ソケット通信する関数、ヤンク処理時にテキスト送信をするマッピングを実装したスクリプトを作る
            - `docker cp` で `/SendToTcp.vim` にコピーし、 `-c "source /SendToTcp.vim` する
        3. `devcontainer.vim` 起動時に「1.」のプログラムを実行
            - 多重起動防止のために既にプログラムが実行済みかどうかを確認する必要がある
            - 終了時にも、「他の `devcontainer.vim` が存在するか」を確認して終了させるか判定
- [ ] : v0.7.0
    - [ ] : キャッシュクリアコマンド
    - [ ] : アンインストールコマンド
    - [ ] : Vim アップデートコマンド


## License:

Copyright (C) 2024 mikoto2000

This software is released under the MIT License, see LICENSE

このソフトウェアは MIT ライセンスの下で公開されています。 LICENSE を参照してください。


## Author:

mikoto2000 <mikoto2000@gmail.com>



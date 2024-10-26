# devcontainer.vim

コンテナ上で Vim を使った開発をするためのツール。 (VSCode Dev Container の Vim 版)

VSCode 向けに作成された `devcontainer.json` に追加する形で Vim による Dev Container 開発のための設定を追加・起動するツールです。

## Getting Started

- [devcontainer.vim で、コンテナ上の Vim に引きこもって作業を行う(ゼロから環境構築をしてみよう編) - mikoto2000 の日記](https://mikoto2000.blogspot.com/2024/10/devcontainervim-vim.html)
- [？「えっ！1分でGo言語の環境構築を！？」 devcontainer.vim「できらぁ！」 - YouTube](https://www.youtube.com/shorts/v0h6AfRIyvs)


## Usage:

```
NAME:
   devcontainer.vim - devcontainer for vim.

USAGE:
   devcontainer.vim [global options] command [command options] 

VERSION:
   1.0.11

COMMANDS:
   run          Run container use `docker run`
   templates    Run `devcontainer templates`
   start        Run `devcontainer up` and `devcontainer exec`
   stop         Stop devcontainers.
   down         Stop and remove devcontainers.
   config       devcontainer.vim's config information.
   vimrc        devcontainer.vim's vimrc information.
   runargs      run subcommand's default arguments.
   tool         Management tools
   clean        clean workspace cache files.
   index        Management index file
   self-update  Update devcontainer.vim itself
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --license, -l  show licensesa.
   --help, -h     show help
   --version, -v  print the version
```

### `devcontainer.json` が存在しないプロジェクトで、ワンショットで環境を立ち上げる

```sh
devcontainer.vim run [DOCKER_OPTIONS] [DOCKER_ARGS]
```

`docker run -it --rm -v "$(pwd):/work" --workdir /work -v "$HOME/.vim:/root/.vim" --name golang golang:1.22.1-bookworm` コマンド相当の環境でコンテナを立ち上げる場合の例:

```sh
devcontainer.vim run -v "$(pwd):/work" --workdir /work -v "$HOME/.vim:/root/.vim" --name golang golang:1.22.1-bookworm
```

### `devcontainer.json` が存在する場合

#### 環境の起動

`start` サブコマンドで、環境を立ち上げ、Vim を転送し、起動できる。

たとえば、カレントディレクトリから `devcontainer.json` を検索し、読み込み、環境を立ち上げ、Vim を転送し、起動する場合は以下。

```sh
devcontainer.vim start .
```

`devcontainer` への引数を(`--workspace-folder` 以外は) そのまま利用できるため、
`.vim` をバインドしたい場合、以下のように指定する。

```sh
devcontainer.vim start --mount "type=bind,source=$HOME/.vim,target=/root/.vim" .
```

#### 環境の停止

`stop` サブコマンドで環境の停止ができる。

```sh
devcontainer.vim stop .
```

再開したい場合は、もう一度 `start` サブコマンドを実行する。


#### 環境の削除

`down` サブコマンドで環境の削除ができる。

```sh
devcontainer.vim down .
```


#### ツールのアップデート

`devcontainer.vim` が内部で利用するツールをアップデートしたい場合には、 `tool` サブコマンドを使用する。

```sh
# Vim のアップデート
devcontainer.vim tool vim download

# devcontainer CLI のアップデート
devcontainer.vim tool devcontainer download
```

#### devcontainer.vim 自身のアップデート

`self-update` サブコマンドを使用して、 `devcontainer.vim` 自身を最新バージョンに更新できます。

```sh
devcontainer.vim self-update
```

### テンプレートをもとに `devcontainer.json` を作成する

`devcontainer.vim templates apply` サブコマンドを使用することで、 devcontainers が提供しているテンプレートから `devcontainer.json` を生成できる。

`Go` のテンプレートを用いて `devcontainer.json` を生成する場合の例:

```sh
$ devcontainer.vim templates apply .
Search: Go
? Select Template: 
  ▸ Go
    Go & PostgreSQL
    Node.js & Mongo DB
    Hugo & pnpm
```

`devcontainer.vim templates apply` を実行すると、テンプレート名の一覧が表示されるので、
キー入力で名前をインクリメンタル検索し、上下キーでカーソルを移動・エンターキーでテンプレートを決定できる。


## Customize:

### コンテナのカスタマイズ

`.vim` や `vimfiles` など、ホストからバインドマンとさせたいものがあるが、
VSCode 等の他ツール向けに作成した `devcontainer.json` に devcontainer.vim 専用の `mounts` 定義を付けることはしたくない。

そのため、別途 devcontainer.vim のみが読み込むファイルを `.devcontainer/devcontainer.vim.json` に配置する。
devcontainer.vim は、 `.devcontainer/devcontainer.json` と `.devcontainer/devcontainer.vim.json` をマージして実行する。

```
PROJECT_ROOT/
    +- .devcontainer/
    |   +- devcontainer.json      # 普通の devcontainer 向けの設定を記述
    |   +- devcontainer.vim.json  # .vim のマウントなど、 devcontainer.vim のみで利用したい設定を記述
    |
    +- ...(other project files)
```

`devcontainer.json`:

```json
{
  "name":"Go",
  "image":"mcr.microsoft.com/devcontainers/go:1-1.22-bookworm",
  "features":{},
  "remoteUser":"vscode"
}
```

`devcontainer.vim.json`:

```json
{
  "mounts": [
    {
      "type": "bind",
      "source": "${localEnv:HOME}/.vim",
      "target": "/home/vscode/.vim"
    }
  ]
}
```

#### 追加の設定を生成する

`devcontainer.vim config -g` で `devcontainer.vim` が使用するための追加設定ファイルのテンプレートを生成できる。

```sh
devcontainer.vim config -g --home /home/containerUser > .devcontainer/devcontainer.vim.json
```

使用できるオプションは以下:

- `-g` : 設定生成フラグ
- `-o` : 生成した設定の出力先ファイルを指定(default: STDOUT)
- `--home` : 設定テンプレート内のホームディレクトリのパス

### Vim のカスタマイズ

`devcontainer.vim vimrc -o` で、コンテナ上で実行する Vim に、追加で読み込ませるスクリプトが開きます。

このスクリプトを更新することで、コンテナ上の Vim のみに適用させたい設定ができます。

デフォルトでは、以下の内容になっています。
(ノーマルモードで `"*yy`, ヴィジュアルモードで `"*y` でホストへ `"` レジスタの内容を送信する)
好みに応じて修正してください。

```vimrc
nnoremap <silent> "*yy yy:call SendToCdr('"')<CR>
vnoremap <silent> "*y y:call SendToCdr('"')<CR>
```

また、デフォルトに戻したい場合には、 `-g` オプションで vimrc を再生成してください。

```sh
devcontainer.vim vimrc -g
```


### run サブコマンドの引数のカスタマイズ

`devcontainer.vim runargs -o` で、 run サブコマンドへ暗黙的に設定される引数設定ファイルが開きます。

このファイルを更新することで、暗黙的に適用させたい引数が指定できます。

デフォルトでは、以下の内容になっています。
(カレントディレクトリを `/work` へマウントし、ワーキングディレクトリも同じ場所へ設定)
好みに応じて修正してください。

```
-v "$(pwd):/work" -v "$HOME/.vim:/root/.vim" --workdir /work
```

また、デフォルトに戻したい場合には、 `-g` オプションで runargs を再生成してください。

```sh
devcontainer.vim runargs -g
```


## Requirements:

以下コマンドがインストール済みで、PATH が通っていること。

- docker


## Features:

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
- [x] : v0.4.0
    - [x] : down コマンドの実装
        - [x] : `composeContainer` と `nonComposeBase` の判定
            - [x] : `devcontainer read-configuration` の結果に `dockerComposeFile` が含まれているかで判定
        - [x] : `composeContainer` の場合
            - `docker compose ps --format json` して `Project` の値を取得し、 `docker compose -p ${PROJECT_NAME} down` する
        - [x] : `nonComposeBase` の場合
            - `docker ps --format json` して `Labels` 内に `devcontainer.local_folder=xxx` が含まれており、 `xxx` が現在のディレクトリと一致するものを探し、そいつの ID で `docker rm -f ${CONTAINER_ID}` する
- [x] : v0.5.0
    - [x] : devcontainer.vim のみが利用する設定に関する仕組みを追加
        - [x] : `devcontainer.json` と `devcontainer.vim.json` をマージしてからコンテナを起動する
        - [x] : キャッシュディレクトリ内の構造整理
    - [x] : `devcontainer.vim.json` の設定例出力機能
        - [x] : 標準出力
        - [x] : ファイルパス指定( `-o` オプション)
    - [x] : Windows 向けに環境変数をセット
        - `USERPROFILE` -> `HOME`
    - [x] : config コマンドの実装
    - [x] : リリーススクリプト・リリースワークフローを作る
- [x] : v0.6.0
    - [x] : `devcontainer up` の出力を表示する
    - [x] : `devcontainer templates apply` コマンドを使えるようにする
- [x] : v0.7.0
    - [x] : Vim アップデートコマンドを追加
- [x] : v0.8.0
    - [x] : クリップボード転送機能追加
        1. TCP でテキストを待ち受け、受信したテキストをクリップボードへ反映するプログラムを作る
        2. TCP ソケット通信する関数、ヤンク処理時にテキスト送信をするマッピングを実装したスクリプトを作る
            - `docker cp` で `/SendToTcp.vim` にコピーし、 `-c "source /SendToTcp.vim` する
        3. `devcontainer.vim` 起動時に「1.」のプログラムを実行
            - 多重起動防止のために既にプログラムが実行済みかどうかを確認する必要がある
            - 終了時にも、「他の `devcontainer.vim` が存在するか」を確認して終了させるか判定
- [x] : v0.9.0
    - [x] : run サブコマンドのデフォルト引数を自分で指定できるようにする
        - [x] : `<os.UserConfigDir>/devcontainer.vim/runargs` にデフォルトで付与したい引数を記載する
        - ※ sh にパスの通った Linux のみで有効。(Windows PowerShell でシェル変数の展開が上手くできないため)
- [x] : v0.10.0
    - [x] : キャッシュクリアコマンド
    - [x] : stop コマンドの実装
- [x] : v0.11.0
    - [x] : テンプレートリスト出力機能
        - `devcontainer.vim templates apply` に渡す `--template-id` として使える ID の一覧を出力


## Limitation:

- amd64 のコンテナしか使用できません
- alpine 系のコンテナでは使用できません


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


## License:

Copyright (C) 2024 mikoto2000

This software is released under the MIT License, see LICENSE

このソフトウェアは MIT ライセンスの下で公開されています。 LICENSE を参照してください。


## Author:

mikoto2000 <mikoto2000@gmail.com>



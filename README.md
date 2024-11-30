**日本語** / [English](README_en.md)

# devcontainer.vim

コンテナ上で Vim を使った開発をするためのツール。 (VSCode Dev Container の Vim 版)

VSCode 向けに作成された `devcontainer.json` に追加する形で Vim による Dev Container 開発のための設定を追加・起動するツールです。

## Getting Started

- [devcontainer.vim で、コンテナ上の Vim に引きこもって作業を行う(ゼロから環境構築をしてみよう編) - mikoto2000 の日記](https://mikoto2000.blogspot.com/2024/10/devcontainervim-vim.html)
- [？「えっ！1分でGo言語の環境構築を！？」 devcontainer.vim「できらぁ！」 - YouTube](https://www.youtube.com/shorts/v0h6AfRIyvs)


## Features:

- 開発用コンテナを立ち上げ、そこに Vim を転送し、起動する
    - `devcontainre.json` が無いプロジェクトで、ワンショットで開発用コンテナを立ち上げる
        - docker に渡す引数のカスタマイズができる
    - `devcontainer.json` が無いプロジェクトに、`devcontainer.json` のテンプレートを追加できる
    - `devcontainer.json` があるプロジェクトで、開発用コンテナを開始・停止・削除できる
    - `devcontainer.json` とは別に、 `devcontainer.vim.json` を記述することで
      開発用コンテナに `devcontainer.vim` 用の設定を追加できる
    - 開発用コンテナで起動する Vim に追加で設定する vimrc を定義できる
- 開発用コンテナ上の Vim でヤンクした文字列を、ホスト PC のクリップボードへ貼り付けられる
- 開発用コンテナ内で使用するしたいツールを、開発用コンテナに転送して使用可能にする
- `vim`, `devcontainer`, `clipboard-data-receiver` など、使用するツールのアップデートができる
- セルフアップデートができる


## Requirements:

以下コマンドがインストール済みで、PATH が通っていること。

- docker


## Usage:

```
NAME:
   devcontainer.vim - devcontainer for vim.

USAGE:
   devcontainer.vim [global options] command [command options] 

VERSION:
   1.3.1

COMMANDS:
   run                 Run container use `docker run`
   templates           Run `devcontainer templates`
   start               Run `devcontainer up` and `devcontainer exec`
   stop                Stop devcontainers.
   down                Stop and remove devcontainers.
   config              devcontainer.vim's config information.
   vimrc               devcontainer.vim's vimrc information.
   runargs             run subcommand's default arguments.
   tool                Management tools
   clean               clean workspace cache files.
   index               Management dev container templates index file
   self-update         Update devcontainer.vim itself
   bash-complete-func  Show bash complete func
   help, h             Shows a list of commands or help for one command

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

`devcontainer.vim templates apply` を実行すると、テンプレート名の一覧が表示されるので、キー入力で名前をインクリメンタル検索し、上下キーでカーソルを移動・エンターキーでテンプレートを決定できる。


## サブコマンドの補完

`devcontainer.vim` にパスを通し、 `.bashrc` などに以下コードを追加することで、サブコマンドの補完が有効になります。

```sh
eval "$(devcontainer.vim bash-complete-func)"
```


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

#### 追加のランタイムをコンテナへインストールする

[denops.vim](https://github.com/vim-denops/denops.vim) や [coc.nvim](https://github.com/neoclide/coc.nvim) など 別途ランタイムが必要なプラグインを使用している場合、 `devcontainer.vim.json` の `features` にイメージ ID を追加することで、コンテナへランタイムをインストールできる。

deno をコンテナにインストールする例:

```json
...(snip)
  "features": {
    "ghcr.io/devcontainers-community/features/deno:1": {}
  }
...(snip)
```

`features` に指定できるイメージは [Available Dev Container Features](https://containers.dev/features) で確認できる。


### Vim のカスタマイズ

#### devcontainer.vim 上の Vim にのみ読み込まれる vimrc の作成

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

#### devcontainer.vim 上の Vim でのみ実行したい Vim script

devcontainer.vim 上で動く Vim には、 `g:devcontainer_vim` 変数が `v:true` で定義される。
以下のように判定すれば、「devcontainer.vim 上で動く Vim のみで実行される Vim script」が記述できる。

```vim
if get(g:, "devcontainer_vim", v:false)
  " Run only on devcontainer.vim
endif
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


## Limitation:

- amd64 のコンテナしか使用できません
- alpine 系のコンテナでは使用できません
- arm 系 macOS では動作しません


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



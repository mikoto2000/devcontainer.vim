[日本語](README.md) / **English**

# devcontainer.vim

A tool for developing with Vim on a container. (Vim version of VSCode Dev Container)

It's a tool that adds and starts settings for Vim-based Dev Container development in the form of additions to the `devcontainer.json` file created for VSCode.


## Getting Started

- [devcontainer.vim で、コンテナ上の Vim に引きこもって作業を行う(ゼロから環境構築をしてみよう編) - mikoto2000 の日記](https://mikoto2000.blogspot.com/2024/10/devcontainervim-vim.html)
- [？「えっ！1分でGo言語の環境構築を！？」 devcontainer.vim「できらぁ！」 - YouTube](https://www.youtube.com/shorts/v0h6AfRIyvs)


## Features:

- Set up a development container, transfer Vim/NeoVim to it, and start it.
    - For projects without `devcontainre.json`, launch a development container in a single shot
        - You can customize the arguments passed to docker.
    - Add a template for `devcontainer.json` to projects that don't have a `devcontainer.json` file
    - In a project with a `devcontainer.json` file, you can start, stop, and delete the development container.
    - In addition to `devcontainer.json`, you can add settings for `devcontainer.vim`
      to the development container by specifying `devcontainer.vim.json`.
    - You can define a vimrc to be used with Vim launched in a development container.
    - If the path to Vim/NeoVim is set in the development container, use it.
- The text copied in Vim on the development container can be pasted to the clipboard of the host PC.
- Transfer tools to be used in the development container to make them usable in the development container.
- Tools such as `vim`, `devcontainer`, and `clipboard-data-receiver` can be updated.
- Self-update capability


## Requirements:

The following commands are installed and in the PATH.

- docker

The following commands must exist in the container and be in the PATH.

- which


For ARM, the `tar` command must be present in the container.


## Usage:

```
NAME:
   devcontainer.vim - devcontainer for vim.

USAGE:
   devcontainer.vim [global options] command [command options] 

VERSION:
   3.5.0

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
   index               Management dev container template index file
   self-update         Update devcontainer.vim itself
   bash-complete-func  Show bash complete func
   help, h             Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --license, -l  show licensesa.
   --nvim         use NeoVim.
   --shell value  start with shell.
   --help, -h     show help
   --version, -v  print the version
```

### Set up an environment in a project where `devcontainer.json` does not exist.

```sh
devcontainer.vim run [DOCKER_OPTIONS] [DOCKER_ARGS]
```
Example of running a container in an environment equivalent to the `docker run -it --rm -v "$(pwd):/work" --workdir /work -v "$HOME/.vim:/root/.vim" --name golang golang:1.22.1-bookworm` command:

```sh
devcontainer.vim run -v "$(pwd):/work" --workdir /work -v "$HOME/.vim:/root/.vim" --name golang golang:1.22.1-bookworm
```


### if `devcontainer.json` exists

#### Environmental start

The `start` subcommand sets up the environment, transfers Vim, and starts it.

For example, if you want to search for `devcontainer.json` in the current directory, read it, set up the environment, transfer Vim, and start it, you can do the following.

```sh
devcontainer.vim start .
```

You can use arguments to `devcontainer` (except for `--workspace-folder`) as they are, so if you want to bind `.vim`, specify it as follows.

```sh
devcontainer.vim start --mount "type=bind,source=$HOME/.vim,target=/root/.vim" .
```

#### Environmental stop

The `stop` subcommand allows you to stop the environment.

```sh
devcontainer.vim stop .
```

To resume, execute the `start` subcommand again.


#### Environment deletion

The `down` subcommand can delete the environment.

```sh
devcontainer.vim down .
```


#### Tool update

To update the tools used internally by `devcontainer.vim`, use the `tool` subcommand.


```sh
# update Vim
devcontainer.vim tool vim download

# update devcontainer CLI 
devcontainer.vim tool devcontainer download
```

#### self update

You can update `devcontainer.vim` to the latest version using the `self-update` subcommand.

```sh
devcontainer.vim self-update
```

### Create `devcontainer.json` based on the template

The `devcontainer.vim templates apply` subcommand allows you to generate a `devcontainer.json` file from templates provided by devcontainers.

Example of generating `devcontainer.json` using the `Go` template:

```sh
$ devcontainer.vim templates apply .
Search: Go
? Select Template: 
  ▸ Go
    Go & PostgreSQL
    Node.js & Mongo DB
    Hugo & pnpm
```

Running `devcontainer.vim templates apply` will show a list of template names.
You can use the key input to incrementally search for names, move the cursor with the up and down keys, and select a template with the Enter key.


## Subcommand Completion

Passing the path to `devcontainer.vim` and adding the following code to `.bashrc` or similar will enable subcommand completion.

```sh
eval "$(devcontainer.vim bash-complete-func)"
```

## Customize:

### Container Customization

I want to bind some things like `.vim` and `vimfiles` from the host to the bindman,
but I don't want to add a `mounts` definition specific to `devcontainer.vim` to the `devcontainer.json` that I created for other tools like VSCode.

Therefore, the file that is read only by devcontainer.vim is placed in `.devcontainer/devcontainer.vim.json`.
`devcontainer.vim` merges and executes `.devcontainer/devcontainer.json` and `.devcontainer/devcontainer.vim.json`.

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


#### Generate additional settings

`devcontainer.vim config -g` generates a template for additional configuration files used by `devcontainer.vim`.

```sh
devcontainer.vim config -g --home /home/containerUser > .devcontainer/devcontainer.vim.json
```

Available options are as follows:

- `-g` : setting generation flag
- `-o` : Specify the output file for the generated configuration (default: STDOUT)
- `--home`: Path to the home directory in the configuration template


#### Install additional runtime in the container

If you use plugins that require a separate runtime, such as
[denops.vim](https://github.com/vim-denops/denops.vim) or
[coc.nvim](https://github.com/neoclide/coc.nvim),
you can install the runtime in the container by adding the image ID to the `features` section of `devcontainer.vim.json`.

Example of installing deno in a container:

```json
...(snip)
  "features": {
    "ghcr.io/devcontainers-community/features/deno:1": {}
  }
...(snip)
```

The images that can be specified in `features` can be checked in
[Available Dev Container Features](https://containers.dev/features).


### Vim Customization

#### Creating a vimrc that is only loaded for Vim on devcontainer.vim

`devcontainer.vim vimrc -o` opens a script that will be additionally loaded into Vim running in the container.

Updating this script allows you to apply settings only to Vim on the container.

Send the contents of the `"` register to the host in normal mode with `"*yy`, and in visual mode with `"*y`.
Adjust as desired.

The default is as follows:

```vimrc
if !has("nvim")
  nnoremap <silent> "*yy yy:call SendToCdr('"')<CR>
  vnoremap <silent> "*y y:call SendToCdr('"')<CR>
else
  nnoremap <silent> "*yy yy:lua SendToCdr('"')<CR>
  vnoremap <silent> "*y y:lua SendToCdr('"')<CR>
endif
```

To revert to the default, regenerate vimrc with the `-g` option.

```sh
devcontainer.vim vimrc -g
```

#### Vim script that should only be executed in Vim on devcontainer.vim

When Vim runs on devcontainer.vim, the `g:devcontainer_vim` variable is defined as `v:true`.

If you judge it like this, you can describe "a Vim script that runs only on Vim in devcontainer.vim."

```vim
if get(g:, "devcontainer_vim", v:false)
  " Run only on devcontainer.vim
endif
```


### run Customize the arguments of subcommands

`devcontainer.vim runargs -o` opens the argument settings file that is implicitly set to the run subcommand.

Updating this file allows you to specify arguments that you want to apply implicitly.

Mount the current directory to `/work` and set the working directory to the same location. Adjust as desired.

The default is as follows:

```
-v "$(pwd):/work" -v "$HOME/.vim:/root/.vim" --workdir /work
```

To revert to the default, regenerate runargs with the `-g` option.

```sh
devcontainer.vim runargs -g
```


### Using NeoVim

By adding the `--nvim` option or setting the environment variable `DEVCONTAINER_VIM_TYPE` to `nvim`, the nvim AppImage will be transferred and launched instead of vim.


```sh
devcontainer.vim --nvim start .
```

or

```sh
export DEVCONTAINER_VIM_TYPE=nvim
devcontainer.vim start .
```


### Using Shell

By adding the `--shell` option or setting the environment variable `DEVCONTAINER_SHELL_TYPE` to using shell, the shell will be transferred and launched instead of vim.


```sh
devcontainer.vim --shell bash start .
```

or

```sh
export DEVCONTAINER_VIM_TYPE=bash
devcontainer.vim start .
```

If you want to use the transferred Vim/Neovim, run `/VimRun.sh`.


## Migration:

### 2.x.x to 3.x.x

If you are using Vim with devcontainer.vim 2.x.x and will start using NeoVim with devcontainer.vim 3.x.x, you need either remove the vimrc mapping shown with `devcontainer.vim vimrc -o` or enclose the mapping with `if !has(“nvim”)` to enclose the mapping.

```vim
if !has("nvim")
  nnoremap <silent> "*yy yy:call SendToCdr('"')<CR>
  vnoremap <silent> "*y y:call SendToCdr('"')<CR>
endif
```


### 3.1.0 to 3.2.0

Clipboard integration with NeoVim has been available since v3.2.0,
so if you use both Vim and NeoVim, please rewrite your vimrc as follows.

```vim
if !has("nvim")
  nnoremap <silent> "*yy yy:call SendToCdr('"')<CR>
  vnoremap <silent> "*y y:call SendToCdr('"')<CR>
else
  nnoremap <silent> "*yy yy:lua SendToCdr('"')<CR>
  vnoremap <silent> "*y y:lua SendToCdr('"')<CR>
endif
```

## Limitation:

- amd64 architecture cannot be used in alpine-based containers
- When using NeoVim in an aarch64 container, you must use the system-installed version
- If the NeoVim AppImage is not available and there is no system-installed NeoVim, Vim will start instead of NeoVim
- When using NeoVim on macOS, only the system-installed NeoVim can be used
  If the system-installed NeoVim cannot be detected, Vim will start instead

## Install:

### binary download

[Latest version](https://github.com/mikoto2000/devcontainer.vim/releases/latest)


### go install

```sh
go install github.com/mikoto2000/devcontainer.vim@latest
```

If you want to install by specifying the version, install with the following command.
† Due to an incomplete setting, only v3.0.2 or later can be installed by specifying the version.

```sh
go install github.com/mikoto2000/devcontainer.vim/v3@3.0.2
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

This software is released under the MIT license. Please refer to LICENSE.


## Author:

mikoto2000 <mikoto2000@gmail.com>


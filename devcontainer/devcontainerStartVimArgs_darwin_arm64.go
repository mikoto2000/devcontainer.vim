//go:build darwin && arm64

package devcontainer

// `devcontainer.vim start` 時の `devcontainer exec` の引数を組み立てる
//
// Args:
//     - containerID: コンテナ ID
//     - workspaceFolder: ワークスペースフォルダパス
//     - vimFileName: コンテナ上に転送した vim のファイル名
//     - useSystemVim: vim or nvim of ""(空文字) 空文字でない場合は、
//       システムにインストールされた vim/nvim を使用する
// Return:
//     `devcontainer exec` に使うコマンドライン引数の配列
func devcontainerStartVimArgs(containerID string, workspaceFolder string, vimFileName string, useSystemVim string) []string {
	if useSystemVim != "" {
		return []string{
			"exec",
			"--container-id",
			containerID,
			"--workspace-folder",
			workspaceFolder,
			"sh",
			"-c",
			useSystemVim + " --cmd \"let g:devcontainer_vim = v:true\" -S /SendToTcp.vim -S /vimrc"}
	} else {
		return []string{
			"exec",
			"--container-id",
			containerID,
			"--workspace-folder",
			workspaceFolder,
			"sh",
			"-c",
			"cd /; tar zxf ./" + vimFileName + " -C ~/ > /dev/null; cd ~; sudo rm -rf ~/vim-static; mv $(ls -d ~/vim-*-aarch64) ~/vim-static;~/vim-static/AppRun --cmd \"let g:devcontainer_vim = v:true\" -S /SendToTcp.vim -S /vimrc"}
	}
}

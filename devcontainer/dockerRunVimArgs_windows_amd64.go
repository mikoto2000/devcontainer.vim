//go:build windows && amd64

package devcontainer

// `devcontainer.vim run` 時の `docker exec` の引数を組み立てる
//
// Args:
//   - containerID: コンテナ ID
//   - vimFileName: コンテナ上に転送した vim のファイル名
//   - useSystemVim: true の場合、システムにインストールされた vim/nvim を使用する
//
// Return:
//
//	`docker exec` に使うコマンドライン引数の配列
func dockerRunVimArgs(containerID string, vimFileName string, useSystemVim bool) []string {
	if useSystemVim {
		return []string{
			"exec",
			"-it",
			containerID,
			"sh",
			"-c",
			vimFileName + " --cmd \"let g:devcontainer_vim = v:true\" -S /SendToTcp.vim -S /vimrc"}
	} else {
		return []string{
			"exec",
			"-it",
			containerID,
			"sh",
			"-c",
			"cd ~; /" + vimFileName + " --appimage-extract > /dev/null; cd -; ~/squashfs-root/AppRun --cmd \"let g:devcontainer_vim = v:true\" -S /SendToTcp.vim -S /vimrc"}
	}
}

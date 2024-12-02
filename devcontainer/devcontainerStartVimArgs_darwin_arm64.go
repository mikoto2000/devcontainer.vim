//go:build darwin && arm64

package devcontainer

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

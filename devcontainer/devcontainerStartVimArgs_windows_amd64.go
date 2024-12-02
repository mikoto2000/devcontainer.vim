//go:build windows && amd64

package devcontainer

func devcontainerStartVimArgs(containerID string, workspaceFolder string, vimFileName string, useSystemVim string) []string {
	if useSystemVim != ""{
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
			"cd ~; /" + vimFileName + " --appimage-extract > /dev/null; cd -; ~/squashfs-root/AppRun --cmd \"let g:devcontainer_vim = v:true\" -S /SendToTcp.vim -S /vimrc"}
	}
}

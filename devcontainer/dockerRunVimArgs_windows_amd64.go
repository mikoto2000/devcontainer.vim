//go:build windows && amd64

package devcontainer

func dockerRunVimArgs(containerID string, vimFileName string, useSystemVim bool) []string {
	if useSystemVim {
		return []string{
			"exec",
			"-it",
			containerID,
			"sh",
			"-c",
			"vim --cmd \"let g:devcontainer_vim = v:true\" -S /SendToTcp.vim -S /vimrc"}
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

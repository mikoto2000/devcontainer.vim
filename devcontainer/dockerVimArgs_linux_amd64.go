//go:build linux && amd64

package devcontainer

func DockerVimArgs(containerID string, workspaceFolder string, vimFileName string) []string {
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

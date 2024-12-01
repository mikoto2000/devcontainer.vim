//go:build linux && amd64

package docker

func DockerVimArgs(containerID string, vimFileName string) []string {
	return []string{
		"exec",
		"-it",
		containerID,
		"sh",
		"-c",
		"cd ~; /" + vimFileName + " --appimage-extract > /dev/null; cd -; ~/squashfs-root/AppRun --cmd \"let g:devcontainer_vim = v:true\" -S /SendToTcp.vim -S /vimrc"}
}

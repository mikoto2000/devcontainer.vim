package util

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Jeffail/gabs/v2"
	"github.com/tailscale/hujson"
)

const binDirName = "bin"
const configDirName = "config"
const userDirectoryMode = 0700

// command で指定したものへパスが通っているかを確認する。
// パスが通っている場合 true を返却し、通っていない場合 false を返却する。
func IsExistsCommand(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}

type GetDirFunc func() (string, error)

// devcontainer.vim が使用するコンフィグディレクトリを作成し、返却する。
func CreateConfigDirectory(pathFunc GetDirFunc, dirName string) (string, error) {
	var baseDir, err = pathFunc()
	if err != nil {
		return "", err
	}
	var appConfigDir = filepath.Join(baseDir, dirName)
	if err := os.MkdirAll(appConfigDir, userDirectoryMode); err != nil {
		return "", err
	}
	return appConfigDir, nil
}

// devcontainer.vim が使用するキャッシュディレクトリを作成し、返却する。
//
// 返却値:
// devcontainer.vim 用のキャッシュディレクトリ
// devcontainer.vim 用の実行バイナリ格納ディレクトリ
// devcontainer.vim のマージ済み設定ファイル格納ディレクトリ
func CreateCacheDirectory(pathFunc GetDirFunc, dirName string) (string, string, string, string, error) {
	var baseDir, err = pathFunc()
	if err != nil {
		return "", "", "", "", err
	}
	var appCacheDir = filepath.Join(baseDir, dirName)
	if err := os.MkdirAll(appCacheDir, userDirectoryMode); err != nil {
		return "", "", "", "", err
	}
	var binDir = filepath.Join(baseDir, dirName, binDirName)
	if err := os.MkdirAll(binDir, userDirectoryMode); err != nil {
		return appCacheDir, "", "", "", err
	}
	var configDir = filepath.Join(baseDir, dirName, configDirName)
	if err := os.MkdirAll(configDir, userDirectoryMode); err != nil {
		return appCacheDir, binDir, "", "", err
	}
	// docker 用のコンフィグディレクトリ作成
	var configDirForDocker = filepath.Join(baseDir, dirName, configDirName, "docker")
	if err := os.MkdirAll(configDirForDocker, userDirectoryMode); err != nil {
		return appCacheDir, binDir, "", "", err
	}
	// devcontainer 用のコンフィグディレクトリ作成
	var configDirForDevcontainer = filepath.Join(baseDir, dirName, configDirName, "devcontainer")
	if err := os.MkdirAll(configDirForDevcontainer, userDirectoryMode); err != nil {
		return appCacheDir, binDir, configDir, "", err
	}
	return appCacheDir, binDir, configDirForDocker, configDirForDevcontainer, nil
}

func IsExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// AddExecutePermission adds owner execute permission and keeps the existing mode.
func AddExecutePermission(filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	return os.Chmod(filePath, fileInfo.Mode()|0100)
}

// baseConfigPath で指定した JSON に additionalConfigPath で指定した JSON をマージし、その結果を返却する
func readAndMergeConfig(baseConfigPath string, additionalConfigPath string) ([]byte, error) {

	// 設定ファイルを JWCC としてパースし、標準 JSON へ変換
	parsedBaseJSON, err := ParseJwcc(baseConfigPath)
	if err != nil {
		return nil, err
	}

	// 標準 JSON を gabs を使って再パース
	parsedBaseJSONGrabContainer, err := gabs.ParseJSON(parsedBaseJSON)
	if err != nil {
		return nil, err
	}

	// devcontainer.vim 用追加設定ファイル読み込み
	parsedAdditionalJSON, err := ParseJwcc(additionalConfigPath)
	if err != nil {
		return nil, err
	}

	// 標準 JSON を gabs を使って再パース
	parsedAdditionalJSONGrabContainer, err := gabs.ParseJSON(parsedAdditionalJSON)
	if err != nil {
		return nil, err
	}

	// MergeFn merges nested objects. Append matching arrays and prefer the
	// additional config for all other collisions.
	err = parsedBaseJSONGrabContainer.MergeFn(parsedAdditionalJSONGrabContainer, func(base, additional any) any {
		baseArray, baseIsArray := base.([]any)
		additionalArray, additionalIsArray := additional.([]any)
		if baseIsArray && additionalIsArray {
			return append(baseArray, additionalArray...)
		}
		return additional
	})
	if err != nil {
		return nil, err
	}

	// 設定ファイルの内容を返却
	return parsedBaseJSONGrabContainer.Bytes(), nil
}

// JWCC を標準 JSON に変換し、 []byte として返却
func ParseJwcc(jwccPath string) ([]byte, error) {
	// JWCC ファイル読み込み
	jwccContentBytes, err := os.ReadFile(jwccPath)
	if err != nil {
		return []byte{}, err
	}

	// JWCC をパースし、標準 JSON へ変換
	parsedJSON, err := hujson.Parse(jwccContentBytes)
	if err != nil {
		return []byte{}, err
	}

	parsedJSON.Standardize()

	return parsedJSON.Pack(), nil
}

// configFilePath と additionalConfigFilePath の JSON をマージし、
// devcontainer.vim のキャッシュディレクトリ内の設定ファイル格納ディレクトリへ格納する。
// 作成した devcontainer.json を格納しているディレクトリのパスを返却する。
func CreateConfigFileForDevcontainer(configDirForDevcontainer string, workspaceFolder string, configFilePath string, additionalConfigFilePath string) (string, error) {

	// マージ要否判定して最終的に使う JSON のコンテンツを組み立てる
	var configFileContent []byte
	var err error
	if IsExists(additionalConfigFilePath) {
		// JSON のマージ
		configFileContent, err = readAndMergeConfig(configFilePath, additionalConfigFilePath)
	} else {
		// ベースの設定をそのまま使用
		configFileContent, err = os.ReadFile(configFilePath)
	}
	if err != nil {
		return "", err
	}

	// 設定管理フォルダに JSON を配置
	generateConfigDir, err := GetConfigDir(configDirForDevcontainer, workspaceFolder)
	if err != nil {
		return "", err
	}
	generateConfigFilePath := filepath.Join(generateConfigDir, "devcontainer.json")
	err = os.MkdirAll(generateConfigDir, userDirectoryMode)
	if err != nil {
		return "", err
	}
	err = os.WriteFile(generateConfigFilePath, configFileContent, 0666)
	if err != nil {
		return "", err
	}
	return generateConfigFilePath, nil
}

// devcontainer.vim 用の devcontainer.json 格納先ディレクトリを計算して返却する。
// `<devcontainer.vim のキャッシュディレクトリ>/config/<workspaceFolder の絶対パスを md5 播種化した文字列>` のディレクトリを返却
func GetConfigDir(configDirForDevcontainer string, workspaceFolder string) (string, error) {
	workspaceFolderAbs, err := filepath.Abs(workspaceFolder)
	if err != nil {
		return "", err
	}
	workspaceFolderHash := md5.Sum([]byte(workspaceFolderAbs))

	workspaceFolderHashString := hex.EncodeToString(workspaceFolderHash[:])
	return filepath.Join(configDirForDevcontainer, workspaceFolderHashString), nil
}

// WSL 上で動いているかを判定する
func IsWsl() bool {
	if _, exists := os.LookupEnv("WSL_DISTRO_NAME"); exists {
		return true
	}
	// A WSL session can still launch Windows processes if the distribution
	// name was removed but its interop socket is available.
	socket := os.Getenv("WSL_INTEROP")
	if runtime.GOOS != "linux" || !filepath.IsAbs(socket) {
		return false
	}
	info, err := os.Stat(socket)
	return err == nil && info.Mode()&os.ModeSocket != 0
}

func CreateFileWithContents(file string, content string, permission fs.FileMode) error {
	err := os.WriteFile(file, []byte(content), permission)
	if err != nil {
		return err
	}
	return nil
}

// NormalizeContainerArch returns amd64 or aarch64 for container installers and
// their cache keys. URL builders map these names to each release's asset names,
// which may use arm64 or x86_64. An empty input identifies a host-only tool.
func NormalizeContainerArch(containerArch string) (string, error) {
	if containerArch == "amd64" || containerArch == "x86_64" {
		return "amd64", nil
	} else if containerArch == "arm64" || containerArch == "aarch64" {
		return "aarch64", nil
	} else if containerArch == "" {
		return "", nil
	} else {
		return "", fmt.Errorf("Unknown Architecture: %q", containerArch)
	}
}

func RemoveEmptyString(input []string) []string {
	var result []string

	for _, v := range input {
		if strings.TrimSpace(v) != "" {
			result = append(result, v)
		}
	}

	return result
}

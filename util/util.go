package util

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/Jeffail/gabs/v2"
	"github.com/tailscale/hujson"
)

const binDirName = "bin"
const configDirName = "config"

// command で指定したものへパスが通っているかを確認する。
// パスが通っている場合 true を返却し、通っていない場合 false を返却する。
func IsExistsCommand(command string) bool {
	_, err := exec.LookPath(command)
	if err != nil {
		return false
	}
	return true
}

type GetDirFunc func() (string, error)

// devcontainer.vim が使用するコンフィグディレクトリを作成し、返却する。
func CreateConfigDirectory(pathFunc GetDirFunc, dirName string) (string) {
	var baseDir, err = pathFunc()
	if err != nil {
		panic(err)
	}
	var appConfigDir = filepath.Join(baseDir, dirName)
	if err := os.MkdirAll(appConfigDir, 0766); err != nil {
		panic(err)
	}
	return appConfigDir
}

// devcontainer.vim が使用するキャッシュディレクトリを作成し、返却する。
//
// 返却値:
// devcontainer.vim 用のキャッシュディレクトリ
// devcontainer.vim 用の実行バイナリ格納ディレクトリ
// devcontainer.vim のマージ済み設定ファイル格納ディレクトリ
func CreateCacheDirectory(pathFunc GetDirFunc, dirName string) (string, string, string, string) {
	var baseDir, err = pathFunc()
	if err != nil {
		panic(err)
	}
	var appCacheDir = filepath.Join(baseDir, dirName)
	if err := os.MkdirAll(appCacheDir, 0766); err != nil {
		panic(err)
	}
	var binDir = filepath.Join(baseDir, dirName, binDirName)
	if err := os.MkdirAll(binDir, 0766); err != nil {
		panic(err)
	}
	var configDir = filepath.Join(baseDir, dirName, configDirName)
	if err := os.MkdirAll(configDir, 0766); err != nil {
		panic(err)
	}
	// docker 用のコンフィグディレクトリ作成
	var configDirForDocker = filepath.Join(baseDir, dirName, configDirName, "docker")
	if err := os.MkdirAll(configDirForDocker, 0766); err != nil {
		panic(err)
	}
	// devcontainer 用のコンフィグディレクトリ作成
	var configDirForDevcontainer = filepath.Join(baseDir, dirName, configDirName, "devcontainer")
	if err := os.MkdirAll(configDirForDevcontainer, 0766); err != nil {
		panic(err)
	}
	return appCacheDir, binDir, configDirForDocker, configDirForDevcontainer
}

func IsExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

func AddExecutePermission(filePath string) error {
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return err
	}
	fileMode := fileInfo.Mode()
	err = os.Chmod(filePath, fileMode|0111)
	if err != nil {
		return err
	}

	return nil
}

// baseConfigPath で指定した JSON に additionalConfigPath で指定した JSON をマージし、その結果を返却する
func readAndMergeConfig(baseConfigPath string, additionalConfigPath string) ([]byte, error) {

	// 設定ファイル読み込み
	parsedBaseJsonContentBytes, err := os.ReadFile(baseConfigPath)
	if err != nil {
		return nil, err
	}

	// 設定ファイルを JWCC としてパースし、標準 JSON へ変換
	parsedBaseJson, err := hujson.Parse(parsedBaseJsonContentBytes)
	if err != nil {
		return nil, err
	}
	parsedBaseJson.Standardize()

	// 標準 JSON を gabs を使って再パース
	parsedBaseJsonGrabContainer, err := gabs.ParseJSON(parsedBaseJson.Pack())
	if err != nil {
		return nil, err
	}

	// devcontainer.vim 用追加設定ファイル読み込み
	parsedAdditionalJsonContentBytes, err := os.ReadFile(additionalConfigPath)
	if err != nil {
		return nil, err
	}

	// devcontainer.vim 用追加設定ファイルを JWCC としてパースし、標準 JSON へ変換
	parsedAdditionalJson, err := hujson.Parse(parsedAdditionalJsonContentBytes)
	if err != nil {
		return nil, err
	}
	parsedAdditionalJson.Standardize()

	// 標準 JSON を gabs を使って再パース
	parsedAdditionalJsonGrabContainer, err := gabs.ParseJSON(parsedAdditionalJson.Pack())
	if err != nil {
		return nil, err
	}

	// gabs を使って JSON をマージ
	parsedBaseJsonGrabContainer.Merge(parsedAdditionalJsonGrabContainer)

	// 設定ファイルの内容を返却
	return parsedBaseJsonGrabContainer.Bytes(), nil
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
	generateConfigDir := GetConfigDir(configDirForDevcontainer, workspaceFolder)
	generateConfigFilePath := filepath.Join(generateConfigDir, "devcontainer.json")
	err = os.MkdirAll(generateConfigDir, 0777)
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
func GetConfigDir(configDirForDevcontainer string, workspaceFolder string) string {
	workspaceFolderAbs, err := filepath.Abs(workspaceFolder)
	if err != nil {
		panic(err)
	}
	workspaceFolderHash := md5.Sum([]byte(workspaceFolderAbs))

	workspaceFolderHashString := hex.EncodeToString(workspaceFolderHash[:])
	return filepath.Join(configDirForDevcontainer, workspaceFolderHashString)
}

// WSL 上で動いているかを判定する
func IsWsl() bool {
	_, exists := os.LookupEnv("WSL_DISTRO_NAME")
	return exists
}

// 関連付けられたアプリケーションで開く
func OpenFileWithDefaultApp(filePath string) error {
    var cmd *exec.Cmd

    switch runtime.GOOS {
    case "darwin":
        cmd = exec.Command("open", filePath) // macOS
    case "windows":
        cmd = exec.Command("cmd", "/c", "start", "", filePath) // Windows
    default:
        cmd = exec.Command("xdg-open", filePath) // Linux
    }

    return cmd.Run()
}

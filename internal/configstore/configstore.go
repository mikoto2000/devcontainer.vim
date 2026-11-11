// Package configstore owns the application's configuration directories and workspace configuration files.
package configstore

import (
	"crypto/md5"
	"encoding/hex"
	"os"
	"path/filepath"

	"github.com/Jeffail/gabs/v2"
	"github.com/tailscale/hujson"
)

const binDirName = "bin"
const configDirName = "config"
const userDirectoryMode = 0700

type GetDirFunc func() (string, error)

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

func CreateConfigFileForDevcontainer(configDirForDevcontainer string, workspaceFolder string, configFilePath string, additionalConfigFilePath string) (string, error) {

	// マージ要否判定して最終的に使う JSON のコンテンツを組み立てる
	var configFileContent []byte
	var err error
	if _, statErr := os.Stat(additionalConfigFilePath); statErr == nil {
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

// GetConfigDir hashes the absolute workspace path so callers share one cache key.
func GetConfigDir(configDirForDevcontainer string, workspaceFolder string) (string, error) {
	workspaceFolderAbs, err := filepath.Abs(workspaceFolder)
	if err != nil {
		return "", err
	}
	workspaceFolderHash := md5.Sum([]byte(workspaceFolderAbs))

	workspaceFolderHashString := hex.EncodeToString(workspaceFolderHash[:])
	return filepath.Join(configDirForDevcontainer, workspaceFolderHashString), nil
}

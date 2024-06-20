package util

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
)

func ExtractOneFromTgz(filePath string, destDir string, targetFile string) error {

	// ファイルオープン
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// gzip 展開用 Reader 作成
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzipReader.Close()

	// tar 展開用 Reader の作成
	tarReader := tar.NewReader(gzipReader)

	// tar 内のファイルを走査
	for {
		// エントリ取得
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// ファイル名が一致していなければ次のエントリへ
		if header.Name != targetFile {
			continue
		}

		// 出力先ファイルパス組み立て
		destFilePath := filepath.Join(destDir, header.Name)

		err = os.MkdirAll(destDir, 0755)
		if err != nil {
			return err
		}

		destFile, err := os.OpenFile(
			destFilePath,
			os.O_CREATE|os.O_TRUNC|os.O_WRONLY,
			os.FileMode(header.Mode))
		if err != nil {
			return err
		}
		defer destFile.Close()

		_, err = io.Copy(destFile, tarReader)
		if err != nil {
			return err
		}

		break
	}

	return nil
}

func ExtractOneFromZip(filePath string, destDir string, targetFile string) error {

	// zip 展開用 Reader 作成
	zipReader, err := zip.OpenReader(filePath)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	// tar 内のファイルを走査
	for _, file := range zipReader.File {

		// ファイル名が一致していなければ次のエントリへ
		if file.Name != targetFile {
			continue
		}

		// 出力先ファイルパス組み立て
		destFilePath := filepath.Join(destDir, file.Name)

		err = os.MkdirAll(destDir, 0755)
		if err != nil {
			return err
		}

		destFile, err := os.Create(destFilePath)
		if err != nil {
			return err
		}
		defer destFile.Close()

		srcFile, err := file.Open()
		if err != nil {
			return err
		}
		defer srcFile.Close()

		_, err = io.Copy(destFile, srcFile)
	}

	return nil
}

func Copy(src string, dest string) error {
	// 出力元をオープン
	srcFile, err := os.OpenFile(src, os.O_RDONLY, 0666)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// 出力先ディレクトリ作成
	destDir := filepath.Dir(dest)
	err = os.MkdirAll(destDir, 0755)
	if err != nil {
		return err
	}

	// 出力先ファイル作成
	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	// 出力元のファイル情報を取得
	fileInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	// パーミッションをコピー
	err = os.Chmod(dest, fileInfo.Mode())
	if err != nil {
		return err
	}

	return nil
}

package oras

import (
	"context"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
)

func Pull(id string, tagName string, destDir string) error {

	// リモートリポジトリの設定
	src, err := remote.NewRepository(id)
	if err != nil {
		return err
	}

	// メモリストアの作成
	dst, err := file.New(destDir)
	if err != nil {
		return err
	}
	ctx := context.Background()

	_, err = oras.Copy(ctx, src, tagName, dst, tagName, oras.DefaultCopyOptions)
	if err != nil {
		return err
	}

	return nil
}

package oras

import (
	"context"
	"errors"

	"oras.land/oras-go/v2"
	"oras.land/oras-go/v2/content/file"
	"oras.land/oras-go/v2/registry/remote"
)

func Pull(ctx context.Context, id string, tagName string, destDir string) error {

	// リモートリポジトリの設定
	src, err := remote.NewRepository(id)
	if err != nil {
		return err
	}

	// ローカルファイルストアの作成
	dst, err := file.New(destDir)
	if err != nil {
		return err
	}
	_, err = oras.Copy(ctx, src, tagName, dst, tagName, oras.DefaultCopyOptions)
	return errors.Join(err, dst.Close())
}

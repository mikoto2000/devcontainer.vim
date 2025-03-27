# devcontainer.vim 開発ガイド

## ビルドコマンド
- 現在のプラットフォーム用にビルド: `make build`
- 全プラットフォーム用にビルド: `make build-all`
- コードフォーマット: `make fmt`
- リンター実行: `make lint`

## テストコマンド
- 全テスト実行: `make test`
- 単一テスト実行: `go test -v ./[パッケージ] -run [テスト名]`
- 例: `go test -v ./devcontainer -run TestStart`

## コードスタイルガイドライン
- Go標準フォーマットを使用 (`go fmt`)
- staticcheckルールに従う (ST1003, ST1016)
- インポート順: 標準ライブラリ、サードパーティ、プロジェクト固有の順
- エラー処理: 即時チェックとコンテキストを含むメッセージ
- エラーは `fmt.Fprintf(os.Stderr, ...)` を使用、通常出力はstdout
- テストリソース: `/test/resource/` または `/test/project/` に配置
- deferを使用してリソースを確実にクリーンアップ
- テスト容易性のためインターフェースベースの設計を使用
- プラットフォーム固有コード: ランタイムチェックと必要に応じて別ファイル
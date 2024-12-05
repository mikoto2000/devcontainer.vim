package util

import (
	"io"
	"net"
)

// ポートフォワーディングの主処理を行う。
// 転送先・転送元のデータ通信をリレーする。
func forward(src net.Conn, dstAddr string) error {
	defer src.Close()

	// 転送先に接続
	dst, err := net.Dial("tcp", dstAddr)
	if err != nil {
		return err
	}
	defer dst.Close()

	// 双方向のデータ転送を行う
	// クライアント → 転送先
	go io.Copy(dst, src)

	// 転送先 → クライアント
	io.Copy(src, dst)

	return nil
}

// ポートフォワーディング開始
func StartForwarding(listenAddr, forwardAddr string) error {
	// listener 生成
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return err
	}
	defer listener.Close()

	for {
		// listen 開始
		client, err := listener.Accept()
		if err != nil {
			continue
		}

		// 主処理(バケツリレー)を実行
		go forward(client, forwardAddr)
	}
}

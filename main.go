package main

import (
	"fmt"
	"log"
	"net"
	"time"

	"kuramo.ch/eibs7-controller/echonetlite" // モジュールパスはご自身のものに合わせてください
)

// ECHONET Lite の標準ポート
const echonetLitePort = 3610

// 送信元 (コントローラー) の ECHONET Lite オブジェクト (例: コントローラークラス)
// 関数外で定義しておき、フレーム作成時に使用する
var controllerEOJ = echonetlite.NewEOJ(0x05, 0xFF, 0x01) // クラスグループ: 管理操作, クラス: コントローラ, インスタンス: 1

// トランザクションIDを管理するための変数 (単純な例)
var currentTID echonetlite.TID = 0

// 次のトランザクションIDを取得する関数
func getNextTID() echonetlite.TID {
	currentTID++
	// TIDは16ビットなので、オーバーフローしたらリセット (0は予約されている場合があるので1から)
	if currentTID == 0 {
		currentTID = 1
	}
	return currentTID
}

// sendAndReceiveEchonetLiteFrame は指定された ECHONET Lite フレームを送信し、
// 応答を指定されたタイムアウト時間まで待機して受信します。
// 送信元ポートは echonetLitePort (3610) にバインドされます。
// 成功した場合、受信したバイト列、送信元アドレス、nil エラーを返します。
// 失敗した場合 (タイムアウト含む)、nil, nil, エラーを返します。
func sendAndReceiveEchonetLiteFrame(targetIP string, frame echonetlite.Frame, timeout time.Duration) ([]byte, *net.UDPAddr, error) {
	// 1. フレームをバイト列にシリアライズする
	sendData, err := frame.MarshalBinary()
	if err != nil {
		return nil, nil, fmt.Errorf("フレームのシリアライズに失敗しました (TID: %d): %w", frame.TID, err)
	}
	log.Printf("送信データ (Hex, TID: %d): %X", frame.TID, sendData)

	// 2. 送信先アドレスを解決する
	remoteAddrStr := net.JoinHostPort(targetIP, fmt.Sprintf("%d", echonetLitePort))
	remoteAddr, err := net.ResolveUDPAddr("udp", remoteAddrStr)
	if err != nil {
		return nil, nil, fmt.Errorf("送信先アドレスの解決に失敗しました (%s): %w", remoteAddrStr, err)
	}
	log.Printf("送信先: %s", remoteAddr.String())

	// 3. UDPソケットを開く (送信元ポートを 3610 にバインド)
	// ListenUDP を使うことで、応答元の情報を ReadFromUDP で取得できる
	localAddr := &net.UDPAddr{Port: echonetLitePort}
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		// ポートが既に使用されている可能性も考慮 (特にデバッグ中に複数起動した場合)
		return nil, nil, fmt.Errorf("UDPポート %d でのListenに失敗しました: %w", echonetLitePort, err)
	}
	defer conn.Close() // 関数終了時にソケットを閉じる
	log.Printf("UDPソケットを開きました (ローカル: %s)", conn.LocalAddr().String())

	// 4. バイト列を UDP で送信する
	bytesSent, err := conn.WriteToUDP(sendData, remoteAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("UDPデータの送信に失敗しました (宛先: %s): %w", remoteAddr.String(), err)
	}
	log.Printf("%d バイトのデータを送信しました (宛先: %s, TID: %d)", bytesSent, remoteAddr.String(), frame.TID)

	// 5. 応答を待機する
	log.Printf("応答を待機しています (TID: %d, タイムアウト: %s)...", frame.TID, timeout)

	// 応答受信用のバッファ
	buffer := make([]byte, 1024) // ECHONET Lite フレームには十分なサイズ

	// 読み取りタイムアウトを設定
	conn.SetReadDeadline(time.Now().Add(timeout))

	// データを受信する
	bytesRead, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		// タイムアウトエラーか、それ以外のエラーかを確認
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("応答がタイムアウトしました (TID: %d)", frame.TID)
			// タイムアウトもエラーとして返す
			return nil, nil, err
		}
		// その他の受信エラー
		return nil, nil, fmt.Errorf("UDPデータの受信に失敗しました (TID: %d): %w", frame.TID, err)
	}

	log.Printf("%s から %d バイトのデータを受信しました (TID: %d)", addr.String(), bytesRead, frame.TID)
	log.Printf("受信データ (Hex, TID: %d): %X", frame.TID, buffer[:bytesRead])

	// 受信成功：受信したバイト列、送信元アドレス、nilエラーを返す
	return buffer[:bytesRead], addr, nil
}

func main() {
	// --- 設定値 (将来的には設定ファイルから読み込む) ---
	targetIP := "192.168.2.107" // ★★★ 実際の EIBS7 の IP アドレスに置き換えてください ★★★
	responseTimeout := 5 * time.Second

	// --- 送信するフレームを作成 ---
	targetDeviceEOJ := echonetlite.NewEOJ(0x02, 0x7D, 0x01) // 蓄電池クラス
	propertyEPC := byte(0xE4)                               // 蓄電残量3
	tid := getNextTID()                                     // 新しいトランザクションIDを取得

	getFrame := echonetlite.Frame{
		EHD1: echonetlite.EchonetLiteEHD1,
		EHD2: echonetlite.Format1,
		TID:  tid,
		SEOJ: controllerEOJ,          // 送信元EOJ
		DEOJ: targetDeviceEOJ,        // 宛先EOJ
		ESV:  echonetlite.ESVGet_SNA, // Get 要求 (応答要)
		OPC:  1,                      // 操作プロパティ数: 1
		Properties: []echonetlite.Property{
			{
				EPC: propertyEPC,
				PDC: 0,   // Get 要求なのでデータ長は 0
				EDT: nil, // Get 要求なのでデータは nil
			},
		},
	}

	// --- フレームを送信し、応答を受信 ---
	receivedData, sourceAddr, err := sendAndReceiveEchonetLiteFrame(targetIP, getFrame, responseTimeout)
	if err != nil {
		// エラー処理 (タイムアウトもここに含まれる)
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("処理がタイムアウトしました (TID: %d)", tid)
			// タイムアウト時の処理 (リトライ、ログ記録など)
		} else {
			// その他のエラー
			log.Printf("ECHONET Lite 通信中にエラーが発生しました (TID: %d): %v", tid, err)
		}
		// エラーが発生したら main 処理を終了 (または適切にハンドリング)
		return
	}

	// --- 応答受信成功時の処理 ---
	log.Printf("正常に応答を受信しました (TID: %d, 送信元: %s)", tid, sourceAddr.String())

	_ = receivedData

	// TODO: 受信したバイト列 (receivedData) を echonetlite.Frame にデシリアライズする処理を実装する
	// responseFrame, err := echonetlite.UnmarshalBinary(receivedData) // デシリアライズ関数 (未実装)
	// if err != nil {
	//     log.Printf("受信データのデシリアライズに失敗しました (TID: %d): %v", tid, err)
	// } else {
	//     // デシリアライズ成功時の処理
	//     log.Printf("受信フレーム (TID: %d): %+v", tid, responseFrame)
	//     // TID の一致確認
	//     if responseFrame.TID != tid {
	//         log.Printf("警告: 受信したTID (%d) が送信したTID (%d) と一致しません。", responseFrame.TID, tid)
	//     }
	//     // ESV の確認 (Get応答か、エラー応答か)
	//     switch responseFrame.ESV {
	//     case echonetlite.ESVGet_Res: // 0x72
	//         log.Println("Get応答を受信しました。")
	//         // プロパティデータの処理 (EDTのデコードなど)
	//         for _, prop := range responseFrame.Properties {
	//             if prop.EPC == propertyEPC {
	//                 log.Printf("  EPC: 0x%X, PDC: %d, EDT: %X", prop.EPC, prop.PDC, prop.EDT)
	//                 // ここで EDT を具体的な値に変換する処理を追加
	//             }
	//         }
	//     case echonetlite.ESVGet_SNA_Err: // 0x52
	//         log.Printf("Getエラー応答を受信しました。")
	//         // エラー内容の確認 (EDTにエラー情報が含まれる場合がある)
	//     default:
	//         log.Printf("予期しないESV (%X) を受信しました。", responseFrame.ESV)
	//     }
	// }
}

package main

import (
	"fmt"
	"log"
	"net"
	"os" // ファイル読み込み用に os パッケージをインポート
	"time"

	"github.com/BurntSushi/toml"             // TOMLパーサーをインポート
	"kuramo.ch/eibs7-controller/echonetlite" // モジュールパスはご自身のものに合わせてください
)

// ECHONET Lite の標準ポート
const echonetLitePort = 3610

// 送信元 (コントローラー) の ECHONET Lite オブジェクト (例: コントローラークラス)
var controllerEOJ = echonetlite.NewEOJ(0x05, 0xFF, 0x01) // クラスグループ: 管理操作, クラス: コントローラ, インスタンス: 1

// トランザクションIDを管理するための変数 (単純な例)
var currentTID echonetlite.TID = 0

// 設定ファイルの内容をマッピングする構造体
type Config struct {
	TargetIP               string `toml:"target_ip"`
	MonitorIntervalSeconds int    `toml:"monitor_interval_seconds"`
}

// 設定ファイル名
const configFileName = "config.toml"

// loadConfig は設定ファイルを読み込み、Config構造体を返します。
func loadConfig(filePath string) (*Config, error) {
	var config Config

	// ファイルの内容を読み込む
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("設定ファイル '%s' の読み込みに失敗しました: %w", filePath, err)
	}

	// TOMLデータを構造体にデコードする
	if err := toml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("設定ファイル '%s' の解析に失敗しました: %w", filePath, err)
	}

	// 必須項目のチェック (例: TargetIP)
	if config.TargetIP == "" {
		return nil, fmt.Errorf("設定ファイル '%s' に 'target_ip' が設定されていないか、空です", filePath)
	}

	// MonitorIntervalSeconds のデフォルト値設定
	if config.MonitorIntervalSeconds <= 0 {
		log.Printf("設定ファイル '%s' の 'monitor_interval_seconds' が未設定または0以下です。デフォルト値10秒を使用します。", filePath)
		config.MonitorIntervalSeconds = 10
	}

	return &config, nil
}

// 次のトランザクションIDを取得する関数
func getNextTID() echonetlite.TID {
	currentTID++
	if currentTID == 0 {
		currentTID = 1
	}
	return currentTID
}

// sendAndReceiveEchonetLiteFrame は指定された ECHONET Lite フレームを送信し、
// 応答を指定されたタイムアウト時間まで待機して受信します。
// (この関数は変更なし)
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
	localAddr := &net.UDPAddr{Port: echonetLitePort}
	conn, err := net.ListenUDP("udp", localAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("UDPポート %d でのListenに失敗しました: %w", echonetLitePort, err)
	}
	defer conn.Close()
	log.Printf("UDPソケットを開きました (ローカル: %s)", conn.LocalAddr().String())

	// 4. バイト列を UDP で送信する
	bytesSent, err := conn.WriteToUDP(sendData, remoteAddr)
	if err != nil {
		return nil, nil, fmt.Errorf("UDPデータの送信に失敗しました (宛先: %s): %w", remoteAddr.String(), err)
	}
	log.Printf("%d バイトのデータを送信しました (宛先: %s, TID: %d)", bytesSent, remoteAddr.String(), frame.TID)

	// 5. 応答を待機する
	log.Printf("応答を待機しています (TID: %d, タイムアウト: %s)...", frame.TID, timeout)

	buffer := make([]byte, 1024)
	conn.SetReadDeadline(time.Now().Add(timeout))

	bytesRead, addr, err := conn.ReadFromUDP(buffer)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			log.Printf("応答がタイムアウトしました (TID: %d)", frame.TID)
			return nil, nil, err
		}
		return nil, nil, fmt.Errorf("UDPデータの受信に失敗しました (TID: %d): %w", frame.TID, err)
	}

	log.Printf("%s から %d バイトのデータを受信しました (TID: %d)", addr.String(), bytesRead, frame.TID)
	log.Printf("受信データ (Hex, TID: %d): %X", frame.TID, buffer[:bytesRead])

	return buffer[:bytesRead], addr, nil
}

// MonitoringTarget は、監視対象のECHONET Liteオブジェクトと取得するプロパティのリストを定義します。
type MonitoringTarget struct {
	EOJ        echonetlite.EOJ
	EPCs       []byte
	ObjectName string // ログ出力用のオブジェクト名
}

func main() {
	// --- 設定ファイルの読み込み ---
	cfg, err := loadConfig(configFileName)
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}
	log.Printf("設定ファイル '%s' を読み込みました。TargetIP: %s, MonitorIntervalSeconds: %d", configFileName, cfg.TargetIP, cfg.MonitorIntervalSeconds)

	// --- 設定値 ---
	targetIP := cfg.TargetIP // 設定ファイルから読み込んだIPアドレスを使用
	responseTimeout := 5 * time.Second

	// --- 監視対象の定義 ---
	// README_prototype.md および以前の指示に基づく
	targets := []MonitoringTarget{
		{
			EOJ:        echonetlite.NewEOJ(0x02, 0x7D, 0x01), // 蓄電池
			EPCs:       []byte{0xE4, 0xDA, 0xEB, 0xD3, 0xA0}, // 蓄電残量3, 運転モード, 充電電力設定値, 瞬時充放電電力, AC実効容量
			ObjectName: "蓄電池 (027D01)",
		},
		{
			EOJ:        echonetlite.NewEOJ(0x02, 0x79, 0x01), // 住宅用太陽光発電
			EPCs:       []byte{0xE0},                         // 瞬時発電電力計測値
			ObjectName: "住宅用太陽光発電 (027901)",
		},
		{
			EOJ:        echonetlite.NewEOJ(0x02, 0x87, 0x01), // 分電盤メータリング
			EPCs:       []byte{0xC6},                         // 瞬時電力計測値
			ObjectName: "分電盤メータリング (028701)",
		},
		{
			EOJ:        echonetlite.NewEOJ(0x02, 0xA5, 0x01), // マルチ入力PCS
			EPCs:       []byte{0xE7},                         // 瞬時電力計測値
			ObjectName: "マルチ入力PCS (02A501)",
		},
	}

	// --- 定期実行のための Ticker を作成 ---
	ticker := time.NewTicker(time.Duration(cfg.MonitorIntervalSeconds) * time.Second)
	defer ticker.Stop()

	log.Printf("監視を開始します。監視間隔: %d秒", cfg.MonitorIntervalSeconds)

	// --- メインループ (監視サイクル) ---
	for range ticker.C {
		log.Println("--------------------------------------------------")
		log.Println("監視サイクル開始")

		for _, target := range targets {
			tid := getNextTID()
			log.Printf("[%s] データ取得開始 (TID: %d)", target.ObjectName, tid)

			var props []echonetlite.Property
			for _, epc := range target.EPCs {
				props = append(props, echonetlite.Property{EPC: epc, PDC: 0, EDT: nil})
			}

			getFrame := echonetlite.Frame{
				EHD1:       echonetlite.EchonetLiteEHD1,
				EHD2:       echonetlite.Format1,
				TID:        tid,
				SEOJ:       controllerEOJ,
				DEOJ:       target.EOJ,
				ESV:        echonetlite.ESVGet,
				OPC:        byte(len(props)),
				Properties: props,
			}

			// --- フレームを送信し、応答を受信 ---
			receivedData, sourceAddr, err := sendAndReceiveEchonetLiteFrame(targetIP, getFrame, responseTimeout)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					log.Printf("[%s] 処理がタイムアウトしました (TID: %d)", target.ObjectName, tid)
				} else {
					log.Printf("[%s] ECHONET Lite 通信中にエラーが発生しました (TID: %d): %v", target.ObjectName, tid, err)
				}
				continue // エラーが発生しても次のターゲットの処理へ
			}

			// --- 応答受信成功時の処理 ---
			log.Printf("[%s] 正常に応答を受信しました (TID: %d, 送信元: %s, データ長: %d bytes)", target.ObjectName, tid, sourceAddr.String(), len(receivedData))

			// 受信したバイト列 (receivedData) を echonetlite.Frame にデシリアライズする
			var responseFrame echonetlite.Frame
			err = responseFrame.UnmarshalBinary(receivedData)
			if err != nil {
				log.Printf("[%s] 受信データのデシリアライズに失敗しました (TID: %d): %v", target.ObjectName, tid, err)
				continue // 次のターゲットへ
			}

			// TID の一致確認
			if responseFrame.TID != tid {
				log.Printf("[%s] 警告: 受信したTID (%d) が送信したTID (%d) と一致しません。", target.ObjectName, responseFrame.TID, tid)
				// TIDが不一致でも処理を続けるか、ここで中断するかは要件による
			}

			// ESV の確認
			switch responseFrame.ESV {
			case echonetlite.ESVGet_Res: // 0x72 - Property value read response
				log.Printf("[%s] Get応答を受信しました (TID: %d, ESV: 0x%X)", target.ObjectName, responseFrame.TID, responseFrame.ESV)
				if len(responseFrame.Properties) == 0 {
					log.Printf("[%s] Get応答にプロパティが含まれていません (TID: %d)", target.ObjectName, responseFrame.TID)
				}
				for _, prop := range responseFrame.Properties {
					log.Printf("[%s]   EPC: 0x%X, PDC: %d, EDT: %X (TID: %d)", target.ObjectName, prop.EPC, prop.PDC, prop.EDT, responseFrame.TID)
					// TODO: ここで EDT を具体的なデータ型に変換し、意味のある値として解釈する処理を追加する
				}
			case echonetlite.ESVGet_SNA: // 0x52 - Property value read request error
				log.Printf("[%s] Getエラー応答を受信しました (TID: %d, ESV: 0x%X)", target.ObjectName, responseFrame.TID, responseFrame.ESV)
				// エラー応答の場合、Propertiesにエラーの原因を示す情報が含まれることがある (例: EPCが処理不可など)
			default:
				log.Printf("[%s] 予期しないESV (0x%X) を受信しました (TID: %d)", target.ObjectName, responseFrame.ESV, responseFrame.TID)
			}
		}
		log.Println("監視サイクル終了 (全ターゲット処理完了)")
	}
}

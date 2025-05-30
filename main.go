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

	// --- 定期実行のための Ticker を作成 ---
	ticker := time.NewTicker(time.Duration(cfg.MonitorIntervalSeconds) * time.Second)
	defer ticker.Stop()

	log.Printf("監視を開始します。監視間隔: %d秒", cfg.MonitorIntervalSeconds)

	// --- メインループ ---
	for range ticker.C {
		log.Println("--------------------------------------------------")
		log.Println("監視サイクル開始")

		// --- 送信するフレームを作成 ---
		targetDeviceEOJ := echonetlite.NewEOJ(0x02, 0x7D, 0x01) // 蓄電池クラス
		propertyEPC_RemainingCapacity := byte(0xE4)             // 蓄電残量3
		propertyEPC_OperatingMode := byte(0xDA)                 // 運転モード設定
		propertyEPC_ChargingPowerSetting := byte(0xEB)          // 充電電力設定値
		propertyEPC_InstChargeDischargePower := byte(0xD3)      // 瞬時充放電電力計測値
		propertyEPC_ACEffectiveCapacity := byte(0xA0)           // AC実効容量（充電）
		tid := getNextTID()

		getFrame := echonetlite.Frame{
			EHD1: echonetlite.EchonetLiteEHD1,
			EHD2: echonetlite.Format1,
			TID:  tid,
			SEOJ: controllerEOJ,
			DEOJ: targetDeviceEOJ,
			ESV:  echonetlite.ESVGet, // Property value read request (response required)
			OPC:  5,                  // 要求するプロパティ数を5に更新
			Properties: []echonetlite.Property{
				{
					EPC: propertyEPC_RemainingCapacity,
					PDC: 0,
					EDT: nil,
				},
				{
					EPC: propertyEPC_OperatingMode,
					PDC: 0,
					EDT: nil,
				},
				{
					EPC: propertyEPC_ChargingPowerSetting,
					PDC: 0,
					EDT: nil,
				},
				{
					EPC: propertyEPC_InstChargeDischargePower,
					PDC: 0,
					EDT: nil,
				},
				{
					EPC: propertyEPC_ACEffectiveCapacity,
					PDC: 0,
					EDT: nil,
				},
			},
		}

		// --- フレームを送信し、応答を受信 ---
		receivedData, sourceAddr, err := sendAndReceiveEchonetLiteFrame(targetIP, getFrame, responseTimeout)
		if err != nil {
			if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
				log.Printf("処理がタイムアウトしました (TID: %d)", tid)
			} else {
				log.Printf("ECHONET Lite 通信中にエラーが発生しました (TID: %d): %v", tid, err)
			}
			log.Println("監視サイクル終了 (エラー)")
			continue // エラーが発生しても次のサイクルへ
		}

		// --- 応答受信成功時の処理 ---
		log.Printf("正常に応答を受信しました (TID: %d, 送信元: %s)", tid, sourceAddr.String())

		_ = receivedData // 将来の使用のためにエラー抑制 (デシリアライズ処理で実際に使用します)

		// TODO: 受信したバイト列 (receivedData) を echonetlite.Frame にデシリアライズする処理を実装する
		// (この部分は変更なし、次のステップで実装します)

		log.Println("監視サイクル終了 (正常)")
	}
}

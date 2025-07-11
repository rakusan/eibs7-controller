package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"log/syslog"
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
	TargetIP                         string `toml:"target_ip"`
	MonitorIntervalSeconds           int    `toml:"monitor_interval_seconds"`
	ChargeStartTime                  string `toml:"charge_start_time"`
	ChargeEndTime                    string `toml:"charge_end_time"`
	ChargePowerUpdateIntervalMinutes int    `toml:"charge_power_update_interval_minutes"`
	AutoModeThresholdWatts           int    `toml:"auto_mode_threshold_watts"`
	ChargeModeThresholdWatts         int    `toml:"charge_mode_threshold_watts"`
	ModeChangeInhibitMinutes         int    `toml:"mode_change_inhibit_minutes"`
	LogMonitoringData                bool   `toml:"log_monitoring_data"`
}

// 設定ファイル名
const configFileName = "config.toml"

// setupLogger は、ログの出力先を標準出力とsyslogの両方に設定します。
func setupLogger() {
	// syslogライターを作成
	// 優先度は INFO、ファシリティは LOG_USER、タグは "eibs7-controller"
	syslogWriter, err := syslog.New(syslog.LOG_INFO|syslog.LOG_USER, "eibs7-controller")
	if err != nil {
		// syslogに接続できない場合でも、標準出力へのログは機能するように
		// log.Printf を使い、処理は続行する。
		log.Printf("警告: syslogへの接続に失敗しました: %v。ログは標準出力にのみ出力されます。", err)
		return
	}

	// 標準出力とsyslogの両方に書き込むMultiWriterを作成
	multiWriter := io.MultiWriter(os.Stdout, syslogWriter)

	// logパッケージのデフォルトロガーの出力先をMultiWriterに設定
	// これ以降、log.Printf などで出力したものは、両方に書き込まれる
	log.SetOutput(multiWriter)

	// ログのフォーマットに日付と時刻、短いファイル名（行番号付き）を含める
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	log.Println("ロガーの設定が完了しました。標準出力とsyslogの両方に出力します。")
}

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

	// ChargePowerUpdateIntervalMinutes のデフォルト値設定
	if config.ChargePowerUpdateIntervalMinutes <= 0 {
		log.Printf("設定ファイル '%s' の 'charge_power_update_interval_minutes' が未設定または0以下です。デフォルト値10分を使用します。", filePath)
		config.ChargePowerUpdateIntervalMinutes = 10
	}

	// ModeChangeInhibitMinutes のデフォルト値設定
	if config.ModeChangeInhibitMinutes <= 0 {
		log.Printf("設定ファイル '%s' の 'mode_change_inhibit_minutes' が未設定または0以下です。デフォルト値5分を使用します。", filePath)
		config.ModeChangeInhibitMinutes = 5
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

// decodeEDT は、指定されたEPCに基づいてEDT（プロパティ値データ）を適切なGoの型にデコードします。
// 対応していないEPCの場合は、元のバイト列とエラーを返します。
func decodeEDT(deoj echonetlite.EOJ, epc byte, edt []byte) (interface{}, string, error) {
	if edt == nil {
		// Get要求の応答でPDC=0の場合、EDTはnilになりうる。これはエラーではない。
		// ただし、値がないことを示すためにnilを返す。
		return nil, getPropertyName(deoj, epc), nil
	}
	pdc := len(edt)
	propName := getPropertyName(deoj, epc)

	switch deoj.ClassGroupCode {
	case 0x02: // 住宅設備関連機器クラスグループ
		switch deoj.ClassCode {
		case 0x7D: // 蓄電池クラス
			switch epc {
			case 0xE4: // 蓄電残量3 (%) - unsigned char (1 byte)
				if pdc != 1 {
					return edt, propName, fmt.Errorf("EPC 0xE4 (蓄電残量3) expects PDC=1, got %d", pdc)
				}
				return uint8(edt[0]), propName, nil
			case 0xDA: // 運転モード設定 - unsigned char (1 byte)
				if pdc != 1 {
					return edt, propName, fmt.Errorf("EPC 0xDA (運転モード設定) expects PDC=1, got %d", pdc)
				}
				return uint8(edt[0]), propName, nil // 具体的な値の意味は別途解釈
			case 0xEB: // 充電電力設定値 (W) - unsigned long (4 bytes)
				if pdc != 4 {
					return edt, propName, fmt.Errorf("EPC 0xEB (充電電力設定値) expects PDC=4, got %d", pdc)
				}
				return binary.BigEndian.Uint32(edt), propName, nil
			case 0xD3: // 瞬時充放電電力計測値 (W) - signed long (4 bytes)
				if pdc != 4 {
					return edt, propName, fmt.Errorf("EPC 0xD3 (瞬時充放電電力計測値) expects PDC=4, got %d", pdc)
				}
				return int32(binary.BigEndian.Uint32(edt)), propName, nil
			case 0xA0: // AC実効容量（充電） (Wh) - unsigned long (4 bytes)
				if pdc != 4 {
					return edt, propName, fmt.Errorf("EPC 0xA0 (AC実効容量) expects PDC=4, got %d", pdc)
				}
				return binary.BigEndian.Uint32(edt), propName, nil
			}
		case 0x79: // 住宅用太陽光発電クラス
			switch epc {
			case 0xE0: // 瞬時発電電力計測値 (W) - unsigned short (2 bytes)
				if pdc != 2 {
					return edt, propName, fmt.Errorf("EPC 0xE0 (瞬時発電電力計測値) expects PDC=2, got %d", pdc)
				}
				return binary.BigEndian.Uint16(edt), propName, nil
			}
		case 0x87: // 分電盤メータリングクラス
			switch epc {
			case 0xC6: // 瞬時電力計測値 (W) - signed long (4 bytes)
				if pdc != 4 {
					return edt, propName, fmt.Errorf("EPC 0xC6 (瞬時電力計測値) expects PDC=4, got %d", pdc)
				}
				return int32(binary.BigEndian.Uint32(edt)), propName, nil
			}
		case 0xA5: // マルチ入力PCSクラス
			switch epc {
			case 0xE7: // 瞬時電力計測値 (W) - signed long (4 bytes)
				if pdc != 4 {
					return edt, propName, fmt.Errorf("EPC 0xE7 (瞬時電力計測値) expects PDC=4, got %d", pdc)
				}
				return int32(binary.BigEndian.Uint32(edt)), propName, nil
			}
		}
	}
	// 未知のDEOJ/EPCの組み合わせ
	return edt, propName, fmt.Errorf("unknown DEOJ (ClassGroup: 0x%02X, Class: 0x%02X) or EPC 0x%X, cannot decode EDT, returning raw bytes", deoj.ClassGroupCode, deoj.ClassCode, epc)
}

// getPropertyName はEPCに対応するプロパティ名を返します。decodeEDTでPDC=0の場合などに使用。
func getPropertyName(deoj echonetlite.EOJ, epc byte) string {
	switch deoj.ClassGroupCode {
	case 0x02: // 住宅設備関連機器クラスグループ
		switch deoj.ClassCode {
		case 0x7D: // 蓄電池クラス
			switch epc {
			case 0xE4:
				return "蓄電残量3"
			case 0xDA:
				return "運転モード設定"
			case 0xEB:
				return "充電電力設定値"
			case 0xD3:
				return "瞬時充放電電力計測値"
			case 0xA0:
				return "AC実効容量（充電）"
			}
		case 0x79: // 住宅用太陽光発電クラス
			switch epc {
			case 0xE0:
				return "瞬時発電電力計測値"
			}
		case 0x87: // 分電盤メータリングクラス
			switch epc {
			case 0xC6:
				return "瞬時電力計測値"
			}
		case 0xA5: // マルチ入力PCSクラス
			switch epc {
			case 0xE7:
				return "瞬時電力計測値"
			}
		}
	}
	return fmt.Sprintf("不明なプロパティ (DEOJ: %02X%02X, EPC: %02X)", deoj.ClassGroupCode, deoj.ClassCode, epc)
}

// isChargingTime は、現在時刻が設定された充電時間帯内にあるかどうかを判定します。
func isChargingTime(startTimeStr, endTimeStr string) (bool, error) {
	const timeFormat = "15:04"
	now := time.Now()

	// 時刻部分のみを抽出
	currentTime, err := time.Parse(timeFormat, now.Format(timeFormat))
	if err != nil {
		return false, fmt.Errorf("現在時刻の解析に失敗しました: %w", err)
	}

	startTime, err := time.Parse(timeFormat, startTimeStr)
	if err != nil {
		return false, fmt.Errorf("開始時刻の解析に失敗しました ('%s'): %w", startTimeStr, err)
	}

	endTime, err := time.Parse(timeFormat, endTimeStr)
	if err != nil {
		return false, fmt.Errorf("終了時刻の解析に失敗しました ('%s'): %w", endTimeStr, err)
	}

	// 終了時刻が開始時刻より前の場合は、日付をまたぐ設定と判断
	if endTime.Before(startTime) {
		// 例: 23:00 - 02:00 の場合
		// (現在時刻 >= 開始時刻) OR (現在時刻 < 終了時刻)
		return !currentTime.Before(startTime) || currentTime.Before(endTime), nil
	} else {
		// 例: 09:00 - 15:00 の場合
		// (現在時刻 >= 開始時刻) AND (現在時刻 < 終了時刻)
		return !currentTime.Before(startTime) && currentTime.Before(endTime), nil
	}
}

func main() {
	// コマンドライン引数の定義
	loopCount := flag.Int("loop", -1, "監視ループの実行回数を指定します。-1の場合は無限に実行します。")
	flag.Parse()

	setupLogger() // ロガーを設定

	// --- 設定ファイルの読み込み ---
	cfg, err := loadConfig(configFileName)
	if err != nil {
		log.Fatalf("設定の読み込みに失敗しました: %v", err)
	}
	log.Printf("設定ファイル '%s' を読み込みました。", configFileName)
	log.Printf("  TargetIP: %s", cfg.TargetIP)
	log.Printf("  MonitorIntervalSeconds: %d", cfg.MonitorIntervalSeconds)
	log.Printf("  ChargeStartTime: %s", cfg.ChargeStartTime)
	log.Printf("  ChargeEndTime: %s", cfg.ChargeEndTime)
	log.Printf("  ChargePowerUpdateIntervalMinutes: %d", cfg.ChargePowerUpdateIntervalMinutes)
	log.Printf("  AutoModeThresholdWatts: %d", cfg.AutoModeThresholdWatts)
	log.Printf("  ChargeModeThresholdWatts: %d", cfg.ChargeModeThresholdWatts)
	log.Printf("  ModeChangeInhibitMinutes: %d", cfg.ModeChangeInhibitMinutes)
	log.Printf("  LogMonitoringData: %t", cfg.LogMonitoringData)

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
	var lastModeChangeTime time.Time
	var lastChargePowerIncreaseTime time.Time
	for i := 0; *loopCount == -1 || i < *loopCount; i++ {
		if i > 0 {
			<-ticker.C // 2回目以降はtickerを待つ
		}

		// 監視サイクルごとのデータを保持するマップ
		monitoringData := make(map[string]interface{})
		var surplusPower int32 // 余剰電力をループのスコープで定義
		var currentOperationMode byte

		log.Println("--------------------------------------------------")
		log.Println("監視サイクル開始")

		isChargingTimePeriod, err := isChargingTime(cfg.ChargeStartTime, cfg.ChargeEndTime)
		if err != nil {
			log.Printf("充電時間帯の判定に失敗しました: %v", err)
		} else {
			log.Printf("現在、充電時間帯です: %t", isChargingTimePeriod)
		}

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
					decodedValue, propName, err := decodeEDT(responseFrame.SEOJ, prop.EPC, prop.EDT)
					if err != nil {
						// デコードエラーが発生した場合でも、生データとエラー情報をログに出力
						log.Printf("[%s]   プロパティ: %s (EPC: 0x%X), PDC: %d, EDT: %X (TID: %d) - デコードエラー: %v", target.ObjectName, propName, prop.EPC, prop.PDC, prop.EDT, responseFrame.TID, err)
					} else if decodedValue == nil && prop.PDC == 0 { // PDC=0でEDTがnilの場合 (Get要求の正常な応答)
						log.Printf("[%s]   プロパティ: %s (EPC: 0x%X), PDC: %d, EDT: (なし) (TID: %d)", target.ObjectName, propName, prop.EPC, prop.PDC, responseFrame.TID)
					} else {
						log.Printf("[%s]   プロパティ: %s (EPC: 0x%X), PDC: %d, EDT: %X, 値: %v (TID: %d)", target.ObjectName, propName, prop.EPC, prop.PDC, prop.EDT, decodedValue, responseFrame.TID)
						// デコードした値をマップに保存
						monitoringData[fmt.Sprintf("%s.%s", target.ObjectName, propName)] = decodedValue

						// 現在の運転モードを更新
						if target.ObjectName == "蓄電池 (027D01)" && prop.EPC == 0xDA {
							if mode, ok := decodedValue.(uint8); ok {
								currentOperationMode = mode
							}
						}
					}
				}
			case echonetlite.ESVGet_SNA: // 0x52 - Property value read request error
				log.Printf("[%s] Getエラー応答を受信しました (TID: %d, ESV: 0x%X)", target.ObjectName, responseFrame.TID, responseFrame.ESV)
				// エラー応答の場合、Propertiesにエラーの原因を示す情報が含まれることがある (例: EPCが処理不可など)
			default:
				log.Printf("[%s] 予期しないESV (0x%X) を受信しました (TID: %d)", target.ObjectName, responseFrame.ESV, responseFrame.TID)
			}
		}

		// --- 計算値の算出 ---
		// 型アサーションで各値を取得
		gridPower, gOK := monitoringData["分電盤メータリング (028701).瞬時電力計測値"].(int32)
		pcsPower, pOK := monitoringData["マルチ入力PCS (02A501).瞬時電力計測値"].(int32)
		pvPower, pvOK := monitoringData["住宅用太陽光発電 (027901).瞬時発電電力計測値"].(uint16)

		if gOK && pOK && pvOK {
			// 自家消費電力 = 分電盤メータリング.瞬時電力計測値 - マルチ入力PCS.瞬時電力計測値
			selfConsumption := gridPower - pcsPower
			// 余剰電力 = 太陽光発電.瞬時発電電力計測値 - 自家消費電力
			surplusPower = int32(pvPower) - selfConsumption

			log.Printf("[計算値] 自家消費電力: %d W, 余剰電力: %d W", selfConsumption, surplusPower)
		} else {
			log.Println("[計算値] 計算に必要なデータが不足しているため、計算をスキップしました。")
		}

		// --- 制御ロジック --- 
		if isChargingTimePeriod {
			log.Println("[制御] 充電時間帯です。制御ロジックを実行します。")

			// 安全性: モード変更頻度抑制
			if !lastModeChangeTime.IsZero() && time.Since(lastModeChangeTime) < time.Duration(cfg.ModeChangeInhibitMinutes)*time.Minute {
				log.Printf("[制御] モード変更後、抑制時間が経過していないため（残り: %s）、制御をスキップします。", (time.Duration(cfg.ModeChangeInhibitMinutes)*time.Minute - time.Since(lastModeChangeTime)).Truncate(time.Second))
				continue
			}

			// 基本動作: 運転モードを「充電」に設定
			if currentOperationMode != 0x42 {
				err = setBatteryOperationMode(targetIP, 0x42, responseTimeout) // 0x42: 充電モード
				if err != nil {
					log.Printf("[制御] 蓄電池の運転モード設定（充電）に失敗しました: %v", err)
					// エラーが発生しても処理を続行
				}
			}

			// 買電抑制制御
			if surplusPower < int32(cfg.AutoModeThresholdWatts) {
				log.Printf("[制御] 余剰電力が閾値 (%d W) を下回ったため、運転モードを「自動」に設定します。", cfg.AutoModeThresholdWatts)
				if currentOperationMode != 0x46 {
					err = setBatteryOperationMode(targetIP, 0x46, responseTimeout) // 0x46: 自動モード
					if err != nil {
						log.Printf("[制御] 蓄電池の運転モード設定（自動）に失敗しました: %v", err)
					} else {
						lastModeChangeTime = time.Now()
					}
				}
			} else {
				log.Println("[制御] 余剰電力は閾値以上です。充電を継続します。")
			}

			// 目標充電量 (Wh) = AC実効容量(0xA0) * (1.0 - 蓄電残量3(0xE4) / 100.0)
			// 残り時間 (分) = 充電終了時刻 - 現在時刻
			// 目標充電電力 (W) = 目標充電量(Wh) * 60 / 残り時間(分) （ただし上限 5430W）

			// 必要なデータがmonitoringDataにあるか確認
			acCapacity, acOK := monitoringData["蓄電池 (027D01).AC実効容量（充電）"].(uint32)
			batteryRemaining, brOK := monitoringData["蓄電池 (027D01).蓄電残量3"].(uint8)

			if acOK && brOK {
				// 目標充電量 (Wh)
				targetChargeAmount := float64(acCapacity) * (1.0 - float64(batteryRemaining)/100.0)

				// 残り時間 (分) の計算
				const timeFormat = "15:04"
				now := time.Now()
				currentTime, _ := time.Parse(timeFormat, now.Format(timeFormat))
				chargeEndTime, _ := time.Parse(timeFormat, cfg.ChargeEndTime)

				remainingMinutes := chargeEndTime.Sub(currentTime).Minutes()
				if remainingMinutes <= 0 {
					log.Println("[制御] 充電終了時刻を過ぎているか、残り時間が0以下です。充電電力計算をスキップします。")
				} else {
					// 目標充電電力 (W)
					targetChargePower := int(targetChargeAmount * 60 / remainingMinutes)

					// 上限値の計算
					// 3000W と (余剰電力 - 500W) の小さい方を上限とする
					powerCap := int32(3000)
					if surplusPower-500 < powerCap {
						powerCap = surplusPower - 500
					}
					if powerCap < 0 {
						powerCap = 0
					}

					// 上限値を適用
					if targetChargePower > int(powerCap) {
						targetChargePower = int(powerCap)
					}

					log.Printf("[制御] 目標充電電力: %d W (目標充電量: %.2f Wh, 残り時間: %.2f 分)", targetChargePower, targetChargeAmount, remainingMinutes)

					// 現在の充電電力設定値を取得
					currentChargePower, cok := monitoringData["蓄電池 (027D01).充電電力設定値"].(uint32)

					if cok {
						if targetChargePower > int(currentChargePower) {
							// 引き上げの場合
							if time.Since(lastChargePowerIncreaseTime) < time.Duration(cfg.ChargePowerUpdateIntervalMinutes)*time.Minute {
								log.Printf("[制御] 充電電力の引き上げは、前回の引き上げから%d分経過するまで行えません（残り: %s）。", cfg.ChargePowerUpdateIntervalMinutes, (time.Duration(cfg.ChargePowerUpdateIntervalMinutes)*time.Minute - time.Since(lastChargePowerIncreaseTime)).Truncate(time.Second))
							} else {
								err = setBatteryChargePower(targetIP, targetChargePower, responseTimeout)
								if err != nil {
									log.Printf("[制御] 蓄電池の充電電力設定に失敗しました: %v", err)
								} else {
									lastChargePowerIncreaseTime = time.Now()
								}
							}
						} else if targetChargePower < int(currentChargePower) {
							// 引き下げの場合
							err = setBatteryChargePower(targetIP, targetChargePower, responseTimeout)
							if err != nil {
								log.Printf("[制御] 蓄電池の充電電力設定に失敗しました: %v", err)
							}
						} else {
							log.Println("[制御] 目標充電電力と現在の設定値が同じため、設定変更は行いません。")
						}
					} else {
						log.Println("[制御] 現在の充電電力設定値が取得できなかったため、充電電力の設定をスキップします。")
					}
				}
			} else {
				log.Println("[制御] 充電電力計算に必要なデータが不足しているため、計算をスキップしました。")
			}
		} else {
			log.Println("[制御] 充電時間帯ではありません。自動モードに設定します。")
			if currentOperationMode != 0x46 {
				err = setBatteryOperationMode(targetIP, 0x46, responseTimeout) // 0x46: 自動モード
				if err != nil {
					log.Printf("[制御] 蓄電池の運転モード設定に失敗しました: %v", err)
				}
			}
		}

		log.Println("監視サイクル終了 (全ターゲット処理完了)")
	}
}

// setBatteryOperationMode は蓄電池の運転モードを設定します。
func setBatteryOperationMode(targetIP string, mode byte, timeout time.Duration) error {
	setTID := getNextTID()
	log.Printf("[制御] 蓄電池の運転モードを 0x%X に設定します (TID: %d)", mode, setTID)

	setFrame := echonetlite.Frame{
		EHD1: echonetlite.EchonetLiteEHD1,
		EHD2: echonetlite.Format1,
		TID:  setTID,
		SEOJ: controllerEOJ,
		DEOJ: echonetlite.NewEOJ(0x02, 0x7D, 0x01), // 蓄電池
		ESV:  echonetlite.ESVSetC,                   // 0x61: SetC (応答要)
		OPC:  1,
		Properties: []echonetlite.Property{
			{
				EPC: 0xDA, // 運転モード設定
				PDC: 1,
				EDT: []byte{mode},
			},
		},
	}

	// --- フレームを送信し、応答を受信 ---
	receivedSetData, _, err := sendAndReceiveEchonetLiteFrame(targetIP, setFrame, timeout)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return fmt.Errorf("処理がタイムアウトしました (TID: %d): %w", setTID, err)
		} else {
			return fmt.Errorf("ECHONET Lite 通信中にエラーが発生しました (TID: %d): %w", setTID, err)
		}
	} else {
		// --- 応答受信成功時の処理 ---
		var responseSetFrame echonetlite.Frame
		err = responseSetFrame.UnmarshalBinary(receivedSetData)
		if err != nil {
			return fmt.Errorf("受信データのデシリアライズに失敗しました (TID: %d): %w", setTID, err)
		} else {
			// TID の一致確認
			if responseSetFrame.TID != setTID {
				log.Printf("[制御] 警告: 受信したTID (%d) が送信したTID (%d) と一致しません。", responseSetFrame.TID, setTID)
			}

			// ESV の確認
			switch responseSetFrame.ESV {
			case echonetlite.ESVSet_Res: // 0x71 - SetCの成功応答
				log.Printf("[制御] SetC応答(成功)を受信しました (TID: %d, ESV: 0x%X)", responseSetFrame.TID, responseSetFrame.ESV)
				return nil
			case echonetlite.ESVSetC_SNA: // 0x51 - SetCの失敗応答
				return fmt.Errorf("SetCエラー応答(失敗)を受信しました (TID: %d, ESV: 0x%X)", responseSetFrame.TID, responseSetFrame.ESV)
			default:
				return fmt.Errorf("予期しないESV (0x%X) を受信しました (TID: %d)", responseSetFrame.ESV, setTID)
			}
		}
	}
}

// setBatteryChargePower は蓄電池の充電電力設定値を設定します。
func setBatteryChargePower(targetIP string, power int, timeout time.Duration) error {
	setTID := getNextTID()
	log.Printf("[制御] 蓄電池の充電電力設定値を %d W に設定します (TID: %d)", power, setTID)

	// 電力値を4バイトのバイト列に変換
	powerBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(powerBytes, uint32(power))

	setFrame := echonetlite.Frame{
		EHD1: echonetlite.EchonetLiteEHD1,
		EHD2: echonetlite.Format1,
		TID:  setTID,
		SEOJ: controllerEOJ,
		DEOJ: echonetlite.NewEOJ(0x02, 0x7D, 0x01), // 蓄電池
		ESV:  echonetlite.ESVSetC,                   // 0x61: SetC (応答要)
		OPC:  1,
		Properties: []echonetlite.Property{
			{
				EPC: 0xEB, // 充電電力設定値
				PDC: 4,
				EDT: powerBytes,
			},
		},
	}

	// --- フレームを送信し、応答を受信 ---
	receivedSetData, _, err := sendAndReceiveEchonetLiteFrame(targetIP, setFrame, timeout)
	if err != nil {
		if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
			return fmt.Errorf("処理がタイムアウトしました (TID: %d): %w", setTID, err)
		} else {
			return fmt.Errorf("ECHONET Lite 通信中にエラーが発生しました (TID: %d): %w", setTID, err)
		}
	} else {
		// --- 応答受信成功時の処理 ---
		var responseSetFrame echonetlite.Frame
		err = responseSetFrame.UnmarshalBinary(receivedSetData)
		if err != nil {
			return fmt.Errorf("受信データのデシリアライズに失敗しました (TID: %d): %w", setTID, err)
		} else {
			// TID の一致確認
			if responseSetFrame.TID != setTID {
				log.Printf("[制御] 警告: 受信したTID (%d) が送信したTID (%d) と一致しません。", responseSetFrame.TID, setTID)
			}

			// ESV の確認
			switch responseSetFrame.ESV {
			case echonetlite.ESVSet_Res: // 0x71 - SetCの成功応答
				log.Printf("[制御] SetC応答(成功)を受信しました (TID: %d, ESV: 0x%X)", responseSetFrame.TID, responseSetFrame.ESV)
				return nil
			case echonetlite.ESVSetC_SNA: // 0x51 - SetCの失敗応答
				return fmt.Errorf("SetCエラー応答(失敗)を受信しました (TID: %d, ESV: 0x%X)", responseSetFrame.TID, responseSetFrame.ESV)
			default:
				return fmt.Errorf("予期しないESV (0x%X) を受信しました (TID: %d)", responseSetFrame.ESV, setTID)
			}
		}
	}
}

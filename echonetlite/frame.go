package echonetlite

import (
	"bytes"
	"encoding/binary"
	"fmt" // エラーメッセージ用
)

// Echonet Lite Header 1
type EHD1 byte

const (
	EchonetLiteEHD1 EHD1 = 0x10 // ECHONET Lite規格
)

// Echonet Lite Header 2
type EHD2 byte

const (
	Format1 EHD2 = 0x81 // 電文形式1 (規定電文形式)
	Format2 EHD2 = 0x82 // 電文形式2 (任意電文形式) - 今回は主にFormat1を使用
)

// Transaction ID
type TID uint16

// Echonet Lite Object (EOJ)
type EOJ struct {
	ClassGroupCode byte
	ClassCode      byte
	InstanceCode   byte
}

// Helper function to create a new EOJ
func NewEOJ(classGroup, class, instance byte) EOJ {
	return EOJ{
		ClassGroupCode: classGroup,
		ClassCode:      class,
		InstanceCode:   instance,
	}
}

// Echonet Lite Service (ESV)
type ESV byte

// Property represents EPC, PDC, EDT
type Property struct {
	EPC byte   // Echonet Property Code
	PDC byte   // Property Data Counter (length of EDT)
	EDT []byte // Property Value Data
}

// Echonet Lite Frame
type Frame struct {
	EHD1 EHD1
	EHD2 EHD2
	TID  TID
	SEOJ EOJ // Source Echonet Lite Object
	DEOJ EOJ // Destination Echonet Lite Object
	ESV  ESV
	OPC  byte // Operation Property Counter
	// OPCSet byte // For SetGet ESV (0x6E, 0x7E, 0x5E) - Not implemented in this version
	// OPCGet byte // For SetGet ESV (0x6E, 0x7E, 0x5E) - Not implemented in this version
	Properties []Property
}

// ESV constants
const (
	// Requests
	ESVSetI   ESV = 0x60 // Property value write request (no response required) / SetI
	ESVSetC   ESV = 0x61 // Property value write request (response required) / SetC
	ESVGet    ESV = 0x62 // Property value read request (response required) / Get
	ESVInfReq ESV = 0x63 // Property value notification request / INF_REQ
	ESVSetGet ESV = 0x6E // Property value write & read request / SetGet

	// Responses / Notifications
	ESVSet_Res    ESV = 0x71 // Property value write response
	ESVGet_Res    ESV = 0x72 // Property value read response
	ESVInf        ESV = 0x73 // Property value notification
	ESVInfC       ESV = 0x74 // Property value notification (response required)
	ESVSetGet_Res ESV = 0x7E // Property value write & read response
	ESVInfC_Res   ESV = 0x7A // Property value notification response (response to INFC) / INFC_Res

	// Error Responses
	ESVSetI_SNA   ESV = 0x50 // Error response to SetI (Property value write request, no response required)
	ESVSetC_SNA   ESV = 0x51 // Error response to SetC (Property value write request, response required)
	ESVGet_SNA    ESV = 0x52 // Error response to Get (Property value read request)
	ESVInf_SNA    ESV = 0x53 // Error response to INF_REQ (Property value notification request) / INF_SNA
	ESVSetGet_SNA ESV = 0x5E // Error response to SetGet (Property value write & read request)
)

// MarshalBinary は Frame 構造体を ECHONET Lite フレームのバイト列にシリアライズします。
// encoding.BinaryMarshaler インターフェースを実装します。
func (f *Frame) MarshalBinary() ([]byte, error) {
	// ECHONET Lite フレームの最小サイズはヘッダ(4) + EOJ(6) + ESV(1) + OPC(1) = 12 バイト
	// プロパティのサイズを考慮して初期バッファサイズを推定（最適化の余地あり）
	estimatedSize := 12
	for _, prop := range f.Properties {
		estimatedSize += 1 + 1 + int(prop.PDC) // EPC + PDC + EDT size
	}
	buf := bytes.NewBuffer(make([]byte, 0, estimatedSize))

	// 1. EHD1 (1 byte) - 固定値 0x10
	if err := buf.WriteByte(byte(EchonetLiteEHD1)); err != nil {
		return nil, fmt.Errorf("failed to write EHD1: %w", err)
	}

	// 2. EHD2 (1 byte) - 通常は Format1 (0x81)
	if err := buf.WriteByte(byte(f.EHD2)); err != nil {
		return nil, fmt.Errorf("failed to write EHD2: %w", err)
	}
	// TODO: Format2 (0x82) の場合の処理は未実装

	// 3. TID (2 bytes, Big Endian)
	tidBytes := make([]byte, 2)
	binary.BigEndian.PutUint16(tidBytes, uint16(f.TID))
	if _, err := buf.Write(tidBytes); err != nil {
		return nil, fmt.Errorf("failed to write TID: %w", err)
	}

	// 4. SEOJ (3 bytes)
	if err := buf.WriteByte(f.SEOJ.ClassGroupCode); err != nil {
		return nil, fmt.Errorf("failed to write SEOJ ClassGroupCode: %w", err)
	}
	if err := buf.WriteByte(f.SEOJ.ClassCode); err != nil {
		return nil, fmt.Errorf("failed to write SEOJ ClassCode: %w", err)
	}
	if err := buf.WriteByte(f.SEOJ.InstanceCode); err != nil {
		return nil, fmt.Errorf("failed to write SEOJ InstanceCode: %w", err)
	}

	// 5. DEOJ (3 bytes)
	if err := buf.WriteByte(f.DEOJ.ClassGroupCode); err != nil {
		return nil, fmt.Errorf("failed to write DEOJ ClassGroupCode: %w", err)
	}
	if err := buf.WriteByte(f.DEOJ.ClassCode); err != nil {
		return nil, fmt.Errorf("failed to write DEOJ ClassCode: %w", err)
	}
	if err := buf.WriteByte(f.DEOJ.InstanceCode); err != nil {
		return nil, fmt.Errorf("failed to write DEOJ InstanceCode: %w", err)
	}

	// 6. ESV (1 byte)
	if err := buf.WriteByte(byte(f.ESV)); err != nil {
		return nil, fmt.Errorf("failed to write ESV: %w", err)
	}

	// 7. OPC (Operation Property Count) (1 byte)
	// OPC の値が Properties スライスの要素数と一致するかチェック
	if f.OPC != byte(len(f.Properties)) {
		// 開発中は警告を出すなどしても良いが、基本的には呼び出し側が正しく設定する責務
		// fmt.Printf("Warning: OPC mismatch: Frame.OPC=%d, len(Frame.Properties)=%d. Using Frame.OPC.\n", f.OPC, len(f.Properties))
		// return nil, fmt.Errorf("OPC mismatch: Frame.OPC=%d, len(Frame.Properties)=%d", f.OPC, len(f.Properties))
	}
	if err := buf.WriteByte(f.OPC); err != nil {
		return nil, fmt.Errorf("failed to write OPC: %w", err)
	}
	// TODO: ESV が SetGet (0x6E, 0x7E, 0x5E) の場合、OPCSet/OPCGet の処理が必要

	// 8. Properties (Variable length)
	for i, prop := range f.Properties {
		// 8a. EPC (Echonet Property Code) (1 byte)
		if err := buf.WriteByte(prop.EPC); err != nil {
			return nil, fmt.Errorf("failed to write EPC for property %d: %w", i, err)
		}
		// 8b. PDC (Property Data Counter) (1 byte)
		// PDC の値が EDT の長さと一致するかチェック
		if prop.PDC != byte(len(prop.EDT)) {
			// 開発中は警告を出すなどしても良いが、基本的には呼び出し側が正しく設定する責務
			// fmt.Printf("Warning: PDC mismatch for property %d (EPC: 0x%X): Property.PDC=%d, len(Property.EDT)=%d. Using Property.PDC.\n", i, prop.EPC, prop.PDC, len(prop.EDT))
			// return nil, fmt.Errorf("PDC mismatch for property %d (EPC: 0x%X): Property.PDC=%d, len(Property.EDT)=%d", i, prop.EPC, prop.PDC, len(prop.EDT))
		}
		if err := buf.WriteByte(prop.PDC); err != nil {
			return nil, fmt.Errorf("failed to write PDC for property %d: %w", i, err)
		}
		// 8c. EDT (Property Value Data) (prop.PDC bytes)
		if prop.PDC > 0 {
			// EDT の実際の長さが PDC 以上であることを確認 (PDC 分だけ書き込むため)
			if len(prop.EDT) < int(prop.PDC) {
				return nil, fmt.Errorf("EDT length is less than PDC for property %d (EPC: 0x%X): PDC=%d, len(EDT)=%d", i, prop.EPC, prop.PDC, len(prop.EDT))
			}
			// PDC で指定されたバイト数だけ書き込む
			if _, err := buf.Write(prop.EDT[:prop.PDC]); err != nil {
				return nil, fmt.Errorf("failed to write EDT for property %d: %w", i, err)
			}
		}
	}

	return buf.Bytes(), nil
}

// UnmarshalBinary は ECHONET Lite フレームのバイト列を Frame 構造体にデシリアライズします。
// encoding.BinaryUnmarshaler インターフェースを実装します。
func (f *Frame) UnmarshalBinary(data []byte) error {
	// ECHONET Lite フレームの最小サイズはヘッダ(4) + EOJ(6) + ESV(1) + OPC(1) = 12 バイト
	// (プロパティがない場合)
	minLength := 12
	if len(data) < minLength {
		return fmt.Errorf("data too short for ECHONET Lite frame: got %d bytes, want at least %d", len(data), minLength)
	}

	reader := bytes.NewReader(data)

	// 1. EHD1 (1 byte)
	ehd1Byte, err := reader.ReadByte()
	if err != nil {
		return fmt.Errorf("failed to read EHD1: %w", err)
	}
	f.EHD1 = EHD1(ehd1Byte)
	if f.EHD1 != EchonetLiteEHD1 {
		return fmt.Errorf("invalid EHD1: expected 0x%X, got 0x%X", EchonetLiteEHD1, f.EHD1)
	}

	// 2. EHD2 (1 byte)
	ehd2Byte, err := reader.ReadByte()
	if err != nil {
		return fmt.Errorf("failed to read EHD2: %w", err)
	}
	f.EHD2 = EHD2(ehd2Byte)
	// TODO: Format2 (0x82) の場合の処理は未実装 (主に Format1 を想定)
	if f.EHD2 != Format1 {
		// 厳密にはエラーではないが、この実装では Format1 のみを想定
		// fmt.Printf("Warning: EHD2 is not Format1 (0x81), got 0x%X. Parsing as Format1.\n", f.EHD2)
	}

	// 3. TID (2 bytes, Big Endian)
	var tidVal uint16
	if err := binary.Read(reader, binary.BigEndian, &tidVal); err != nil {
		return fmt.Errorf("failed to read TID: %w", err)
	}
	f.TID = TID(tidVal)

	// 4. SEOJ (3 bytes)
	seojBytes := make([]byte, 3)
	if _, err := reader.Read(seojBytes); err != nil {
		return fmt.Errorf("failed to read SEOJ: %w", err)
	}
	f.SEOJ = NewEOJ(seojBytes[0], seojBytes[1], seojBytes[2])

	// 5. DEOJ (3 bytes)
	deojBytes := make([]byte, 3)
	if _, err := reader.Read(deojBytes); err != nil {
		return fmt.Errorf("failed to read DEOJ: %w", err)
	}
	f.DEOJ = NewEOJ(deojBytes[0], deojBytes[1], deojBytes[2])

	// 6. ESV (1 byte)
	esvByte, err := reader.ReadByte()
	if err != nil {
		return fmt.Errorf("failed to read ESV: %w", err)
	}
	f.ESV = ESV(esvByte)

	// 7. OPC (Operation Property Counter) (1 byte)
	opcByte, err := reader.ReadByte()
	if err != nil {
		return fmt.Errorf("failed to read OPC: %w", err)
	}
	f.OPC = opcByte
	// TODO: ESV が SetGet (0x6E, 0x7E, 0x5E) の場合、OPCSet/OPCGet の処理が必要

	// 8. Properties (Variable length)
	f.Properties = make([]Property, 0, f.OPC)
	for i := 0; i < int(f.OPC); i++ {
		var prop Property
		// 8a. EPC (Echonet Property Code) (1 byte)
		epcByte, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read EPC for property %d: %w", i, err)
		}
		prop.EPC = epcByte

		// 8b. PDC (Property Data Counter) (1 byte)
		pdcByte, err := reader.ReadByte()
		if err != nil {
			return fmt.Errorf("failed to read PDC for property %d: %w", i, err)
		}
		prop.PDC = pdcByte

		// 8c. EDT (Property Value Data) (prop.PDC bytes)
		if prop.PDC > 0 {
			prop.EDT = make([]byte, prop.PDC)
			if _, err := reader.Read(prop.EDT); err != nil {
				return fmt.Errorf("failed to read EDT for property %d (EPC: 0x%X, PDC: %d): %w", i, prop.EPC, prop.PDC, err)
			}
		} else {
			prop.EDT = nil // PDC が 0 の場合は EDT は空
		}
		f.Properties = append(f.Properties, prop)
	}

	// OPC で指定されたプロパティ数と実際に読み込めたプロパティ数が一致するか確認
	if len(f.Properties) != int(f.OPC) {
		// 通常はループ条件で担保されるが、念のため
		return fmt.Errorf("property count mismatch: OPC specified %d, but read %d properties", f.OPC, len(f.Properties))
	}

	// すべてのデータを読み込んだ後、Readerに余分なデータがないか確認 (オプション)
	// if reader.Len() > 0 {
	// 	return fmt.Errorf("trailing data in frame: %d bytes remaining", reader.Len())
	// }

	return nil
}

// --- Example Usage (for testing, can be placed in a _test.go file or temporarily in main) ---
/*
func main() {
	// Example Get frame: Get 蓄電池(027D01)の蓄電残量3(E4)
	frame := Frame{
		EHD1: EchonetLiteEHD1, // 0x10
		EHD2: Format1,        // 0x81
		TID:  1,               // Transaction ID 1
		SEOJ: NewEOJ(0x05, 0xFF, 0x01), // Controller object
		DEOJ: NewEOJ(0x02, 0x7D, 0x01), // Storage Battery object
		ESV:  ESVGet,          // 0x62 (Get)
		OPC:  1,               // 1 property
		Properties: []Property{
			{EPC: 0xE4, PDC: 0x00, EDT: nil}, // EPC: Remaining capacity 3, PDC: 0 (no data for Get)
		},
	}

	serializedData, err := frame.MarshalBinary()
	if err != nil {
		fmt.Println("Error serializing frame:", err)
		return
	}

	fmt.Printf("Serialized data (hex): %X\n", serializedData)
	// Expected output: 1081000105FF01027D016201E400

	// Test UnmarshalBinary
	var receivedFrame Frame
	// Example received data (Get_Res for 蓄電残量3(E4) = 50% (0x32))
	// 10 81 0001 027D01 05FF01 72 01 E4 01 32
	// EHD1 EHD2 TID   DEOJ   SEOJ   ESV OPC EPC PDC EDT
	receivedBytes := []byte{0x10, 0x81, 0x00, 0x01, 0x02, 0x7D, 0x01, 0x05, 0xFF, 0x01, 0x72, 0x01, 0xE4, 0x01, 0x32}
	err = receivedFrame.UnmarshalBinary(receivedBytes)
	if err != nil {
		fmt.Println("Error unmarshaling frame:", err)
		return
	}
	fmt.Printf("Unmarshaled frame: %+v\n", receivedFrame)
	fmt.Printf("  Property 0: EPC=0x%X, PDC=%d, EDT=%X\n", receivedFrame.Properties[0].EPC, receivedFrame.Properties[0].PDC, receivedFrame.Properties[0].EDT)

}
*/

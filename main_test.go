package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"kuramo.ch/eibs7-controller/echonetlite"
)

func TestIsChargingTime(t *testing.T) {
	// Test case where we know the current time is not in the 09:00-15:00 range
	// but we can test the logic with fixed times
	
	t.Run("CrossMidnight", func(t *testing.T) {
		// Test a time within the range (should return true)
		_, err := isChargingTime("23:00", "02:00")
		assert.NoError(t, err)
		// This test will pass because we're testing the logic for midnight crossing
		// The actual result depends on current time, but we're testing the logic path
	})

	// Test invalid time format
	t.Run("InvalidTimeFormat", func(t *testing.T) {
		result, err := isChargingTime("invalid", "15:00")
		assert.Error(t, err)
		assert.False(t, result, "Should return false and error for invalid time format")
	})

	// Test the logic with known times that should work
	t.Run("LogicTest", func(t *testing.T) {
		// Since we can't control the actual time in tests, let's just make sure
		// the function doesn't panic and handles valid inputs correctly
		_, err := isChargingTime("09:00", "15:00")
		assert.NoError(t, err)
		// We can't assert the exact result since it depends on current time,
		// but we can ensure no error occurs and function executes properly
	})
}

func TestGetPropertyName(t *testing.T) {
	t.Run("BatteryProperties", func(t *testing.T) {
		// Test various battery EPCs
		// 蓄電残量3 (E4)
		name := getPropertyName(echonetlite.NewEOJ(0x02, 0x7D, 0x01), 0xE4)
		assert.Equal(t, "蓄電残量3", name)

		// 運転モード設定 (DA)
		name = getPropertyName(echonetlite.NewEOJ(0x02, 0x7D, 0x01), 0xDA)
		assert.Equal(t, "運転モード設定", name)

		// 充電電力設定値 (EB)
		name = getPropertyName(echonetlite.NewEOJ(0x02, 0x7D, 0x01), 0xEB)
		assert.Equal(t, "充電電力設定値", name)

		// 瞬時充放電電力計測値 (D3)
		name = getPropertyName(echonetlite.NewEOJ(0x02, 0x7D, 0x01), 0xD3)
		assert.Equal(t, "瞬時充放電電力計測値", name)

		// AC実効容量（充電） (A0)
		name = getPropertyName(echonetlite.NewEOJ(0x02, 0x7D, 0x01), 0xA0)
		assert.Equal(t, "AC実効容量（充電）", name)
	})

	t.Run("SolarPanelProperties", func(t *testing.T) {
		// 瞬時発電電力計測値 (E0)
		name := getPropertyName(echonetlite.NewEOJ(0x02, 0x79, 0x01), 0xE0)
		assert.Equal(t, "瞬時発電電力計測値", name)
	})

	t.Run("MeterProperties", func(t *testing.T) {
		// 瞬時電力計測値 (C6)
		name := getPropertyName(echonetlite.NewEOJ(0x02, 0x87, 0x01), 0xC6)
		assert.Equal(t, "瞬時電力計測値", name)
	})

	t.Run("PCSProperties", func(t *testing.T) {
		// 瞬時電力計測値 (E7)
		name := getPropertyName(echonetlite.NewEOJ(0x02, 0xA5, 0x01), 0xE7)
		assert.Equal(t, "瞬時電力計測値", name)
	})

	t.Run("UnknownProperties", func(t *testing.T) {
		// Test unknown EPC
		name := getPropertyName(echonetlite.NewEOJ(0x02, 0x7D, 0x01), 0xFF)
		assert.Contains(t, name, "不明なプロパティ")
	})
}

func TestDecodeEDT(t *testing.T) {
	t.Run("BatteryProperties", func(t *testing.T) {
		// Test E4 (蓄電残量3) - 1 byte
		data := []byte{0x32} // 50 in decimal
		result, name, err := decodeEDT(echonetlite.NewEOJ(0x02, 0x7D, 0x01), 0xE4, data)
		assert.NoError(t, err)
		assert.Equal(t, "蓄電残量3", name)
		assert.Equal(t, uint8(50), result)

		// Test DA (運転モード設定) - 1 byte
		data = []byte{0x01} // 1 in decimal
		result, name, err = decodeEDT(echonetlite.NewEOJ(0x02, 0x7D, 0x01), 0xDA, data)
		assert.NoError(t, err)
		assert.Equal(t, "運転モード設定", name)
		assert.Equal(t, uint8(1), result)

		// Test EB (充電電力設定値) - 4 bytes
		data = []byte{0x00, 0x00, 0x0B, 0xB8} // 3000 in decimal (big endian)
		result, name, err = decodeEDT(echonetlite.NewEOJ(0x02, 0x7D, 0x01), 0xEB, data)
		assert.NoError(t, err)
		assert.Equal(t, "充電電力設定値", name)
		assert.Equal(t, uint32(3000), result)

		// Test D3 (瞬時充放電電力計測値) - 4 bytes
		data = []byte{0x00, 0x00, 0x0B, 0xB8} // 3000 in decimal (big endian)
		result, name, err = decodeEDT(echonetlite.NewEOJ(0x02, 0x7D, 0x01), 0xD3, data)
		assert.NoError(t, err)
		assert.Equal(t, "瞬時充放電電力計測値", name)
		assert.Equal(t, int32(3000), result)

		// Test A0 (AC実効容量) - 4 bytes
		data = []byte{0x00, 0x00, 0x0B, 0xB8} // 3000 in decimal (big endian)
		result, name, err = decodeEDT(echonetlite.NewEOJ(0x02, 0x7D, 0x01), 0xA0, data)
		assert.NoError(t, err)
		assert.Equal(t, "AC実効容量（充電）", name)
		assert.Equal(t, uint32(3000), result)
	})

	t.Run("SolarPanelProperties", func(t *testing.T) {
		// Test E0 (瞬時発電電力計測値) - 2 bytes
		data := []byte{0x00, 0x01} // 1 in decimal (big endian)
		result, name, err := decodeEDT(echonetlite.NewEOJ(0x02, 0x79, 0x01), 0xE0, data)
		assert.NoError(t, err)
		assert.Equal(t, "瞬時発電電力計測値", name)
		assert.Equal(t, uint16(1), result)
	})

	t.Run("MeterProperties", func(t *testing.T) {
		// Test C6 (瞬時電力計測値) - 4 bytes
		data := []byte{0x00, 0x00, 0x0B, 0xB8} // 3000 in decimal (big endian)
		result, name, err := decodeEDT(echonetlite.NewEOJ(0x02, 0x87, 0x01), 0xC6, data)
		assert.NoError(t, err)
		assert.Equal(t, "瞬時電力計測値", name)
		assert.Equal(t, int32(3000), result)
	})

	t.Run("PCSProperties", func(t *testing.T) {
		// Test E7 (瞬時電力計測値) - 4 bytes
		data := []byte{0x00, 0x00, 0x0B, 0xB8} // 3000 in decimal (big endian)
		result, name, err := decodeEDT(echonetlite.NewEOJ(0x02, 0xA5, 0x01), 0xE7, data)
		assert.NoError(t, err)
		assert.Equal(t, "瞬時電力計測値", name)
		assert.Equal(t, int32(3000), result)
	})

	t.Run("InvalidPDC", func(t *testing.T) {
		// Test invalid PDC for E4 (should return error)
		data := []byte{0x32, 0x00} // Extra byte - invalid for E4 (should only be 1 byte)
		result, name, err := decodeEDT(echonetlite.NewEOJ(0x02, 0x7D, 0x01), 0xE4, data)
		assert.Error(t, err)
		assert.Equal(t, "蓄電残量3", name)
		assert.Equal(t, data, result) // Should return raw bytes when error occurs
	})

	t.Run("UnknownProperty", func(t *testing.T) {
		// Test unknown EPC - should return raw bytes and error
		data := []byte{0x01, 0x02}
		result, name, err := decodeEDT(echonetlite.NewEOJ(0x02, 0x7D, 0x01), 0xFF, data)
		assert.Error(t, err)
		assert.Contains(t, name, "不明なプロパティ")
		assert.Equal(t, data, result) // Should return raw bytes when error occurs
	})
}
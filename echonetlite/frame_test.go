package echonetlite

import (
    "reflect"
    "testing"
)

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
    original := Frame{
        EHD1: EchonetLiteEHD1,
        EHD2: Format1,
        TID:  0x1234,
        SEOJ: NewEOJ(0x05, 0xFF, 0x01),
        DEOJ: NewEOJ(0x02, 0x7D, 0x01),
        ESV:  ESVGet,
        OPC:  2,
        Properties: []Property{{
            EPC: 0xE4,
            PDC: 0x00,
            EDT: nil,
        }, {
            EPC: 0x80,
            PDC: 0x01,
            EDT: []byte{0x55},
        }},
    }

    data, err := original.MarshalBinary()
    if err != nil {
        t.Fatalf("MarshalBinary failed: %v", err)
    }

    var decoded Frame
    if err := decoded.UnmarshalBinary(data); err != nil {
        t.Fatalf("UnmarshalBinary failed: %v", err)
    }

    // Compare all fields except slice backing differences using reflect.DeepEqual
    if !reflect.DeepEqual(original, decoded) {
        t.Errorf("Round‑trip mismatch.\nOriginal: %+v\nDecoded : %+v", original, decoded)
    }

    // Ensure that the binary representation matches expected length (header + properties)
    expectedLen := 12 + int(original.OPC)*(1+1) // EPC+PDC per property
    for _, p := range original.Properties {
        expectedLen += int(p.PDC)
    }
    if len(data) != expectedLen {
        t.Errorf("Unexpected serialized length: got %d, want %d", len(data), expectedLen)
    }
}

func TestMarshalBinaryExpected(t *testing.T) {
    // Example Get frame from documentation
    frame := Frame{
        EHD1: EchonetLiteEHD1,
        EHD2: Format1,
        TID:  0x0001,
        SEOJ: NewEOJ(0x05, 0xFF, 0x01),
        DEOJ: NewEOJ(0x02, 0x7D, 0x01),
        ESV:  ESVGet,
        OPC:  1,
        Properties: []Property{{EPC: 0xE4, PDC: 0x00, EDT: nil}},
    }
    data, err := frame.MarshalBinary()
    if err != nil {
        t.Fatalf("MarshalBinary failed: %v", err)
    }
    // Expected byte sequence as per example in source code comments
    expected := []byte{0x10, 0x81, 0x00, 0x01, 0x05, 0xFF, 0x01, 0x02, 0x7D, 0x01, 0x62, 0x01, 0xE4, 0x00}
    if !reflect.DeepEqual(data, expected) {
        t.Errorf("MarshalBinary output mismatch.\nGot:      % X\nExpected: % X", data, expected)
    }
}

func TestUnmarshalBinaryExpected(t *testing.T) {
    // Example Get_Res frame from documentation (remaining capacity 50% -> EDT 0x32)
    raw := []byte{0x10, 0x81, 0x00, 0x01, 0x02, 0x7D, 0x01, 0x05, 0xFF, 0x01, 0x72, 0x01, 0xE4, 0x01, 0x32}
    var f Frame
    if err := f.UnmarshalBinary(raw); err != nil {
        t.Fatalf("UnmarshalBinary failed: %v", err)
    }
    // Verify fields
    if f.EHD1 != EchonetLiteEHD1 || f.EHD2 != Format1 || f.TID != 0x0001 {
        t.Errorf("Header fields mismatch: %+v", f)
    }
    if !reflect.DeepEqual(f.SEOJ, NewEOJ(0x05, 0xFF, 0x01)) {
        t.Errorf("SEOJ mismatch: got %v", f.SEOJ)
    }
    if !reflect.DeepEqual(f.DEOJ, NewEOJ(0x02, 0x7D, 0x01)) {
        t.Errorf("DEOJ mismatch: got %v", f.DEOJ)
    }
    if f.ESV != ESVGet_Res { // raw has 0x72 which is ESVGet_Res
        t.Errorf("ESV mismatch: got %X want %X", f.ESV, ESVGet_Res)
    }
    if f.OPC != 1 {
        t.Errorf("OPC mismatch: got %d", f.OPC)
    }
    if len(f.Properties) != 1 {
        t.Fatalf("Properties count mismatch: %d", len(f.Properties))
    }
    p := f.Properties[0]
    if p.EPC != 0xE4 || p.PDC != 0x01 || !reflect.DeepEqual(p.EDT, []byte{0x32}) {
        t.Errorf("Property mismatch: %+v", p)
    }
}


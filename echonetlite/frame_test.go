package echonetlite

import (
    "bytes"
    "reflect"
    "testing"
)

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
    // Construct a sample frame with one property (Get request)
    original := Frame{
        EHD1: EchonetLiteEHD1,
        EHD2: Format1,
        TID:  0x1234,
        SEOJ: NewEOJ(0x05, 0xFF, 0x01), // Controller object
        DEOJ: NewEOJ(0x02, 0x7D, 0x01), // Storage Battery object
        ESV:  ESVGet,
        OPC:  1,
        Properties: []Property{{
            EPC: 0xE4,
            PDC: 0x00,
            EDT: nil,
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

    // Compare the two frames (excluding slice backing array differences)
    if !reflect.DeepEqual(original, decoded) {
        t.Errorf("Round‑trip mismatch.\nOriginal: %+v\nDecoded : %+v", original, decoded)
    }

    // Additionally ensure that re‑marshalling yields identical bytes
    data2, err := decoded.MarshalBinary()
    if err != nil {
        t.Fatalf("MarshalBinary on decoded failed: %v", err)
    }
    if !bytes.Equal(data, data2) {
        t.Errorf("Marshalled bytes differ after round‑trip.\nFirst:  % X\nSecond: % X", data, data2)
    }
}

func TestMarshalExpectedBytes(t *testing.T) {
    // Expected byte sequence for a Get request frame with no data
    expected := []byte{0x10, 0x81, 0x12, 0x34, 0x05, 0xFF, 0x01, 0x02, 0x7D, 0x01, 0x62, 0x01, 0xE4, 0x00}
    frame := Frame{
        EHD1: EchonetLiteEHD1,
        EHD2: Format1,
        TID:  0x1234,
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
    if !bytes.Equal(data, expected) {
        t.Errorf("Marshaled bytes mismatch.\nGot:      % X\nExpected: % X", data, expected)
    }
}

func TestUnmarshalFromBytes(t *testing.T) {
    // Byte sequence representing a Get response with value 0x32 (50%)
    raw := []byte{0x10, 0x81, 0x00, 0x01, 0x02, 0x7D, 0x01, 0x05, 0xFF, 0x01, 0x72, 0x01, 0xE4, 0x01, 0x32}
    var f Frame
    if err := f.UnmarshalBinary(raw); err != nil {
        t.Fatalf("UnmarshalBinary failed: %v", err)
    }
    if f.EHD1 != EchonetLiteEHD1 || f.EHD2 != Format1 || f.TID != 0x0001 {
        t.Errorf("Header fields incorrect: %+v", f)
    }
    if len(f.Properties) != 1 {
        t.Fatalf("Expected 1 property, got %d", len(f.Properties))
    }
    p := f.Properties[0]
    if p.EPC != 0xE4 || p.PDC != 0x01 || !bytes.Equal(p.EDT, []byte{0x32}) {
        t.Errorf("Property mismatch: %+v", p)
    }
}


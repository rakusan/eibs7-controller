package echonetlite

import (
    "reflect"
    "testing"
)

func TestMarshalUnmarshalRoundTrip(t *testing.T) {
    // existing test remains
    original := Frame{
        EHD1: EchonetLiteEHD1,
        EHD2: Format1,
        TID:  0x1234,
        SEOJ: NewEOJ(0x05, 0xFF, 0x01),
        DEOJ: NewEOJ(0x02, 0x7D, 0x01),
        ESV:  ESVGet,
        OPC:  2,
        Properties: []Property{{EPC: 0xE4, PDC: 1, EDT: []byte{0x32}}, {EPC: 0xE5, PDC: 2, EDT: []byte{0x01, 0x02}}},
    }

    data, err := original.MarshalBinary()
    if err != nil {
        t.Fatalf("MarshalBinary failed: %v", err)
    }

    var decoded Frame
    if err := decoded.UnmarshalBinary(data); err != nil {
        t.Fatalf("UnmarshalBinary failed: %v", err)
    }

    // Compare all fields except slice order which should be identical
    if !reflect.DeepEqual(original, decoded) {
        t.Errorf("Round‑trip mismatch.\nOriginal: %+v\nDecoded : %+v", original, decoded)
    }
}

func TestMarshalByteSequence(t *testing.T) {
    // Known frame and expected byte sequence
    frame := Frame{
        EHD1: EchonetLiteEHD1,
        EHD2: Format1,
        TID:  0x0001,
        SEOJ: NewEOJ(0x05, 0xFF, 0x01),
        DEOJ: NewEOJ(0x02, 0x7D, 0x01),
        ESV:  ESVGet,
        OPC:  1,
        Properties: []Property{{EPC: 0xE4, PDC: 0, EDT: nil}},
    }
    expected := []byte{0x10, 0x81, 0x00, 0x01, 0x05, 0xFF, 0x01, 0x02, 0x7D, 0x01, 0x62, 0x01, 0xE4, 0x00}
    data, err := frame.MarshalBinary()
    if err != nil {
        t.Fatalf("MarshalBinary failed: %v", err)
    }
    if !reflect.DeepEqual(data, expected) {
        t.Errorf("Marshaled bytes mismatch.\nGot:      % X\nExpected: % X", data, expected)
    }
}

func TestUnmarshalFrame(t *testing.T) {
    // Byte sequence for a Get_Res frame with one property (E4=0x32)
    raw := []byte{0x10, 0x81, 0x00, 0x01, 0x05, 0xFF, 0x01, 0x02, 0x7D, 0x01, 0x72, 0x01, 0xE4, 0x01, 0x32}
    var f Frame
    if err := f.UnmarshalBinary(raw); err != nil {
        t.Fatalf("UnmarshalBinary failed: %v", err)
    }
    // Verify fields
    if f.EHD1 != EchonetLiteEHD1 || f.EHD2 != Format1 || f.TID != 0x0001 {
        t.Errorf("Header fields incorrect: %+v", f)
    }
    if !reflect.DeepEqual(f.SEOJ, NewEOJ(0x05, 0xFF, 0x01)) {
        t.Errorf("SEOJ mismatch: got %v", f.SEOJ)
    }
    if !reflect.DeepEqual(f.DEOJ, NewEOJ(0x02, 0x7D, 0x01)) {
        t.Errorf("DEOJ mismatch: got %v", f.DEOJ)
    }
    if f.ESV != ESVGet_Res || f.OPC != 1 {
        t.Errorf("ESV/OPC mismatch: ESV=%X OPC=%d", f.ESV, f.OPC)
    }
    if len(f.Properties) != 1 {
        t.Fatalf("Expected 1 property, got %d", len(f.Properties))
    }
    p := f.Properties[0]
    if p.EPC != 0xE4 || p.PDC != 1 || !reflect.DeepEqual(p.EDT, []byte{0x32}) {
        t.Errorf("Property mismatch: %+v", p)
    }
}


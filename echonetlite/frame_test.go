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

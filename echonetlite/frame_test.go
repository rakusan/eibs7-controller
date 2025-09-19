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

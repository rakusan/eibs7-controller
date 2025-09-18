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

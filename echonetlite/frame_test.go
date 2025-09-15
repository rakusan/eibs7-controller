package echonetlite

import (
    "bytes"
    "reflect"
    "testing"
)

func TestMarshalUnmarshalBinary(t *testing.T) {
    // Construct a sample frame with one property
    original := Frame{
        EHD1: EchonetLiteEHD1,
        EHD2: Format1,
        TID:  0x1234,
        SEOJ: NewEOJ(0x05, 0xFF, 0x01),
        DEOJ: NewEOJ(0x02, 0x7D, 0x01),
        ESV:  ESVGet,
        OPC:  1,
        Properties: []Property{{EPC: 0xE4, PDC: 0x00, EDT: nil}},
    }

    data, err := original.MarshalBinary()
    if err != nil {
        t.Fatalf("MarshalBinary failed: %v", err)
    }

    var decoded Frame
    if err := decoded.UnmarshalBinary(data); err != nil {
        t.Fatalf("UnmarshalBinary failed: %v", err)
    }

    // Compare fields (except OPC which may be set by caller). Ensure equality.
    if !reflect.DeepEqual(original, decoded) {
        t.Errorf("Round‑trip mismatch.\nOriginal: %+v\nDecoded : %+v", original, decoded)
    }

    // Additional sanity: ensure MarshalBinary of decoded yields same bytes
    data2, err := decoded.MarshalBinary()
    if err != nil {
        t.Fatalf("MarshalBinary on decoded failed: %v", err)
    }
    if !bytes.Equal(data, data2) {
        t.Errorf("Serialized bytes differ after round‑trip")
    }
}

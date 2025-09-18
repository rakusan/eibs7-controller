package echonetlite

import (
    "bytes"
    "reflect"
    "testing"
)

func TestMarshalUnmarshalBinary_RoundTrip(t *testing.T) {
    // Construct a sample frame similar to the example in the source file.
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
            EPC: 0xE5,
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

    // The OPC field is part of the binary format; ensure it matches.
    if decoded.OPC != original.OPC {
        t.Errorf("OPC mismatch after round‑trip: got %d, want %d", decoded.OPC, original.OPC)
    }

    // Compare all other fields (excluding slice backing array differences).
    if !reflect.DeepEqual(decoded.EHD1, original.EHD1) ||
        !reflect.DeepEqual(decoded.EHD2, original.EHD2) ||
        !reflect.DeepEqual(decoded.TID, original.TID) ||
        !reflect.DeepEqual(decoded.SEOJ, original.SEOJ) ||
        !reflect.DeepEqual(decoded.DEOJ, original.DEOJ) ||
        !reflect.DeepEqual(decoded.ESV, original.ESV) {
        t.Fatalf("Header fields differ after round‑trip")
    }

    if len(decoded.Properties) != len(original.Properties) {
        t.Fatalf("property count mismatch: got %d, want %d", len(decoded.Properties), len(original.Properties))
    }
    for i := range original.Properties {
        if !bytes.Equal(decoded.Properties[i].EDT, original.Properties[i].EDT) ||
            decoded.Properties[i].EPC != original.Properties[i].EPC ||
            decoded.Properties[i].PDC != original.Properties[i].PDC {
            t.Fatalf("property %d differs after round‑trip", i)
        }
    }
}

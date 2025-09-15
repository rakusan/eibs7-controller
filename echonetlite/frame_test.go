package echonetlite

import (
    "bytes"
    "testing"
)

func TestFrameMarshalUnmarshal(t *testing.T) {
    f := Frame{
        EHD1: EchonetLiteEHD1,
        EHD2: Format1,
        TID:  0x1234,
        SEOJ: NewEOJ(0x01, 0x02, 0x03),
        DEOJ: NewEOJ(0x04, 0x05, 0x06),
        ESV:  ESVGet,
        OPC:  2,
        Properties: []Property{{
            EPC: 0x80,
            PDC: 1,
            EDT: []byte{0x01},
        }, {
            EPC: 0x81,
            PDC: 2,
            EDT: []byte{0x02, 0x03},
        }},
    }
    data, err := f.MarshalBinary()
    if err != nil {
        t.Fatalf("MarshalBinary error: %v", err)
    }
    var f2 Frame
    if err := f2.UnmarshalBinary(data); err != nil {
        t.Fatalf("UnmarshalBinary error: %v", err)
    }
    // Compare binary representations for equality
    data2, _ := f2.MarshalBinary()
    if !bytes.Equal(data, data2) {
        t.Fatalf("Roundtrip mismatch\norig: %x\nnew : %x", data, data2)
    }
}

package packstream

import (
	"bytes"
	"math"
	"strconv"
	"testing"
	"time"
)

func TestNewEncoder(t *testing.T) {
	b := new(bytes.Buffer)
	if enc := NewEncoder(b); enc == nil {
		t.Error("returned value should not be nil.")
	} else if buf, ok := enc.wr.(*bytes.Buffer); !ok {
		t.Error("encoder buffer is not valid, expected a bytes buffer.")
	} else if buf != b {
		t.Error("encoder buffer is not valid.")
	}
}

func TestEncoder_Encode(t *testing.T) {
	var b bytes.Buffer
	enc := NewEncoder(&b)
	for _, val := range validTestValues {
		if err := enc.Encode(val.Decoded); err != nil {
			t.Errorf("error while encoding value %v: %v", val.Decoded, err)
		} else if b.Len() == 0 {
			t.Errorf("invalid encoded value for %v, got empty buffer.", val.Decoded)
		} else {
			res := make([]byte, b.Len())
			b.Read(res)
			if !bytes.Equal(res, val.Encoded) {
				t.Errorf("invalid encoded value for %v, got % #X, expected % #X", val.Decoded, res, val.Encoded)
			}
		}
	}
}

func TestMarshal(t *testing.T) {
	for _, val := range validTestValues {
		if b, err := Marshal(val.Decoded); err != nil {
			t.Errorf("error while encoding value %v: %v", val.Decoded, err)
		} else if !bytes.Equal(b, val.Encoded) {
			t.Errorf("invalid encoded value for %v, got % #X, expected % #X", val.Decoded, b, val.Encoded)
		}
	}
}

func TestMarshal_String(t *testing.T) {
	s := maxInt4
	longStr := string(make([]byte, s))
	if b, err := Marshal(longStr); err != nil {
		t.Errorf("error while encoding string of length %v: %v", s, err)
	} else if len(b) != s+2 {
		t.Errorf("error while encoding string of length %v: invalid buffer length got %v, expected %v", s, len(b), s+2)
	} else if b[0] != mStringSize8 {
		t.Errorf("error while encoding string of length %v: invalid marker, got %#X, expected %#X", s, b[0], mStringSize8)
	}

	s = math.MaxUint16
	longStr = string(make([]byte, s))
	if b, err := Marshal(longStr); err != nil {
		t.Errorf("error while encoding string of length %v: %v", s, err)
	} else if len(b) != s+3 {
		t.Errorf("error while encoding string of length %v: invalid buffer length got %v, expected %v", s, len(b), s+3)
	} else if b[0] != mStringSize16 {
		t.Errorf("error while encoding string of length %v: invalid marker, got %#X, expected %#X", s, b[0], mStringSize16)
	}

	s = math.MaxUint16 + 1
	longStr = string(make([]byte, s))
	if b, err := Marshal(longStr); err != nil {
		t.Errorf("error while encoding string of length %v: %v", s, err)
	} else if len(b) != s+5 {
		t.Errorf("error while encoding string of length %v: invalid buffer length got %v, expected %v", s, len(b), s+5)
	} else if b[0] != mStringSize32 {
		t.Errorf("error while encoding string of length %v: invalid marker, got %#X, expected %#X", s, b[0], mStringSize32)
	}
}

func TestMarshal_List(t *testing.T) {
	s := maxInt4
	longList := make([]interface{}, s)
	if b, err := Marshal(longList); err != nil {
		t.Errorf("error while encoding list of length %v: %v", s, err)
	} else if len(b) != s+2 {
		t.Errorf("error while encoding list of length %v: invalid buffer length got %v, expected %v", s, len(b), s+2)
	} else if b[0] != mListSize8 {
		t.Errorf("error while encoding list of length %v: invalid marker, got %#X, expected %#X", s, b[0], mListSize8)
	}

	s = math.MaxUint16
	longList = make([]interface{}, s)
	if b, err := Marshal(longList); err != nil {
		t.Errorf("error while encoding list of length %v: %v", s, err)
	} else if len(b) != s+3 {
		t.Errorf("error while encoding list of length %v: invalid buffer length got %v, expected %v", s, len(b), s+3)
	} else if b[0] != mListSize16 {
		t.Errorf("error while encoding list of length %v: invalid marker, got %#X, expected %#X", s, b[0], mListSize16)
	}

	s = math.MaxUint16 + 1
	longList = make([]interface{}, s)
	if b, err := Marshal(longList); err != nil {
		t.Errorf("error while encoding list of length %v: %v", s, err)
	} else if len(b) != s+5 {
		t.Errorf("error while encoding list of length %v: invalid buffer length got %v, expected %v", s, len(b), s+5)
	} else if b[0] != mListSize32 {
		t.Errorf("error while encoding list of length %v: invalid marker, got %#X, expected %#X", s, b[0], mListSize16)
	}
}

func TestMarshal_Map(t *testing.T) {
	longMap := make(map[string]interface{})
	s := maxInt4 - 1
	for i := 0; i < s; i++ {
		longMap[strconv.Itoa(i)] = i
	}
	if b, err := Marshal(longMap); err != nil {
		t.Errorf("error while encoding map of length %v: %v", s, err)
	} else if len(b) != 51 {
		t.Errorf("error while encoding map of length %v: invalid buffer length got %v, expected %v", s, len(b), 51)
	} else if b[0] != tinyMapSizes[s][0] {
		t.Errorf("error while encoding map of length %v: invalid marker, got %#X, expected %#X", s, b[0], tinyMapSizes[s][0])
	}

	s = maxInt4
	longMap[strconv.Itoa(s)] = s
	if b, err := Marshal(longMap); err != nil {
		t.Errorf("error while encoding map of length %v: %v", s, err)
	} else if len(b) != 56 {
		t.Errorf("error while encoding map of length %v: invalid buffer length got %v, expected %v", s, len(b), 56)
	} else if b[0] != mMapSize8 {
		t.Errorf("error while encoding map of length %v: invalid marker, got %#X, expected %#X", s, b[0], mMapSize8)
	}

	s = math.MaxUint16
	for i := 0; i < s; i++ {
		longMap[strconv.Itoa(i)] = i
	}
	if b, err := Marshal(longMap); err != nil {
		t.Errorf("error while encoding map of length %v: %v", s, err)
	} else if len(b) != 643986 {
		t.Errorf("error while encoding map of length %v: invalid buffer length got %v, expected %v", s, len(b), 643986)
	} else if b[0] != mMapSize16 {
		t.Errorf("error while encoding map of length %v: invalid marker, got %#X, expected %#X", s, b[0], mMapSize16)
	}

	s = math.MaxUint16 + 1
	longMap[strconv.Itoa(s)] = s
	if b, err := Marshal(longMap); err != nil {
		t.Errorf("error while encoding map of length %v: %v", s, err)
	} else if len(b) != 643999 {
		t.Errorf("error while encoding map of length %v: invalid buffer length got %v, expected %v", s, len(b), 643999)
	} else if b[0] != mMapSize32 {
		t.Errorf("error while encoding map of length %v: invalid marker, got %#X, expected %#X", s, b[0], mMapSize32)
	}
}

func TestMarshal_Structure(t *testing.T) {
	s := math.MaxUint16
	longSt := Structure{Signature: 42, Fields: make([]interface{}, s)}
	if b, err := Marshal(longSt); err != nil {
		t.Errorf("error while encoding structure of length %v: %v", s, err)
	} else if len(b) != s+4 {
		t.Errorf("error while encoding structure of length %v: invalid buffer length got %v, expected %v", s, len(b), s+4)
	} else if b[0] != mStructSize16 {
		t.Errorf("error while encoding structure of length %v: invalid marker, got %#X, expected %#X", s, b[0], mStructSize16)
	}
}

func TestMarshal_Bytes(t *testing.T) {
	s := math.MaxUint16
	longBytes := make([]byte, s)
	if b, err := Marshal(longBytes); err != nil {
		t.Errorf("error while encoding bytes of length %v: %v", s, err)
	} else if len(b) != 65538 {
		t.Errorf("error while encoding bytes of length %v: invalid buffer length got %v, expected %v", s, len(b), 65538)
	} else if b[0] != mBytesSize16 {
		t.Errorf("error while encoding bytes of length %v: invalid marker, got %#X, expected %#X", s, b[0], mBytesSize16)
	}

	longBytes = append(longBytes, 0)
	s++
	if b, err := Marshal(longBytes); err != nil {
		t.Errorf("error while encoding bytes of length %v: %v", s, err)
	} else if len(b) != 65541 {
		t.Errorf("error while encoding bytes of length %v: invalid buffer length got %v, expected %v", s, len(b), 65541)
	} else if b[0] != mBytesSize32 {
		t.Errorf("error while encoding bytes of length %v: invalid marker, got %#X, expected %#X", s, b[0], mBytesSize32)
	}
}

func TestMarshal_Float32(t *testing.T) {
	res := []byte{mFloat64, 0x3F, 0xF1, 0x99, 0x99, 0xA0, 0x00, 0x00, 0x00}
	if b, err := Marshal(float32(1.1)); err != nil {
		t.Errorf("error while encoding float32: %v", err)
	} else if !bytes.Equal(res, b) {
		t.Errorf("error while encoding float32 got %v, expected %v", b, res)
	}
}

func TestMarshal_Int64(t *testing.T) {
	// Just check that these types are supported
	res := []byte{mInt8, 0x80}
	if b, err := Marshal(int(math.MinInt8)); err != nil {
		t.Errorf("error while encoding int %v: %v", int(math.MinInt8), err)
	} else if !bytes.Equal(res, b) {
		t.Errorf("error while encoding int got %v, expected %v", b, res)
	}

	if b, err := Marshal(int8(math.MinInt8)); err != nil {
		t.Errorf("error while encoding int8 %v: %v", int8(math.MinInt8), err)
	} else if !bytes.Equal(res, b) {
		t.Errorf("error while encoding int8 got %v, expected %v", b, res)
	}

	if b, err := Marshal(int16(math.MinInt8)); err != nil {
		t.Errorf("error while encoding int16 %v: %v", int16(math.MinInt8), err)
	} else if !bytes.Equal(res, b) {
		t.Errorf("error while encoding int16 got %v, expected %v", b, res)
	}

	if b, err := Marshal(int32(math.MinInt8)); err != nil {
		t.Errorf("error while encoding int32 %v: %v", int32(math.MinInt8), err)
	} else if !bytes.Equal(res, b) {
		t.Errorf("error while encoding int32 got %v, expected %v", b, res)
	}
}

func TestMarshal_Marshaler(t *testing.T) {
	vMarshaller := marshaller(42)

	res := []byte{42}
	if b, err := Marshal(&vMarshaller); err != nil {
		t.Errorf("error while encoding marshaler %v: %v", vMarshaller, err)
	} else if !bytes.Equal(res, b) {
		t.Errorf("error while encoding marshaler got %v, expected %v", b, res)
	}
}

func TestEncoder_Encode_Time(t *testing.T) {
	var b bytes.Buffer
	enc := NewEncoder(&b)

	res := []byte{mInt64, 0x14, 0x25, 0x9C, 0x88, 0x0A, 0x59, 0xDE, 0x05}
	tm := time.Date(2016, time.January, 02, 12, 42, 43, 5, time.FixedZone("UTC", 0))
	if err := enc.Encode(&tm); err != nil {
		t.Error(err)
	} else if !bytes.Equal(res, b.Bytes()) {
		t.Errorf("error while encoding time got %v, expected %v", b.Bytes(), res)
	}
	b.Reset()

	res = []byte{0x0}
	tm = time.Time{}
	if err := enc.Encode(&tm); err != nil {
		t.Error(err)
	} else if !bytes.Equal(res, b.Bytes()) {
		t.Errorf("error while encoding time got %v, expected %v", b.Bytes(), res)
	}
	b.Reset()
}

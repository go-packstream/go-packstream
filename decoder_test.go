package packstream

import (
	"bytes"
	"math"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func TestNewDecoder(t *testing.T) {
	b := new(bytes.Buffer)
	if enc := NewDecoder(b); enc == nil {
		t.Error("returned value should not be nil.")
	} else if buf, ok := enc.stream.(*bytes.Buffer); !ok {
		t.Error("decoder buffer is not valid, expected a bytes buffer.")
	} else if buf != b {
		t.Error("decoder buffer is not valid.")
	}
}

func TestDecoder_Decode(t *testing.T) {
	var (
		b bytes.Buffer
		v interface{}
	)
	dec := NewDecoder(&b)
	for _, val := range validTestValues {
		v = nil
		b.Write(val.Encoded)
		if err := dec.Decode(&v); err != nil {
			t.Errorf("error while decoding value %# x: %v", val.Encoded, err)
		} else if !reflect.DeepEqual(v, val.Decoded) {
			t.Errorf("invalid decoded value, got %v, expected %v", v, val.Decoded)
		} else if b.Len() != 0 {
			t.Errorf("invalid decoding, remaining %v bytes in buffer", b.Len())
		}
	}
}

func TestUnmarshal(t *testing.T) {
	var v interface{}
	for _, val := range validTestValues {
		v = nil
		if err := Unmarshal(val.Encoded, &v); err != nil {
			t.Errorf("error while unmarshaling value %X: %v", val.Encoded, err)
		} else if !reflect.DeepEqual(v, val.Decoded) {
			t.Errorf("invalid decoded value, got %v, expected %v", v, val.Decoded)
		}
	}
}

func TestIndirect(t *testing.T) {
	var (
		p3 ***int
		um **marshaller
	)

	if marshaller, v := indirect(reflect.ValueOf(&p3), false); marshaller != nil {
		t.Errorf("unmarshaller should be nil, got %v.", marshaller)
	} else if _, ok := v.Interface().(int); !ok {
		t.Errorf("value should be an int, got %v.", v.Interface())
	} else {
		v.Set(reflect.ValueOf(42))
		if ***p3 != 42 {
			t.Errorf("expected 42 value, got %v.", ***p3)
		}
	}

	p3 = nil
	if marshaller, v := indirect(reflect.ValueOf(&p3), true); marshaller != nil {
		t.Errorf("unmarshaller should be nil, got %v.", marshaller)
	} else if v.Kind() != reflect.Ptr {
		t.Errorf("value should be an int pointer, got %v.", v.Interface())
	}

	if marshaller, _ := indirect(reflect.ValueOf(&um), false); marshaller == nil {
		t.Error("unmarshaller should not be nil")
	}
}

func TestUnmarshal_Nil(t *testing.T) {
	var (
		i  int64
		st *Structure
	)
	data := []byte{mNull}
	if err := Unmarshal(data, &i); err != nil {
		t.Error(err)
	} else if i != 0 {
		t.Errorf("i should equals 0, got %v.", i)
	}

	if err := Unmarshal(data, &st); err != nil {
		t.Error(err)
	} else if st != nil {
		t.Errorf("st should be nil, got %v.", st)
	}
}

func TestUnmarshal_String(t *testing.T) {
	var str string

	s := maxInt4
	longStr := []byte{mStringSize8}
	longStr = append(longStr, packedUint8Sizes[s]...)
	longStr = append(longStr, make([]byte, s)...)
	if err := Unmarshal(longStr, &str); err != nil {
		t.Errorf("error while unmarshaling string of length %v: %v", s, err)
	} else if len(str) != s {
		t.Errorf("error while unmarshaling string of length %v, got length of %v, expected %v.", s, len(str), s)
	}

	s = math.MaxUint16
	longStr = []byte{mStringSize16}
	longStr = append(longStr, packedUint16Sizes[s]...)
	longStr = append(longStr, make([]byte, s)...)
	if err := Unmarshal(longStr, &str); err != nil {
		t.Errorf("error while unmarshaling string of length %v: %v", s, err)
	} else if len(str) != s {
		t.Errorf("error while unmarshaling string of length %v, got length of %v, expected %v.", s, len(str), s)
	}

	s = math.MaxUint16 + 1
	longStr = []byte{mStringSize32}
	longStr = append(longStr, packedUint32Size(uint32(s))...)
	longStr = append(longStr, make([]byte, s)...)
	if err := Unmarshal(longStr, &str); err != nil {
		t.Errorf("error while unmarshaling string of length %v: %v", s, err)
	} else if len(str) != s {
		t.Errorf("error while unmarshaling string of length %v, got length of %v, expected %v.", s, len(str), s)
	}
}

func TestUnmarshal_List(t *testing.T) {
	var (
		l  []interface{}
		lb []byte
	)

	s := maxInt4
	longList := []byte{mListSize8}
	longList = append(longList, packedUint8Sizes[s]...)
	longList = append(longList, make([]byte, s)...)
	if err := Unmarshal(longList, &l); err != nil {
		t.Errorf("error while unmarshaling list of length %v: %v", s, err)
	} else if len(l) != s {
		t.Errorf("error while unmarshaling list of length %v, got length of %v, expected %v.", s, len(l), s)
	}

	if err := Unmarshal(longList, &lb); err != nil {
		t.Errorf("error while unmarshaling list of length into a typed list %v: %v", s, err)
	} else if len(lb) != s {
		t.Errorf("error while unmarshaling list of length into a typed list %v, got length of %v, expected %v.", s, len(l), s)
	}

	s = math.MaxUint16
	longList = []byte{mListSize16}
	longList = append(longList, packedUint16Sizes[s]...)
	longList = append(longList, make([]byte, s)...)
	if err := Unmarshal(longList, &l); err != nil {
		t.Errorf("error while unmarshaling list of length %v: %v", s, err)
	} else if len(l) != s {
		t.Errorf("error while unmarshaling list of length %v, got length of %v, expected %v.", s, len(l), s)
	}

	s = math.MaxUint16 + 1
	longList = []byte{mListSize32}
	longList = append(longList, packedUint32Size(uint32(s))...)
	longList = append(longList, make([]byte, s)...)
	if err := Unmarshal(longList, &l); err != nil {
		t.Errorf("error while unmarshaling list of length %v: %v", s, err)
	} else if len(l) != s {
		t.Errorf("error while unmarshaling list of length %v, got length of %v, expected %v.", s, len(l), s)
	}

	// Stream
	s = 42
	longList = []byte{mListSizeStream}
	longList = append(longList, make([]byte, s)...)
	longList = append(longList, mEndOfStream)
	if err := Unmarshal(longList, &l); err != nil {
		t.Errorf("error while unmarshaling list of length %v: %v", s, err)
	} else if len(l) != s {
		t.Errorf("error while unmarshaling list of length %v, got length of %v, expected %v.", s, len(l), s)
	}

}

func getEncodedMap(t *testing.T, s int) []byte {
	var (
		b   []byte
		err error
	)
	m := make(map[string]interface{})
	for i := 0; i < s; i++ {
		m[strconv.Itoa(i)] = i
	}
	if b, err = Marshal(m); err != nil {
		t.Fatalf("Cannot encode map of size %v.", s)
		return nil
	}
	return b
}

func TestUnmarshal_Map(t *testing.T) {
	var m map[string]interface{}

	s := maxInt4 - 1
	if err := Unmarshal(getEncodedMap(t, s), &m); err != nil {
		t.Errorf("error while unmarshaling map of length %v: %v", s, err)
	} else if len(m) != s {
		t.Errorf("error while unmarshaling map of length %v, got length of %v, expected %v.", s, len(m), s)
	}

	s = maxInt4
	if err := Unmarshal(getEncodedMap(t, s), &m); err != nil {
		t.Errorf("error while unmarshaling map of length %v: %v", s, err)
	} else if len(m) != s {
		t.Errorf("error while unmarshaling map of length %v, got length of %v, expected %v.", s, len(m), s)
	}

	s = math.MaxUint16
	if err := Unmarshal(getEncodedMap(t, s), &m); err != nil {
		t.Errorf("error while unmarshaling map of length %v: %v", s, err)
	} else if len(m) != s {
		t.Errorf("error while unmarshaling map of length %v, got length of %v, expected %v.", s, len(m), s)
	}

	s = math.MaxUint16 + 1
	if err := Unmarshal(getEncodedMap(t, s), &m); err != nil {
		t.Errorf("error while unmarshaling map of length %v: %v", s, err)
	} else if len(m) != s {
		t.Errorf("error while unmarshaling map of length %v, got length of %v, expected %v.", s, len(m), s)
	}

	// Stream
	m = nil
	s = 1
	if err := Unmarshal([]byte{mMapSizeStream, 0x82, 0x34, 0x32, 0x2A, mEndOfStream}, &m); err != nil {
		t.Errorf("error while unmarshaling map of length %v: %v", s, err)
	} else if len(m) != s {
		t.Errorf("error while unmarshaling map of length %v, got length of %v, expected %v.", s, len(m), s)
	}

	// Check that it reuse an existing map
	m = make(map[string]interface{})
	m["test"] = 42
	if err := Unmarshal([]byte{0xA1, 0x82, 0x34, 0x32, 0x2A}, &m); err != nil {
		t.Errorf("error while unmarshaling into an existing existing map: %v", err)
	} else if len(m) != 2 {
		t.Errorf("error while unmarshaling into an existing map, got length of %v, expected %v.", len(m), 2)
	}
}

func TestUnmarshal_Structure(t *testing.T) {
	var st Structure

	s := math.MaxUint16
	b := []byte{mStructSize16}
	b = append(b, packedUint16Sizes[s]...)
	b = append(b, 42) // Signature
	for i := 0; i < math.MaxUint16; i++ {
		b = append(b, mNull)
	}
	if err := Unmarshal(b, &st); err != nil {
		t.Errorf("error while unmarshaling structure of size %v: %v", s, err)
	} else if len(st.Fields) != s {
		t.Errorf("error while unmarshaling structure of size %v, got size of %v, expected %v.", s, len(st.Fields), s)
	}
}

func TestUnmarshal_Bytes(t *testing.T) {
	var bRes []byte

	s := math.MaxUint16
	b := []byte{mBytesSize16}
	b = append(b, packedUint16Sizes[s]...)
	for i := 0; i < math.MaxUint16; i++ {
		b = append(b, 42)
	}
	if err := Unmarshal(b, &bRes); err != nil {
		t.Errorf("error while unmarshaling bytes of size %v: %v", s, err)
	} else if len(bRes) != s {
		t.Errorf("error while unmarshaling bytes of size %v, got size of %v, expected %v.", s, len(bRes), s)
	}

	s = math.MaxUint16 + 1
	b = []byte{mBytesSize32}
	b = append(b, packedUint32Size(uint32(s))...)
	for i := 0; i < s; i++ {
		b = append(b, 42)
	}
	if err := Unmarshal(b, &bRes); err != nil {
		t.Errorf("error while unmarshaling bytes of size %v: %v", s, err)
	} else if len(bRes) != s {
		t.Errorf("error while unmarshaling bytes of size %v, got size of %v, expected %v.", s, len(bRes), s)
	}
}

func TestUnmarshal_Unmarshaller(t *testing.T) {
	var vMarshaller marshaller
	if err := Unmarshal([]byte{0}, &vMarshaller); err != nil {
		t.Errorf("error while unmarshaling unmarshaler: %v", err)
	} else if vMarshaller != 42 {
		t.Errorf("error while unmarshaling unmarshaler, got %v, expected %v.", vMarshaller, 42)
	}
}

func TestUnmarshal_Int(t *testing.T) {
	var (
		i8   int8
		i16  int16
		i32  int32
		ui8  uint8
		ui16 uint16
		ui32 uint32
		ui64 uint64
	)

	// Just check that theses types are supported
	if err := Unmarshal([]byte{mInt8, 0x80}, &i8); err != nil {
		t.Error(err)
	} else if i8 != math.MinInt8 {
		t.Errorf("error while unmarshaling int8, got %v, expected %v.", i8, math.MinInt8)
	}

	if err := Unmarshal([]byte{mInt16, 0X80, 0X00}, &i16); err != nil {
		t.Error(err)
	} else if i16 != math.MinInt16 {
		t.Errorf("error while unmarshaling int16, got %v, expected %v.", i16, math.MinInt16)
	}

	if err := Unmarshal([]byte{mInt32, 0X80, 0X00, 0X00, 0X00}, &i32); err != nil {
		t.Error(err)
	} else if i32 != math.MinInt32 {
		t.Errorf("error while unmarshaling int32, got %v, expected %v.", i32, math.MinInt16)
	}

	if err := Unmarshal([]byte{0x7F}, &ui8); err != nil {
		t.Error(err)
	} else if ui8 != math.MaxInt8 {
		t.Errorf("error while unmarshaling int32, got %v, expected %v.", ui8, math.MaxInt8)
	}

	if err := Unmarshal([]byte{mInt16, 0X7F, 0XFF}, &ui16); err != nil {
		t.Error(err)
	} else if ui16 != math.MaxInt16 {
		t.Errorf("error while unmarshaling int32, got %v, expected %v.", ui16, math.MaxInt16)
	}

	if err := Unmarshal([]byte{mInt32, 0X7F, 0XFF, 0XFF, 0XFF}, &ui32); err != nil {
		t.Error(err)
	} else if ui32 != math.MaxInt32 {
		t.Errorf("error while unmarshaling int32, got %v, expected %v.", ui32, math.MaxInt32)
	}

	if err := Unmarshal([]byte{mInt64, 0X7F, 0XFF, 0XFF, 0XFF, 0XFF, 0XFF, 0XFF, 0XFF}, &ui64); err != nil {
		t.Error(err)
	} else if ui64 != math.MaxInt64 {
		t.Errorf("error while unmarshaling int64, got %v, expected %v.", ui64, math.MaxInt64)
	}

	// Overflows
	if err := Unmarshal([]byte{mInt64, 0X7F, 0XFF, 0XFF, 0XFF, 0XFF, 0XFF, 0XFF, 0XFF}, &ui8); err == nil {
		t.Error("error should not be nil when integer is too big to be stored.")
	}
}

func TestUnmarshal_Float(t *testing.T) {
	var f32 float32

	if err := Unmarshal([]byte{mFloat64, 0xBF, 0xF1, 0x99, 0x99, 0x99, 0x99, 0x99, 0x9A}, &f32); err != nil {
		t.Error(err)
	} else if f32 != -1.1 {
		t.Errorf("error while unmarshaling int64, got %v, expected %v.", f32, -1.1)
	}

}

func TestUnmarshal_Time(t *testing.T) {
	var (
		tm time.Time
	)
	decoded := time.Date(2016, time.January, 02, 12, 42, 43, 5, time.FixedZone("America/New_York", 0))
	if encoded, err := Marshal(decoded.UnixNano()); err != nil {
		t.Error(err)
	} else if err := Unmarshal(encoded, &tm); err != nil {
		t.Error(err)
	} else if !decoded.Equal(tm) {
		t.Errorf("unexpected time value, expected %v got %v.", decoded, tm)
	}

	if encoded, err := Marshal(0); err != nil {
		t.Error(err)
	} else if err := Unmarshal(encoded, &tm); err != nil {
		t.Error(err)
	} else if !tm.IsZero() {
		t.Errorf("time should be a zero value, got %v.", tm)
	}
}

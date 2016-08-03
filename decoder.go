package packstream

import (
	"bytes"
	"encoding/binary"
	"io"
	"math"
	"reflect"
	"runtime"
	"time"
)

// Decoder can read and decodes packstream data from an input stream.
type Decoder struct {
	stream io.Reader
}

// NewDecoder returns a new decoder that reads from rd.
func NewDecoder(rd io.Reader) *Decoder {
	return &Decoder{stream: rd}
}

type decodeState struct {
	stream io.Reader
	bytes  []byte
	cursor uint64
	marker byte
	eos    bool
}

// readBytes reads s bytes from the input, and returns, and move d.cursor.
// If there is not enough bytes to read, readBytes returns io.EOF error.
func (d *decodeState) readBytes(s uint64) ([]byte, error) {
	var (
		p1 [1]byte
		p2 [2]byte
		p4 [4]byte
		p8 [8]byte
		p  []byte
	)

	if d.stream != nil {
		switch s {
		default:
			p = make([]byte, s)
		case 1:
			p = p1[:]
		case 2:
			p = p2[:]
		case 4:
			p = p4[:]
		case 8:
			p = p8[:]
		}

		if _, err := io.ReadFull(d.stream, p); err != nil {
			if err == io.ErrUnexpectedEOF {
				return nil, io.EOF
			}
			return nil, err
		}
		return p, nil
	}
	if uint64(len(d.bytes))-d.cursor < s {
		d.cursor = uint64(len(d.bytes))
		return d.bytes[d.cursor:], io.EOF
	}

	i := d.cursor
	d.cursor += s
	return d.bytes[i : i+s], nil
}

// readMarker reads one byte and set d.marker
func (d *decodeState) readMarker() error {
	var (
		p   []byte
		err error
	)
	if p, err = d.readBytes(1); err != nil {
		return err
	}
	d.marker = p[0]
	return nil
}

// readSize reads 1, 2 or 4 bytes and interpret them as an uint64.
func (d *decodeState) readSize(s uint64) (ui uint64, err error) {
	var p []byte
	if p, err = d.readBytes(s); err != nil {
		return
	}
	switch s {
	case 1:
		ui = uint64(p[0])
	case 2:
		ui |= uint64(p[0]) << 8
		ui |= uint64(p[1])
	case 4:
		ui |= uint64(p[0]) << 24
		ui |= uint64(p[1]) << 16
		ui |= uint64(p[2]) << 8
		ui |= uint64(p[3])
	}
	return
}

// Decode reads the next packstream encoded value from its input and stores it in the value pointed to by v.
// See the documentation for Unmarshal for details about the conversion of packstream into a Go value.
func (d *Decoder) Decode(v interface{}) error {
	dec := &decodeState{stream: d.stream}
	return dec.unmarshal(v)
}

/*
Unmarshal parses the the packstream encoded data and store the result in the value pointed by v.

Only pointers must be passed as v. If v is a nil pointer, then unmarshal allocates a new value for it to point to.
Unmarshal first handles the nil case, and set the value to its zero value.

Generic decoding can be done by passing a pointer to an empty interface.

Unmarshal can decode the following go values:
	nil
	bool
	float32, float64
	int, int8, int16, int32, int64
	uint, uint8, uint16, uint32, uint64
	string
	[]byte
	[]interface{}
	map[string]interface{}
	Structure
	time.Time

To unmarshal a list into a Go array, Unmarshal decodes packstream list elements into corresponding Go array elements.
If the Go array is smaller than the JSON array, the additional JSON array elements are discarded.
If the JSON array is smaller than the Go array, the additional Go array elements are set to zero values.

To unmarshal a packstream map into a string-keyed map, Unmarshal first establishes a map to use.
If the map is nil, Unmarshal allocates a new map.
Otherwise Unmarshal reuses the existing map, keeping existing entries.
Unmarshal then stores key-value pairs from the packstream map into the map.

To unmarshal a time.Time, the packstream value must be an integer, which represents the number of nanoseconds elapsed
since January 1, 1970 UTC. Then, the time structure is filled using time.Unix(). If the integer is zero, it unmarshals
a zero value time.Time.

If a packstream value is not appropriate for a given target type, or if a number overflows the target type,
Unmarshal returns an error.
*/
func Unmarshal(data []byte, v interface{}) error {
	dec := decodeState{bytes: data}
	return dec.unmarshal(v)
}

func (d *decodeState) unmarshal(v interface{}) (err error) {
	defer func() {
		if r := recover(); r != nil {
			if _, ok := r.(runtime.Error); ok {
				panic(r)
			}
			err = r.(error)
		}
	}()

	rv := reflect.ValueOf(v)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return ErrUnMarshalTypeError
	}
	return d.value(rv)
}

// indirect walks down v allocating pointers as needed,
// until it gets to a non-pointer.
// if it encounters an Unmarshaler, indirect stops and returns that.
// if decodingNull is true, indirect stops at the last pointer so it can be set to nil.
func indirect(v reflect.Value, decodingNull bool) (Unmarshaler, reflect.Value) {
	// If v is a named type and is addressable,
	// start with its address, so that if the type has pointer methods,
	// we find them.
	if v.Kind() != reflect.Ptr && v.Type().Name() != "" && v.CanAddr() {
		v = v.Addr()
	}
	for {
		// Load value from interface, but only if the result will be
		// usefully addressable.
		if v.Kind() == reflect.Interface && !v.IsNil() {
			e := v.Elem()
			if e.Kind() == reflect.Ptr && !e.IsNil() && (!decodingNull || e.Elem().Kind() == reflect.Ptr) {
				v = e
				continue
			}
		}

		if v.Kind() != reflect.Ptr {
			break
		}

		if v.Elem().Kind() != reflect.Ptr && decodingNull && v.CanSet() {
			break
		}
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		if v.Type().NumMethod() > 0 {
			if u, ok := v.Interface().(Unmarshaler); ok {
				return u, reflect.Value{}
			}
		}
		v = v.Elem()
	}
	return nil, v
}

func (d *decodeState) value(rv reflect.Value) (err error) {
	if err = d.readMarker(); err != nil {
		return
	}

	if d.marker == mNull {
		return d.unmarshalNull(rv)
	}

	unmarshaler, rev := indirect(rv, false)
	if unmarshaler != nil {
		return d.unmarshalUnmarshaler(unmarshaler)
	}

	if d.marker >= mTinyStringStart && d.marker <= mTinyStructEnd {
		return d.unmarshalTiny(rev)
	}
	if minTinyInt <= int8(d.marker) {
		return d.unmarshalInt(rev)
	}
	switch d.marker {
	case mStringSize8, mStringSize16, mStringSize32:
		err = d.unmarshalString(rev)
	case mListSize8, mListSize16, mListSize32, mListSizeStream:
		err = d.unmarshalList(rev)
	case mMapSize8, mMapSize16, mMapSize32, mMapSizeStream:
		err = d.unmarshalMap(rev)
	case mStructSize8, mStructSize16:
		err = d.unmarshalStruct(rev)
	case mBytesSize8, mBytesSize16, mBytesSize32:
		err = d.unmarshalBytes(rev)
	case mInt8, mInt16, mInt32, mInt64:
		err = d.unmarshalInt(rev)
	case mFloat64:
		err = d.unmarshalFloat(rev)
	case mFalse, mTrue:
		err = d.unmarshalBool(rev)
	case mEndOfStream:
		d.eos = true
	}
	return
}

func (d *decodeState) unmarshalTiny(rv reflect.Value) (err error) {
	markerHigh := d.marker & 0xF0
	switch markerHigh {
	case mTinyStringStart:
		err = d.unmarshalString(rv)
	case mTinyListStart:
		err = d.unmarshalList(rv)
	case mTinyMapStart:
		err = d.unmarshalMap(rv)
	case mTinyStructStart:
		err = d.unmarshalStruct(rv)
	}
	return
}

func (d *decodeState) unmarshalNull(rv reflect.Value) error {
	unmarshaler, rev := indirect(rv, true)
	if unmarshaler != nil {
		return d.unmarshalUnmarshaler(unmarshaler)
	}
	rev.Set(reflect.Zero(rev.Type()))
	return nil
}

func (d *decodeState) unmarshalUnmarshaler(um Unmarshaler) error {
	var rd *bytes.Reader

	if d.stream != nil {
		return um.UnmarshalPS(d.marker, d.stream)
	}

	rd = bytes.NewReader(d.bytes[d.cursor:])
	i := rd.Len()
	err := um.UnmarshalPS(d.marker, rd)
	d.cursor += uint64(i - rd.Len())
	return err
}

func (d *decodeState) unmarshalInt(rv reflect.Value) (err error) {
	var (
		p []byte
		v int64
		u uint64
	)

	switch {
	case minTinyInt <= int8(d.marker):
		v = int64(int8(d.marker))
	case d.marker == mInt8:
		if p, err = d.readBytes(1); err != nil {
			return err
		}
		v = int64(int8(p[0]))
	case d.marker == mInt16:
		if p, err = d.readBytes(2); err != nil {
			return err
		}
		i := int16(0)
		i |= int16(p[0]) << 8
		i |= int16(p[1])
		v = int64(i)
	case d.marker == mInt32:
		if p, err = d.readBytes(4); err != nil {
			return err
		}
		i := int32(0)
		i |= int32(p[0]) << 24
		i |= int32(p[1]) << 16
		i |= int32(p[2]) << 8
		i |= int32(p[3])
		v = int64(i)
	case d.marker == mInt64:
		if p, err = d.readBytes(8); err != nil {
			return err
		}
		v |= int64(p[0]) << 56
		v |= int64(p[1]) << 48
		v |= int64(p[2]) << 40
		v |= int64(p[3]) << 32
		v |= int64(p[4]) << 24
		v |= int64(p[5]) << 16
		v |= int64(p[6]) << 8
		v |= int64(p[7])
	}
	switch rv.Kind() {
	default:
		if rv.Type().PkgPath() == "time" && rv.Type().Name() == "Time" {
			if v != 0 {
				rv.Set(reflect.ValueOf(time.Unix(0, v).UTC()))
			} else {
				rv.Set(reflect.ValueOf(time.Time{}))
			}
		} else {
			err = ErrUnMarshalTypeError
		}
	case reflect.Interface:
		if rv.NumMethod() != 0 {
			err = ErrUnMarshalTypeError
		} else {
			rv.Set(reflect.ValueOf(v))
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v < 0 {
			err = ErrUnMarshalTypeError
		} else {
			u = uint64(v)
			if rv.OverflowUint(u) {
				err = ErrUnMarshalTypeError
			} else {
				rv.SetUint(u)
			}
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if err != nil || rv.OverflowInt(v) {
			err = ErrUnMarshalTypeError
		} else {
			rv.SetInt(v)
		}
	}
	return
}

func (d *decodeState) unmarshalString(rv reflect.Value) (err error) {
	var (
		p []byte
		s uint64
	)
	if (d.marker & 0xF0) == mTinyStringStart {
		s = uint64(d.marker & 0x0F)
	} else {
		switch {
		case d.marker == mStringSize8:
			s, err = d.readSize(1)
		case d.marker == mStringSize16:
			s, err = d.readSize(2)
		case d.marker == mStringSize32:
			s, err = d.readSize(4)
		}
		if err != nil {
			return
		}
	}

	if p, err = d.readBytes(s); err != nil {
		return
	}
	switch rv.Kind() {
	default:
		err = ErrUnMarshalTypeError
	case reflect.Interface:
		if rv.NumMethod() != 0 {
			err = ErrUnMarshalTypeError
		} else {
			rv.Set(reflect.ValueOf(string(p)))
		}
	case reflect.String:
		rv.Set(reflect.ValueOf(string(p)))

	}
	return
}

func (d *decodeState) unmarshalList(rv reflect.Value) (err error) {
	var (
		s        uint64
		isStream bool
	)

	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array && rv.Kind() != reflect.Interface {
		return ErrUnMarshalTypeError
	}

	if (d.marker & 0xF0) == mTinyListStart {
		s = uint64(d.marker & 0x0F)
	} else {
		switch {
		case d.marker == mListSize8:
			s, err = d.readSize(1)
		case d.marker == mListSize16:
			s, err = d.readSize(2)
		case d.marker == mListSize32:
			s, err = d.readSize(4)
		case d.marker == mListSizeStream:
			isStream = true
		}
		if err != nil {
			return
		}
	}

	if rv.Kind() == reflect.Interface {
		if rv.NumMethod() != 0 {
			return ErrUnMarshalTypeError
		}
		l := make([]interface{}, s)
		rv.Set(reflect.ValueOf(l))
		rv = rv.Elem()
	}

	if !isStream {
		err = d.unmarshalSizedList(rv, int(s))
	} else {
		err = d.unmarshalStreamedList(rv)
	}

	return
}

func (d *decodeState) unmarshalStreamedList(rv reflect.Value) (err error) {
	var skipper interface{}
	i := 0
	for {
		if i >= rv.Cap() {
			newcap := rv.Cap() + rv.Cap()/2
			if newcap < 4 {
				newcap = 4
			}
			newv := reflect.MakeSlice(rv.Type(), rv.Len(), newcap)
			reflect.Copy(newv, rv)
			rv.Set(newv)
		}
		if i >= rv.Len() {
			rv.SetLen(i + 1)
		}
		if i < rv.Len() {
			// Decode into element.
			if err = d.value(rv.Index(i)); err != nil {
				return
			}
		} else {
			// Ran out of fixed array: skip.
			if err = d.unmarshal(&skipper); err != nil {
				return
			}
			skipper = nil
		}
		if d.eos {
			d.eos = false
			break
		}
		i++
	}
	d.adjustSliceLen(rv, i)
	return
}

func (d *decodeState) unmarshalSizedList(rv reflect.Value, s int) (err error) {
	var skipper interface{}
	if rv.Kind() == reflect.Slice {
		// Grow slice if necessary
		if s > rv.Cap() {
			rv.Set(reflect.MakeSlice(rv.Type(), s, s))
		}
		if s > rv.Len() {
			rv.SetLen(s)
		}
	}
	for i := 0; i < s; i++ {
		if i < rv.Len() {
			// Decode into element.
			if err = d.value(rv.Index(i)); err != nil {
				return
			}
		} else {
			// Ran out of fixed array: skip.
			if err = d.unmarshal(&skipper); err != nil {
				return
			}
			skipper = nil
		}
	}
	d.adjustSliceLen(rv, s)
	return
}

func (d *decodeState) adjustSliceLen(rv reflect.Value, s int) {
	if s < rv.Len() {
		if rv.Kind() == reflect.Array {
			// Array.  Zero the rest.
			z := reflect.Zero(rv.Type().Elem())
			for ; s < rv.Len(); s++ {
				rv.Index(s).Set(z)
			}
		} else {
			rv.SetLen(s)
		}
	}
}

func (d *decodeState) unmarshalMap(rv reflect.Value) (err error) {
	var (
		key      string
		value    interface{}
		s        uint64
		m        map[string]interface{}
		isStream bool
	)

	if rv.Kind() != reflect.Map && rv.Kind() != reflect.Interface {
		return ErrUnMarshalTypeError
	} else if rv.Kind() == reflect.Map && rv.Type().Key().Kind() != reflect.String {
		return ErrUnMarshalTypeError
	}

	if rv.Kind() == reflect.Interface {
		if rv.NumMethod() != 0 {
			return ErrUnMarshalTypeError
		}
		m = make(map[string]interface{})
		rv.Set(reflect.ValueOf(m))
		rv = rv.Elem()
	} else if rv.IsNil() {
		rv.Set(reflect.MakeMap(reflect.TypeOf(m)))
	}

	if (d.marker & 0xF0) == mTinyMapStart {
		s = uint64(d.marker & 0x0F)
	} else {
		switch {
		case d.marker == mMapSize8:
			s, err = d.readSize(1)
		case d.marker == mMapSize16:
			s, err = d.readSize(2)
		case d.marker == mMapSize32:
			s, err = d.readSize(4)
		case d.marker == mMapSizeStream:
			isStream = true
		}
		if err != nil {
			return
		}
	}

	iS := int(s)
	i := 0
	for {
		if !isStream && i >= iS {
			break
		}
		if err = d.unmarshal(&key); err != nil {
			break
		}
		if d.eos {
			break
		}
		if err = d.unmarshal(&value); err != nil {
			break
		}
		rv.SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value))

		i++
	}
	return
}

func (d *decodeState) unmarshalStruct(rv reflect.Value) (err error) {
	var (
		p      []byte
		st     Structure
		s      uint64
		fields []interface{}
	)

	if rv.Kind() != reflect.Struct && rv.Kind() != reflect.Interface {
		return ErrUnMarshalTypeError
	} else if rv.Kind() == reflect.Struct {
		if _, ok := rv.Interface().(Structure); !ok {
			return ErrUnMarshalTypeError
		}
	}

	if (d.marker & 0xF0) == mTinyStructStart {
		s = uint64(d.marker & 0x0F)
	} else {
		switch {
		case d.marker == mStructSize8:
			s, err = d.readSize(1)
		case d.marker == mStructSize16:
			s, err = d.readSize(2)
		}
		if err != nil {
			return
		}
	}

	if p, err = d.readBytes(1); err != nil {
		return
	}

	fields = make([]interface{}, s)
	if rv.Kind() == reflect.Interface {
		if rv.NumMethod() != 0 {
			return ErrUnMarshalTypeError
		}
		st.Signature = p[0]
		st.Fields = fields
		rv.Set(reflect.ValueOf(st))
		rv = rv.Elem()
	} else {
		rv.FieldByName("Signature").Set(reflect.ValueOf(p[0]))
		rv.FieldByName("Fields").Set(reflect.ValueOf(fields))
	}
	iS := int(s)
	for i := 0; i < iS; i++ {
		if err = d.unmarshal(&fields[i]); err != nil {
			return
		}
	}
	return
}

func (d *decodeState) unmarshalBytes(rv reflect.Value) (err error) {
	var (
		p []byte
		s uint64
	)
	if rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array && rv.Kind() != reflect.Interface {
		return ErrUnMarshalTypeError
	} else if rv.Kind() != reflect.Interface && rv.Type().Elem().Kind() != reflect.Uint8 {
		return ErrUnMarshalTypeError
	}

	switch {
	case d.marker == mBytesSize8:
		s, err = d.readSize(1)
	case d.marker == mBytesSize16:
		s, err = d.readSize(2)
	case d.marker == mBytesSize32:
		s, err = d.readSize(4)
	}
	if err != nil {
		return
	}
	if p, err = d.readBytes(s); err != nil {
		return
	}

	pV := reflect.ValueOf(p)
	if rv.Kind() == reflect.Interface {
		if rv.NumMethod() != 0 {
			return ErrUnMarshalTypeError
		}
		rv.Set(pV)
		return
	}

	iS := int(s)
	if rv.Kind() == reflect.Slice {
		// Grow slice if necessary
		if iS > rv.Cap() {
			rv.Set(pV)
		}
		if iS != rv.Len() {
			rv.SetLen(iS)
		}
	}
	reflect.Copy(rv, pV)
	return
}

func (d *decodeState) unmarshalBool(rv reflect.Value) error {
	var b bool
	if rv.Kind() != reflect.Bool && rv.Kind() != reflect.Interface {
		return ErrUnMarshalTypeError
	}
	switch d.marker {
	case mTrue:
		b = true
	case mFalse:
		b = false
	}

	if rv.Kind() == reflect.Bool {
		rv.SetBool(b)
	} else if rv.Kind() == reflect.Interface {
		rv.Set(reflect.ValueOf(b))
	}
	return nil
}

func (d *decodeState) unmarshalFloat(rv reflect.Value) (err error) {
	var (
		f float64
		p []byte
	)

	if p, err = d.readBytes(8); err != nil {
		return
	}

	f = math.Float64frombits(binary.BigEndian.Uint64(p))
	switch rv.Kind() {
	default:
		return ErrUnMarshalTypeError
	case reflect.Interface:
		rv.Set(reflect.ValueOf(f))
		return nil
	case reflect.Float32:
		if rv.OverflowFloat(f) {
			return ErrUnMarshalTypeError
		}
	}
	rv.SetFloat(f)
	return
}

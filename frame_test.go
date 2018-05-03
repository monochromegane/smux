package smux

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestPacking(t *testing.T) {
	streamId := uint32(1)
	data := bytes.Repeat([]byte{1, 2, 3, 4, 5, 6, 7, 8}, 2048+1)

	frames := packing(streamId, data, false) // no sealing
	if len(frames) != 2 {
		t.Errorf("packing should return 2 frames, but %d", len(frames))
	}
	for i := 0; i < len(frames); i++ {
		l, t_, f, s := parseHeader(frames[i][0:NUM_BYTES_HEADER])
		length := NUM_BYTES_MAX_PAYLOAD
		if i == len(frames)-1 {
			length = len(data) - (i * NUM_BYTES_MAX_PAYLOAD)
		}
		if l != uint16(length) {
			t.Errorf("packing should return length of payload %d, but %d.", uint16(length), l)
		}
		if t_ != uint8(TYPE_DATA) {
			t.Errorf("packing should return DATA type, but %d", t_)
		}
		if f != uint8(FLAG_DATA_NONE) {
			t.Errorf("packing should return NONE flag, but %d", f)
		}
		if s != streamId {
			t.Errorf("packing should return stream id %d, but %d", streamId, s)
		}
		payload := len(frames[i]) - NUM_BYTES_HEADER
		if payload != length {
			t.Errorf("packing should return payload %d, but %d.", length, payload)
		}
	}
}

func TestPackingAndSeal(t *testing.T) {
	streamId := uint32(1)
	data := bytes.Repeat([]byte{1, 2, 3, 4, 5, 6, 7, 8}, 2048+1)

	frames := packing(streamId, data, true) // sealing
	if len(frames) != 2 {
		t.Errorf("packing should return 2 frames, but %d", len(frames))
	}
	for i := 0; i < len(frames); i++ {
		l, t_, f, s := parseHeader(frames[i][0:NUM_BYTES_HEADER])
		length := NUM_BYTES_MAX_PAYLOAD
		if i == len(frames)-1 {
			length = len(data) - (i * NUM_BYTES_MAX_PAYLOAD)
		}
		if l != uint16(length) {
			t.Errorf("packing should return length of payload %d, but %d.", uint16(length), l)
		}
		if t_ != uint8(TYPE_DATA) {
			t.Errorf("packing should return DATA type, but %d", t_)
		}
		if i == len(frames)-1 {
			// end
			if f != uint8(FLAG_DATA_END_STREAM) {
				t.Errorf("packing should return END_STREAM flag, but %d", f)
			}
		} else {
			// continuous
			if f != uint8(FLAG_DATA_NONE) {
				t.Errorf("packing should return NONE flag, but %d", f)
			}
		}
		if s != streamId {
			t.Errorf("packing should return stream id %d, but %d", streamId, s)
		}
		payload := len(frames[i]) - NUM_BYTES_HEADER
		if payload != length {
			t.Errorf("packing should return payload %d, but %d.", length, payload)
		}
	}
}

func TestSealing(t *testing.T) {
	streamId := uint32(1)
	endFrame := sealing(streamId)

	if len(endFrame) != NUM_BYTES_HEADER {
		t.Errorf("sealing should return only header frame.")
	}
	l, t_, f, s := parseHeader(endFrame)
	if l != 0 {
		t.Errorf("sealing should not return payload.")
	}
	if t_ != uint8(TYPE_DATA) {
		t.Errorf("sealing should return DATA type, but %d", t_)
	}
	if f != uint8(FLAG_DATA_END_STREAM) {
		t.Errorf("sealing should return END_STREAM flag, but %d", f)
	}
	if s != streamId {
		t.Errorf("sealing should return stream id %d, but %d", streamId, s)
	}
}

func TestParseHeader(t *testing.T) {
	header := make([]byte, NUM_BYTES_HEADER)
	length := uint16(NUM_BYTES_MAX_PAYLOAD)
	typ := uint8(TYPE_DATA)
	flag := uint8(FLAG_DATA_END_STREAM)
	streamId := uint32(MAX_STREAM_ID)

	binary.BigEndian.PutUint16(header[0:2], length)
	header[2] = typ
	header[3] = flag
	binary.BigEndian.PutUint32(header[4:], streamId)

	l, t_, f, s := parseHeader(header)
	if l != length {
		t.Errorf("parseHeader should return length %d, but %d", length, l)
	}
	if t_ != typ {
		t.Errorf("parseHeader should return type %d, but %d", typ, t_)
	}
	if f != flag {
		t.Errorf("parseHeader should return flag %d, but %d", flag, f)
	}
	if s != streamId {
		t.Errorf("parseHeader should return streamId %d, but %d", streamId, s)
	}
}

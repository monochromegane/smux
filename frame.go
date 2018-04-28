package smux

import (
	"encoding/binary"
	"math"
)

func NewEndStreamFrame(id uint32) []byte {
	frame := make([]byte, NUM_BYTES_HEADER)

	binary.BigEndian.PutUint16(frame[0:2], uint16(0))
	frame[2] = uint8(TYPE_DATA)
	frame[3] = uint8(FLAG_DATA_END_STREAM)
	binary.BigEndian.PutUint32(frame[4:8], id)
	return frame
}

func NewFrame(id uint32, b []byte, once bool) [][]byte {
	numFrames := int(math.Ceil(float64(len(b)) / float64(NUM_BYTES_MAX_PAYLOAD)))
	frames := make([][]byte, numFrames)
	for i := 0; i < numFrames; i++ {
		length := NUM_BYTES_MAX_PAYLOAD
		typ := TYPE_DATA
		flag := FLAG_DATA_NONE
		if i == numFrames-1 {
			length = len(b) - (i * NUM_BYTES_MAX_PAYLOAD)
			if once {
				flag = FLAG_DATA_END_STREAM
			}
		}
		frame := make([]byte, NUM_BYTES_HEADER+length)

		binary.BigEndian.PutUint16(frame[0:2], uint16(length))
		frame[2] = uint8(typ)
		frame[3] = uint8(flag)
		binary.BigEndian.PutUint32(frame[4:8], id)
		payload := b[i*NUM_BYTES_MAX_PAYLOAD : i*NUM_BYTES_MAX_PAYLOAD+length]
		for j, _ := range payload {
			frame[NUM_BYTES_HEADER+j] = payload[j]
		}
		frames[i] = frame
	}
	return frames
}

func headerFromBytes(b []byte) (uint16, uint8, uint8, uint32) {
	return binary.BigEndian.Uint16(b[0:2]), // length
		uint8(b[2]), // type
		uint8(b[3]), // flag
		binary.BigEndian.Uint32(b[4:]) // stream id
}

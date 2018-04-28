package smux

import (
	"encoding/binary"
	"math"
)

type FrameHeader struct {
	length   uint16
	typ      uint8
	flag     uint8
	streamId uint32
}

const (
	// MAX_PAYLOAD = 2048
	MAX_PAYLOAD = 5
)

func NewEndStreamFrame(id uint32) []byte {
	frame := make([]byte, 8)

	binary.BigEndian.PutUint16(frame[0:2], uint16(0))
	frame[2] = uint8(0)
	frame[3] = uint8(1)
	binary.BigEndian.PutUint32(frame[4:8], id)
	return frame
}

func NewFrame(id uint32, b []byte, once bool) [][]byte {
	numFrames := int(math.Ceil(float64(len(b)) / float64(MAX_PAYLOAD)))
	frames := make([][]byte, numFrames)
	for i := 0; i < numFrames; i++ {
		length := MAX_PAYLOAD
		typ := 0
		flag := 0
		if once && i == numFrames-1 {
			length = len(b) - (i * MAX_PAYLOAD)
			flag = 1
		}
		frame := make([]byte, 8+length)

		binary.BigEndian.PutUint16(frame[0:2], uint16(length))
		frame[2] = uint8(typ)
		frame[3] = uint8(flag)
		binary.BigEndian.PutUint32(frame[4:8], id)
		payload := b[i*MAX_PAYLOAD : i*MAX_PAYLOAD+length]
		for j, _ := range payload {
			frame[8+j] = payload[j]
		}
		frames[i] = frame
	}
	return frames
}

func NewFrameHeader(b []byte) FrameHeader {
	return FrameHeader{
		length:   binary.BigEndian.Uint16(b[0:2]),
		typ:      uint8(b[2]),
		flag:     uint8(b[3]),
		streamId: binary.BigEndian.Uint32(b[4:]),
	}
}

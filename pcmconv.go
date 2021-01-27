package main

import (
	_ "bytes"
	"encoding/binary"
)

func convert16to8(preData, postData []byte, postDataLen int) {
	for pos := 0; pos < postDataLen; pos += 1 {
		//data := make([]byte, 2, 2)
		//data = preData[2*pos : 2*pos+2]

		data := binary.LittleEndian.Uint16(preData[2*pos:])		

		var frame int16
		frame = int16(data)
		//frame = int16(data[1])
		//frame = (frame << 8)
		//frame += int16(data[0])

		var a uint16 // A-law value we are forming
		var b byte

		// -ve value
		// Note, ones compliment is used here as this keeps encoding symetrical
		// and equal spaced around zero cross-over, (it also matches the standard).
		if frame < 0 {
			frame = ^frame
			a = 0x00 // sign = 0
		} else {
			// +ve value
			a = 0x80 // sign = 1
		}

		// Calculate segment and interval numbers
		frame = (frame >> 4)
		if frame > 0x20 {
			if frame >= 0x100 {
				frame = (frame >> 4)
				a += 0x40
			}

			if frame >= 0x40 {
				frame = (frame >> 2)
				a += 0x20
			}

			if frame >= 0x20 {
				frame = (frame >> 1)
				a += 0x10
			}
		}
		// a&0x70 now holds segment value and 'p' the interval number

		a += uint16(frame) // a now equal to encoded A-law value
		a = a ^ 0x55
		b = byte(a)

		postData[pos] = b
	}
}

func convert8to16(preData, postData []byte, preDataLen int) {
	for pos := 0; pos < preDataLen; pos += 1 {
		var alaw byte
		alaw = preData[pos]
		alaw ^= 0x55 // A-law has alternate bits inverted for transmission

		var sign uint16
		sign = uint16(alaw & 0x80)

		var linear int16
		linear = int16(alaw & 0x1f)
		linear = (linear << 4)
		linear += 8 // Add a 'half' bit (0x08) to place PCM value in middle of range

		alaw &= 0x7f
		if alaw > 0x20 {
			linear |= 0x100
			var shift uint16
			shift = uint16((alaw >> 4) - 1)
			linear <<= shift
		}

		if sign == 0 {
			linear = -linear
			postData[2*pos] = byte(linear % 0x100)
			postData[2*pos+1] = byte(linear >> 8)
			//buf := bytes.NewBuffer([]byte{})
			//binary.Write(buf, binary.LittleEndian, linear)
		} else {
			postData[2*pos] = byte(linear % 0x100)
			postData[2*pos+1] = byte(linear >> 8)
		}
	}
}

package main

import (
	"encoding/binary"
)

const (
	Audio_DecodeShortSize = 240
	Audio_DecodeDataSize  = 480
)

func convert16to8(preData, postData []byte, postDataLen int) {
	for pos := 0; pos < postDataLen; pos += 1 {
		data := binary.LittleEndian.Uint16(preData[2*pos:])
		frame := int16(data)

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
		alaw := preData[pos]
		alaw ^= 0x55 // A-law has alternate bits inverted for transmission

		sign := uint16(alaw & 0x80)

		linear := int16(alaw & 0x1f)
		linear = (linear << 4)
		linear += 8 // Add a 'half' bit (0x08) to place PCM value in middle of range

		alaw &= 0x7f
		if alaw > 0x20 {
			linear |= 0x100
			shift := uint16((alaw >> 4) - 1)
			linear <<= shift
		}

		if sign == 0 {
			linear = -linear
		}
		binary.LittleEndian.PutUint16(postData[2*pos:], uint16(linear))
	}
}

/*
func mix(pOutBuf, pInBuf1, pInBuf2 []int16) {
	packLen := Audio_DecodeShortSize
	len1, len2 := len(pInBuf1), len(InBuf2)

}

void mix(short* pOutBuf, short* pInBuf1, short* pInBuf2) {
    int iTemp = Audio_DecodeShortSize;
    if (pInBuf2) {
        while (iTemp--) {
            *(pOutBuf++) = ((*(pInBuf1++)) >> 1) + ((*(pInBuf2++)) >> 1);
        }
    } else {
        while (iTemp--) {
            *(pOutBuf++) = ((*(pInBuf1++)) >> 1);
        }
    }
}
*/

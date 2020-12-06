package utils

import (
	"encoding/binary"
	"math"
)

// Returns the 4 byte array corresponding the the given int value
func ConvertIntToByteArray(i int) []uint8 {
	b := make([]uint8, 4)
	b[0] = uint8(i)
	b[1] = uint8(i >> 8)
	b[2] = uint8(i >> 16)
	b[3] = uint8(i >> 24)
	return b
}

// Returns a byte array containing the float32 representation of the float64 values provided by the input slice
func ConvertTruncateFloat64ToFloat32ByteArray(inData []float64) []uint8 {
	j := 0
	length := len(inData)
	outData := make([]byte, length*4) // Cast float64 to float32
	for i := 0; i < length; i++ {
		bytes := make([]byte, 4)
		binary.LittleEndian.PutUint32(bytes, math.Float32bits(float32(inData[i])))
		outData[j] = bytes[0]
		j++
		outData[j] = bytes[1]
		j++
		outData[j] = bytes[2]
		j++
		outData[j] = bytes[3]
		j++
	}
	return outData
}


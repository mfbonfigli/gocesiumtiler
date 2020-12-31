package unit_test

import (
	"bytes"
	"encoding/binary"
	"github.com/mfbonfigli/gocesiumtiler/tools"
	"math"
	"testing"
)

func TestConvertIntToByteArrayZero(t *testing.T) {
	var testData = map[int][]uint8{
		0:          {0, 0, 0, 0},
		1:          {1, 0, 0, 0},
		255:        {255, 0, 0, 0},
		256:        {0, 1, 0, 0},
		257:        {1, 1, 0, 0},
		511:        {255, 1, 0, 0},
		512:        {0, 2, 0, 0},
		65536:      {0, 0, 1, 0},
		65537:      {1, 0, 1, 0},
		16777216:   {0, 0, 0, 1},
		4294967295: {255, 255, 255, 255},
		4294967296: {0, 0, 0, 0},
	}

	for input, expected := range testData {
		if !bytes.Equal(tools.ConvertIntToByteArray(input), expected) {
			t.Errorf("Expected byte array does not match output byte array")
		}
	}
}

func TestConvertTruncateFloat64ToFloat32ByteArray(t *testing.T) {
	var testData = []float64{
		-1.234598e35,
		-3.156894,
		-1e-9,
		0.0,
		1e-9,
		0.651654990098787,
		3.156894,
		1.234598e35,
	}

	for _, input := range testData {
		expected := make([]byte, 4)
		binary.LittleEndian.PutUint32(expected, math.Float32bits(float32(input)))
		if !bytes.Equal(tools.ConvertTruncateFloat64ToFloat32ByteArray([]float64{input}), expected) {
			t.Errorf("Expected byte array does not match output byte array")
		}

	}
}

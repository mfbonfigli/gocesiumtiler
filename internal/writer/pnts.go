package writer

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"github.com/mfbonfigli/gocesiumtiler/v2/internal/geom"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/tree"
	"github.com/mfbonfigli/gocesiumtiler/v2/internal/utils"
	"github.com/mfbonfigli/gocesiumtiler/v2/version"
)

// PntsEncoder writes a node data as Pnts file (3D Tiles 1.0 specs)
type PntsEncoder struct {
	filename string
}

func (e *PntsEncoder) TilesetVersion() version.TilesetVersion {
	return version.TilesetVersion_1_0
}

func NewPntsEncoder(filename string) *PntsEncoder {
	return &PntsEncoder{filename: filename}
}

func (e *PntsEncoder) Write(node tree.Node, folderPath string, prefix string) error {
	pts := node.Points()

	// Feature table
	featureTableBytes, featureTableLen := e.generateFeatureTable(pts.Len())

	// Batch table
	batchTableBytes, batchTableLen := e.generateBatchTable(pts.Len())

	// Write binary content to file
	pntsFilePath := path.Join(folderPath, prefix+e.filename)
	f, err := os.Create(pntsFilePath)
	if err != nil {
		return err
	}
	defer f.Close()

	wr := bufio.NewWriter(f)

	err = e.writePntsHeader(pts.Len(), featureTableLen, batchTableLen, wr)
	if err != nil {
		return err
	}
	err = e.writeTable(featureTableBytes, wr)
	if err != nil {
		return err
	}

	err = e.writePointCoords(pts, wr)
	if err != nil {
		return err
	}

	err = e.writePointColors(pts, wr)
	if err != nil {
		return err
	}

	err = e.writeTable(batchTableBytes, wr)
	if err != nil {
		return err
	}

	err = e.writePointIntensities(pts, wr)
	if err != nil {
		return err
	}

	err = e.writePointClassifications(pts, wr)
	if err != nil {
		return err
	}

	err = wr.Flush()
	if err != nil {
		return err
	}

	return nil
}

func (e *PntsEncoder) generateFeatureTable(numPoints int) ([]byte, int) {
	featureTableStr := e.generateFeatureTableJsonContent(numPoints, 0)
	featureTableLen := len(featureTableStr)
	return []byte(featureTableStr), featureTableLen
}

func (e *PntsEncoder) generateBatchTable(numPoints int) ([]byte, int) {
	batchTableStr := e.generateBatchTableJsonContent(numPoints, 0)
	batchTableLen := len(batchTableStr)
	return []byte(batchTableStr), batchTableLen
}

func (e *PntsEncoder) writePntsHeader(numPoints int, featureTableLen int, batchTableLen int, wr io.Writer) error {
	_, err := wr.Write([]byte("pnts")) // magic
	if err != nil {
		return err
	}
	err = utils.WriteIntAs4ByteNumber(1, wr) // version number
	if err != nil {
		return err
	}
	positionBytesLen := 4 * 3 * numPoints                                                  // 4 bytes per coordinate component (x,y,z) -> 12 bytes per point
	err = utils.WriteIntAs4ByteNumber(28+featureTableLen+positionBytesLen+numPoints*3, wr) // numpoints*3 is colorbytes (1 byte per color component)
	if err != nil {
		return err
	}
	err = utils.WriteIntAs4ByteNumber(featureTableLen, wr) // feature table length
	if err != nil {
		return err
	}
	err = utils.WriteIntAs4ByteNumber(positionBytesLen+numPoints*3, wr) // feature table binary length (position len + colors len)
	if err != nil {
		return err
	}
	err = utils.WriteIntAs4ByteNumber(batchTableLen, wr) // batch table length
	if err != nil {
		return err
	}
	err = utils.WriteIntAs4ByteNumber(2*numPoints, wr) // intensity + classification
	if err != nil {
		return err
	}
	return nil
}

func (e *PntsEncoder) writeTable(tableBytes []byte, wr io.Writer) error {
	_, err := wr.Write(tableBytes)
	if err != nil {
		return err
	}
	return nil
}

func (e *PntsEncoder) writePointCoords(pts geom.PointList, wr io.Writer) error {
	n := pts.Len()
	// write coords
	for i := 0; i < n; i++ {
		pt, err := pts.Next()
		if err != nil {
			return err
		}
		err = utils.WriteFloat32LittleEndian(pt.X, wr)
		if err != nil {
			return err
		}
		err = utils.WriteFloat32LittleEndian(pt.Y, wr)
		if err != nil {
			return err
		}
		err = utils.WriteFloat32LittleEndian(pt.Z, wr)
		if err != nil {
			return err
		}
	}
	pts.Reset()
	return nil
}

func (e *PntsEncoder) writePointColors(pts geom.PointList, wr io.Writer) error {
	n := pts.Len()
	// write colors
	for i := 0; i < n; i++ {
		pt, err := pts.Next()
		if err != nil {
			return err
		}
		_, err = wr.Write([]byte{pt.R, pt.G, pt.B})
		if err != nil {
			return err
		}
	}
	pts.Reset()
	return nil
}

func (e *PntsEncoder) writePointIntensities(pts geom.PointList, wr io.Writer) error {
	n := pts.Len()
	// write colors
	for i := 0; i < n; i++ {
		pt, err := pts.Next()
		if err != nil {
			return err
		}
		_, err = wr.Write([]byte{pt.Intensity})
		if err != nil {
			return err
		}
	}
	pts.Reset()
	return nil
}

func (e *PntsEncoder) writePointClassifications(pts geom.PointList, wr io.Writer) error {
	n := pts.Len()
	// write colors
	for i := 0; i < n; i++ {
		pt, err := pts.Next()
		if err != nil {
			return err
		}
		_, err = wr.Write([]byte{pt.Classification})
		if err != nil {
			return err
		}
	}
	pts.Reset()
	return nil
}

// Generates the json representation of the feature table
func (e *PntsEncoder) generateFeatureTableJsonContent(pointNo int, spaceNo int) string {
	s := fmt.Sprintf(`{"POINTS_LENGTH":%d,"POSITION":{"byteOffset":0},"RGB":{"byteOffset":%d}}%s`,
		pointNo,
		pointNo*12,
		strings.Repeat(" ", spaceNo),
	)
	headerByteLength := len([]byte(s))
	paddingSize := headerByteLength % 4
	if paddingSize != 0 {
		return e.generateFeatureTableJsonContent(pointNo, 4-paddingSize)
	}
	return s
}

// Generates the json representation of the batch table
func (e *PntsEncoder) generateBatchTableJsonContent(pointNumber, spaceNumber int) string {
	s := fmt.Sprintf(`{"INTENSITY":{"byteOffset":0,"componentType":"UNSIGNED_BYTE","type":"SCALAR"},
	"CLASSIFICATION":{"byteOffset":%d,"componentType":"UNSIGNED_BYTE","type":"SCALAR"}}%s`, 1*pointNumber, strings.Repeat(" ", spaceNumber))
	headerByteLength := len([]byte(s))
	paddingSize := headerByteLength % 4
	if paddingSize != 0 {
		return e.generateBatchTableJsonContent(pointNumber, 4-paddingSize)
	}
	return s
}

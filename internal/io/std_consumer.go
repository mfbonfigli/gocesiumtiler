package io

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mfbonfigli/gocesiumtiler/internal/converters"
	"github.com/mfbonfigli/gocesiumtiler/internal/geometry"
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"github.com/mfbonfigli/gocesiumtiler/tools"
	"io/ioutil"
	"path"
	"strconv"
	"strings"
	"sync"
)

type StandardConsumer struct {
	coordinateConverter converters.CoordinateConverter
}

func NewStandardConsumer(coordinateConverter converters.CoordinateConverter) *StandardConsumer {
	return &StandardConsumer{
		coordinateConverter: coordinateConverter,
	}
}

// struct used to store data in an intermediate format
type intermediateData struct {
	coords          []float64
	colors          []uint8
	intensities     []uint8
	classifications []uint8
	numPoints       int
}

// Continually consumes WorkUnits submitted to a work channel producing corresponding content.pnts files and tileset.json files
// continues working until work channel is closed or if an error is raised. In this last case submits the error to an error
// channel before quitting
func (c *StandardConsumer) Consume(workchan chan *WorkUnit, errchan chan error, waitGroup *sync.WaitGroup) {
	for {
		// get work from channel
		work, ok := <-workchan
		if !ok {
			// channel was closed by producer, quit infinite loop
			break
		}

		// do work
		err := c.doWork(work)

		// if there were errors during work send in error channel and quit
		if err != nil {
			errchan <- err
			fmt.Println("exception in c worker")
			break
		}
	}

	// signal waitgroup finished work
	waitGroup.Done()
}

// Takes a workunit and writes the corresponding content.pnts and tileset.json files
func (c *StandardConsumer) doWork(workUnit *WorkUnit) error {
	// writes the content.pnts file
	err := c.writeBinaryPntsFile(*workUnit)
	if err != nil {
		return err
	}
	if !workUnit.Node.IsLeaf() || workUnit.Node.IsRoot() {
		// if the node has children also writes the tileset.json file
		err := c.writeTilesetJsonFile(*workUnit)
		if err != nil {
			return err
		}
	}
	return nil
}

// Writes a content.pnts binary files from the given WorkUnit
func (c *StandardConsumer) writeBinaryPntsFile(workUnit WorkUnit) error {
	parentFolder := workUnit.BasePath
	node := workUnit.Node

	// Create base folder if it does not exist
	err := tools.CreateDirectoryIfDoesNotExist(parentFolder)
	if err != nil {
		return err
	}

	pointNo := len(node.GetPoints())
	intermediatePointData, err := c.generateIntermediateDataForPnts(node, pointNo)
	if err != nil {
		return err
	}

	// Evaluating average X, Y, Z to express coords relative to tile center
	averageXYZ := c.computeAverageXYZ(intermediatePointData)

	// Normalizing coordinates relative to average
	c.subtractXYZFromIntermediateDataCoords(intermediatePointData, averageXYZ)

	// Coordinate bytes
	positionBytes := tools.ConvertTruncateFloat64ToFloat32ByteArray(intermediatePointData.coords)

	// Feature table
	featureTableBytes, featureTableLen := c.generateFeatureTable(averageXYZ[0], averageXYZ[1], averageXYZ[2], pointNo)

	// Batch table
	batchTableBytes, batchTableLen := c.generateBatchTable(pointNo)

	// Appending binary content to slice
	outputByte := c.generatePntsByteArray(intermediatePointData, positionBytes, featureTableBytes, featureTableLen, batchTableBytes, batchTableLen)

	// Write binary content to file
	pntsFilePath := path.Join(parentFolder, "content.pnts")
	err = ioutil.WriteFile(pntsFilePath, outputByte, 0777)

	if err != nil {
		return err
	}

	return nil
}

func (c *StandardConsumer) generateIntermediateDataForPnts(node octree.INode, numPoints int) (*intermediateData, error) {
	intermediateData := intermediateData{
		coords:          make([]float64, numPoints*3),
		colors:          make([]uint8, numPoints*3),
		intensities:     make([]uint8, numPoints),
		classifications: make([]uint8, numPoints),
		numPoints:       numPoints,
	}

	// Decomposing tile data properties in separate sublists for coords, colors, intensities and classifications
	for i := 0; i < len(node.GetPoints()); i++ {
		point := node.GetPoints()[i]
		if point == nil {
			fmt.Println("a")
		}
		srcCoord := geometry.Coordinate{
			X: point.X,
			Y: point.Y,
			Z: point.Z,
		}

		// ConvertCoordinateSrid coords according to cesium CRS
		outCrd, err := c.coordinateConverter.ConvertToWGS84Cartesian(srcCoord, node.GetInternalSrid())
		if err != nil {
			return nil, err
		}

		intermediateData.coords[i*3] = outCrd.X
		intermediateData.coords[i*3+1] = outCrd.Y
		intermediateData.coords[i*3+2] = outCrd.Z

		intermediateData.colors[i*3] = point.R
		intermediateData.colors[i*3+1] = point.G
		intermediateData.colors[i*3+2] = point.B

		intermediateData.intensities[i] = point.Intensity
		intermediateData.classifications[i] = point.Classification
	}

	return &intermediateData, nil
}

func (c *StandardConsumer) generateFeatureTable(avgX float64, avgY float64, avgZ float64, numPoints int) ([]byte, int) {
	featureTableStr := c.generateFeatureTableJsonContent(avgX, avgY, avgZ, numPoints, 0)
	featureTableLen := len(featureTableStr)
	return []byte(featureTableStr), featureTableLen
}

func (c *StandardConsumer) generateBatchTable(numPoints int) ([]byte, int) {
	batchTableStr := c.generateBatchTableJsonContent(numPoints, 0)
	batchTableLen := len(batchTableStr)
	return []byte(batchTableStr), batchTableLen
}

func (c *StandardConsumer) generatePntsByteArray(intermediateData *intermediateData, positionBytes []byte, featureTableBytes []byte, featureTableLen int, batchTableBytes []byte, batchTableLen int) []byte {
	outputByte := make([]byte, 0)
	outputByte = append(outputByte, []byte("pnts")...)                 // magic
	outputByte = append(outputByte, tools.ConvertIntToByteArray(1)...) // version number
	byteLength := 28 + featureTableLen + len(positionBytes) + len(intermediateData.colors)
	outputByte = append(outputByte, tools.ConvertIntToByteArray(byteLength)...)
	outputByte = append(outputByte, tools.ConvertIntToByteArray(featureTableLen)...)                                                         // feature table length
	outputByte = append(outputByte, tools.ConvertIntToByteArray(len(positionBytes)+len(intermediateData.colors))...)                         // feature table binary length
	outputByte = append(outputByte, tools.ConvertIntToByteArray(batchTableLen)...)                                                           // batch table length
	outputByte = append(outputByte, tools.ConvertIntToByteArray(len(intermediateData.intensities)+len(intermediateData.classifications))...) // batch table binary length
	outputByte = append(outputByte, featureTableBytes...)                                                                                    // feature table
	outputByte = append(outputByte, positionBytes...)                                                                                        // positions array
	outputByte = append(outputByte, intermediateData.colors...)                                                                              // colors array
	outputByte = append(outputByte, batchTableBytes...)                                                                                      // batch table
	outputByte = append(outputByte, intermediateData.intensities...)                                                                         // intensities array
	outputByte = append(outputByte, intermediateData.classifications...)

	return outputByte
}

func (c *StandardConsumer) computeAverageXYZ(intermediatePointData *intermediateData) []float64 {
	var avgX, avgY, avgZ float64

	for i := 0; i < intermediatePointData.numPoints; i++ {
		avgX = avgX + intermediatePointData.coords[i*3]
		avgY = avgY + intermediatePointData.coords[i*3+1]
		avgZ = avgZ + intermediatePointData.coords[i*3+2]
	}
	avgX /= float64(intermediatePointData.numPoints)
	avgY /= float64(intermediatePointData.numPoints)
	avgZ /= float64(intermediatePointData.numPoints)

	return []float64{avgX, avgY, avgZ}
}

func (c *StandardConsumer) subtractXYZFromIntermediateDataCoords(intermediatePointData *intermediateData, xyz []float64) {
	for i := 0; i < intermediatePointData.numPoints; i++ {
		intermediatePointData.coords[i*3] -= xyz[0]
		intermediatePointData.coords[i*3+1] -= xyz[1]
		intermediatePointData.coords[i*3+2] -= xyz[2]
	}
}

// Generates the json representation of the feature table
func (c *StandardConsumer) generateFeatureTableJsonContent(x, y, z float64, pointNo int, spaceNo int) string {
	sb := ""
	sb += "{\"POINTS_LENGTH\":" + strconv.Itoa(pointNo) + ","
	sb += "\"RTC_CENTER\":[" + fmt.Sprintf("%f", x) + strings.Repeat("0", spaceNo)
	sb += "," + fmt.Sprintf("%f", y) + "," + fmt.Sprintf("%f", z) + "],"
	sb += "\"POSITION\":" + "{\"byteOffset\":" + "0" + "},"
	sb += "\"RGB\":" + "{\"byteOffset\":" + strconv.Itoa(pointNo*12) + "}}"
	headerByteLength := len([]byte(sb))
	paddingSize := headerByteLength % 4
	if paddingSize != 0 {
		return c.generateFeatureTableJsonContent(x, y, z, pointNo, 4-paddingSize)
	}
	return sb
}

// Generates the json representation of the batch table
func (c *StandardConsumer) generateBatchTableJsonContent(pointNumber, spaceNumber int) string {
	sb := ""
	sb += "{\"INTENSITY\":" + "{\"byteOffset\":" + "0" + ", \"componentType\":\"UNSIGNED_BYTE\", \"type\":\"SCALAR\"},"
	sb += "\"CLASSIFICATION\":" + "{\"byteOffset\":" + strconv.Itoa(pointNumber) + ", \"componentType\":\"UNSIGNED_BYTE\", \"type\":\"SCALAR\"}}"
	sb += strings.Repeat(" ", spaceNumber)
	headerByteLength := len([]byte(sb))
	paddingSize := headerByteLength % 4
	if paddingSize != 0 {
		return c.generateBatchTableJsonContent(pointNumber, 4-paddingSize)
	}
	return sb
}

// Writes the tileset.json file for the given WorkUnit
func (c *StandardConsumer) writeTilesetJsonFile(workUnit WorkUnit) error {
	parentFolder := workUnit.BasePath
	node := workUnit.Node

	// Create base folder if it does not exist
	err := tools.CreateDirectoryIfDoesNotExist(parentFolder)
	if err != nil {
		return err
	}

	// tileset.json file
	file := path.Join(parentFolder, "tileset.json")
	jsonData, err := c.generateTilesetJson(node)
	if err != nil {
		return err
	}

	// Writes the tileset.json binary content to the given file
	err = ioutil.WriteFile(file, jsonData, 0666)
	if err != nil {
		return err
	}

	return nil
}

// Generates the tileset.json content for the given tree node
func (c *StandardConsumer) generateTilesetJson(node octree.INode) ([]byte, error) {
	if !node.IsLeaf() || node.IsRoot() {
		root, err := c.generateTilesetRoot(node)
		if err != nil {
			return nil, err
		}

		tileset := *c.generateTileset(node, root)

		// Outputting a formatted json file
		e, err := json.MarshalIndent(tileset, "", "\t")
		if err != nil {
			return nil, err
		}

		return e, nil
	}

	return nil, errors.New("this node is a leaf, cannot create a tileset json for it")
}

func (c *StandardConsumer) generateTilesetRoot(node octree.INode) (*Root, error) {
	reg, err := node.GetBoundingBoxRegion(c.coordinateConverter)

	if err != nil {
		return nil, err
	}

	children, err := c.generateTilesetChildren(node)
	if err != nil {
		return nil, err
	}

	root := Root{
		Content:        Content{"content.pnts"},
		BoundingVolume: BoundingVolume{reg.GetAsArray()},
		GeometricError: node.ComputeGeometricError(),
		Refine:         "ADD",
		Children:       children,
	}

	return &root, nil
}

func (c *StandardConsumer) generateTileset(node octree.INode, root *Root) *Tileset {
	tileset := Tileset{}
	tileset.Asset = Asset{Version: "1.0"}
	tileset.GeometricError = node.ComputeGeometricError()
	tileset.Root = *root

	return &tileset
}

func (c *StandardConsumer) generateTilesetChildren(node octree.INode) ([]Child, error) {
	var children []Child
	for i, child := range node.GetChildren() {
		if c.nodeContainsPoints(child) {
			childJson, err := c.generateTilesetChild(child, i)
			if err != nil {
				return nil, err
			}
			children = append(children, *childJson)
		}
	}
	return children, nil
}

func (c *StandardConsumer) nodeContainsPoints(node octree.INode) bool {
	return node != nil && node.TotalNumberOfPoints() > 0
}

func (c *StandardConsumer) generateTilesetChild(child octree.INode, childIndex int) (*Child, error) {
	childJson := Child{}
	filename := "tileset.json"
	if child.IsLeaf() {
		filename = "content.pnts"
	}
	childJson.Content = Content{
		Url: strconv.Itoa(childIndex) + "/" + filename,
	}
	reg, err := child.GetBoundingBoxRegion(c.coordinateConverter)
	if err != nil {
		return nil, err
	}
	childJson.BoundingVolume = BoundingVolume{
		Region: reg.GetAsArray(),
	}
	childJson.GeometricError = child.ComputeGeometricError()
	childJson.Refine = "ADD"
	return &childJson, nil
}

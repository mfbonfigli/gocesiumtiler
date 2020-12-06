package io

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
	"github.com/mfbonfigli/gocesiumtiler/converters"
	"github.com/mfbonfigli/gocesiumtiler/structs/data"
	"github.com/mfbonfigli/gocesiumtiler/structs/geometry"
	"github.com/mfbonfigli/gocesiumtiler/structs/octree"
	"github.com/mfbonfigli/gocesiumtiler/structs/tiler"
	"github.com/mfbonfigli/gocesiumtiler/utils"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
)

// Continually consumes WorkUnits submitted to a work channel producing corresponding content.pnts files and tileset.json files
// continues working until work channel is closed or if an error is raised. In this last case submits the error to an error
// channel before quitting
func Consume(workchan chan *WorkUnit, errchan chan error, wg *sync.WaitGroup, converter converters.CoordinateConverter) {
	for {
		// get work from channel
		work, ok := <-workchan
		if !ok {
			// channel was closed by producer, quit infinite loop
			break
		}

		// do work
		err := doWork(work, converter)

		// if there were errors during work send in error channel and quit
		if err != nil {
			errchan <- err
			fmt.Println("exception in consumer worker")
			break
		}
	}

	// signal waitgroup finished work
	wg.Done()
}

// Takes a workunit and writes the corresponding content.pnts and tileset.json files
func doWork(workUnit *WorkUnit, coordinateConverter converters.CoordinateConverter) error {
	// writes the content.pnts file
	err := writeBinaryPntsFile(*workUnit, coordinateConverter)
	if err != nil {
		return err
	}
	if !workUnit.OctNode.IsLeaf || workUnit.OctNode.Parent == nil {
		// if the node has children also writes the tileset.json file
		err := writeTilesetJsonFile(*workUnit, coordinateConverter)
		if err != nil {
			return err
		}
	}
	return nil
}

// Writes a content.pnts binary files from the given WorkUnit
func writeBinaryPntsFile(workUnit WorkUnit, coordinateConverter converters.CoordinateConverter) error {
	parentFolder := workUnit.BasePath
	node := workUnit.OctNode

	// Create base folder if it does not exist
	if _, err := os.Stat(parentFolder); os.IsNotExist(err) {
		err := os.MkdirAll(parentFolder, 0777)
		if err != nil {
			return err
		}
	}

	// Constructing pnts output file path
	pntsFilePath := path.Join(parentFolder, "content.pnts")

	pointNo := len(node.Items)
	coords := make([]float64, pointNo*3)
	colors := make([]uint8, pointNo*3)
	intensities := make([]uint8, pointNo)
	classifications := make([]uint8, pointNo)

	// Decomposing tile data properties in separate sublists for coords, colors, intensities and classifications
	for i := 0; i < len(node.Items); i++ {
		element := node.Items[i]
		srcCoord := geometry.Coordinate{
			X: &element.X,
			Y: &element.Y,
			Z: &element.Z,
		}

		// ConvertCoordinateSrid coords according to cesium CRS
		outCrd, err := coordinateConverter.ConvertToWGS84Cartesian(srcCoord, workUnit.Opts.Srid)
		if err != nil {
			return err
		}

		coords[i*3] = *outCrd.X
		coords[i*3+1] = *outCrd.Y
		coords[i*3+2] = *outCrd.Z

		colors[i*3] = element.R
		colors[i*3+1] = element.G
		colors[i*3+2] = element.B

		intensities[i] = element.Intensity
		classifications[i] = element.Classification

	}

	// Evaluating average X, Y, Z to express coords relative to tile center
	var avgX, avgY, avgZ float64
	for i := 0; i < pointNo; i++ {
		avgX = avgX + coords[i*3]
		avgY = avgY + coords[i*3+1]
		avgZ = avgZ + coords[i*3+2]
	}
	avgX /= float64(pointNo)
	avgY /= float64(pointNo)
	avgZ /= float64(pointNo)

	// Normalizing coordinates relative to average
	for i := 0; i < pointNo; i++ {
		coords[i*3] -= avgX
		coords[i*3+1] -= avgY
		coords[i*3+2] -= avgZ
	}
	positionBytes := utils.ConvertTruncateFloat64ToFloat32ByteArray(coords)

	// Feature table
	featureTableStr := generateFeatureTableJsonContent(avgX, avgY, avgZ, pointNo, 0)
	featureTableLen := len(featureTableStr)
	featureTableBytes := []byte(featureTableStr)

	// Batch table
	batchTableStr := generateBatchTableJsonContent(pointNo, 0)
	batchTableLen := len(batchTableStr)
	batchTableBytes := []byte(batchTableStr)

	// Appending binary content to slice
	outputByte := make([]byte, 0)
	outputByte = append(outputByte, []byte("pnts")...)                 // magic
	outputByte = append(outputByte, utils.ConvertIntToByteArray(1)...) // version number
	byteLength := 28 + featureTableLen + len(positionBytes) + len(colors)
	outputByte = append(outputByte, utils.ConvertIntToByteArray(byteLength)...)
	outputByte = append(outputByte, utils.ConvertIntToByteArray(featureTableLen)...)                       // feature table length
	outputByte = append(outputByte, utils.ConvertIntToByteArray(len(positionBytes)+len(colors))...)        // feature table binary length
	outputByte = append(outputByte, utils.ConvertIntToByteArray(batchTableLen)...)                         // batch table length
	outputByte = append(outputByte, utils.ConvertIntToByteArray(len(intensities)+len(classifications))...) // batch table binary length
	outputByte = append(outputByte, featureTableBytes...)                                                  // feature table
	outputByte = append(outputByte, positionBytes...)                                                      // positions array
	outputByte = append(outputByte, colors...)                                                             // colors array
	outputByte = append(outputByte, batchTableBytes...)                                                    // batch table
	outputByte = append(outputByte, intensities...)                                                        // intensities array
	outputByte = append(outputByte, classifications...)                                                    // classifications array

	// Write binary content to file
	err := ioutil.WriteFile(pntsFilePath, outputByte, 0777)

	if err != nil {
		return err
	}
	return nil
}

// Generates the json representation of the feature table
func generateFeatureTableJsonContent(x, y, z float64, pointNo int, spaceNo int) string {
	sb := ""
	sb += "{\"POINTS_LENGTH\":" + strconv.Itoa(pointNo) + ","
	sb += "\"RTC_CENTER\":[" + fmt.Sprintf("%f", x) + strings.Repeat("0", spaceNo)
	sb += "," + fmt.Sprintf("%f", y) + "," + fmt.Sprintf("%f", z) + "],"
	sb += "\"POSITION\":" + "{\"byteOffset\":" + "0" + "},"
	sb += "\"RGB\":" + "{\"byteOffset\":" + strconv.Itoa(pointNo*12) + "}}"
	headerByteLength := len([]byte(sb))
	paddingSize := headerByteLength % 4
	if paddingSize != 0 {
		return generateFeatureTableJsonContent(x, y, z, pointNo, 4-paddingSize)
	}
	return sb
}

// Generates the json representation of the batch table
func generateBatchTableJsonContent(pointNumber, spaceNumber int) string {
	sb := ""
	sb += "{\"INTENSITY\":" + "{\"byteOffset\":" + "0" + ", \"componentType\":\"UNSIGNED_BYTE\", \"type\":\"SCALAR\"},"
	sb += "\"CLASSIFICATION\":" + "{\"byteOffset\":" + strconv.Itoa(pointNumber) + ", \"componentType\":\"UNSIGNED_BYTE\", \"type\":\"SCALAR\"}}"
	sb += strings.Repeat(" ", spaceNumber)
	headerByteLength := len([]byte(sb))
	paddingSize := headerByteLength % 4
	if paddingSize != 0 {
		return generateBatchTableJsonContent(pointNumber, 4-paddingSize)
	}
	return sb
}

// Writes the tileset.json file for the given WorkUnit
func writeTilesetJsonFile(workUnit WorkUnit, coordinateConverter converters.CoordinateConverter) error {
	parentFolder := workUnit.BasePath
	node := workUnit.OctNode

	// Create base folder if it does not exist
	if _, err := os.Stat(parentFolder); os.IsNotExist(err) {
		err := os.MkdirAll(parentFolder, 0777)
		if err != nil {
			return err
		}
	}

	// tileset.json file
	file := path.Join(parentFolder, "tileset.json")
	jsonData, err := generateTilesetJsonContent(node, workUnit.Opts, coordinateConverter)
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

// Generates the tileset.json content for the given octnode and tileroptions
func generateTilesetJsonContent(node *octree.OctNode, opts *tiler.TilerOptions, converter converters.CoordinateConverter) ([]byte, error) {
	if !node.IsLeaf || node.Parent == nil {
		tileset := Tileset{}
		tileset.Asset = Asset{Version: "1.0"}
		tileset.GeometricError = computeGeometricError(node)
		root := Root{}
		root.Children = []Child{}
		for i, child := range node.Children {
			if child != nil && child.GlobalChildrenCount > 0 {
				childJson := Child{}
				filename := "tileset.json"
				if child.IsLeaf {
					filename = "content.pnts"
				}
				childJson.Content = Content{
					Url: strconv.Itoa(i) + "/" + filename,
				}
				reg, err := converter.Convert2DBoundingboxToWGS84Region(child.BoundingBox, opts.Srid)
				if err != nil {
					return nil, err
				}
				childJson.BoundingVolume = BoundingVolume{
					Region: reg,
				}
				childJson.GeometricError = computeGeometricError(child)
				childJson.Refine = "ADD"
				root.Children = append(root.Children, childJson)
			}
		}
		root.Content = Content{
			Url: "content.pnts",
		}
		reg, err := converter.Convert2DBoundingboxToWGS84Region(node.BoundingBox, opts.Srid)

		if node.Parent == nil && node.IsLeaf {
			// only one tile, no LoDs. Estimate geometric error as lenght of diagonal of region
			var latA = reg[1]
			var latB = reg[3]
			var lngA = reg[0]
			var lngB = reg[2]
			latA = reg[1]
			tileset.GeometricError = 6371000 * math.Acos(math.Cos(latA)*math.Cos(latB)*math.Cos(lngB-lngA)+math.Sin(latA)*math.Sin(latB))
		}

		if err != nil {
			return nil, err
		}
		root.BoundingVolume = BoundingVolume{
			Region: reg,
		}
		root.GeometricError = computeGeometricError(node)
		root.Refine = "ADD"
		tileset.Root = root

		// Outputting a formatted json file
		e, err := json.MarshalIndent(tileset, "", "\t")
		if err != nil {
			return nil, err
		}

		return e, nil
	}

	return nil, errors.New("this node is a leaf, cannot create tileset json for it")
}

// Computes the geometric error for the given OctNode
func computeGeometricError(node *octree.OctNode) float64 {
	volume := node.BoundingBox.GetVolume()
	totalRenderedPoints := int64(node.LocalChildrenCount)
	parent := node.Parent
	for parent != nil {
		for _, e := range parent.Items {
			if canBoundingBoxContainElement(e, node.BoundingBox) {
				totalRenderedPoints++
			}
		}
		parent = parent.Parent
	}
	densityWithAllPoints := math.Pow(volume/float64(totalRenderedPoints+node.GlobalChildrenCount-int64(node.LocalChildrenCount)), 0.333)
	densityWIthOnlyThisTile := math.Pow(volume/float64(totalRenderedPoints), 0.333)

	return densityWIthOnlyThisTile - densityWithAllPoints
}

// Checks if the bounding box contains the given element
func canBoundingBoxContainElement(e *data.Point, bbox *geometry.BoundingBox) bool {
	return (e.X >= bbox.Xmin && e.X <= bbox.Xmax) &&
		(e.Y >= bbox.Ymin && e.Y <= bbox.Ymax) &&
		(e.Z >= bbox.Zmin && e.Z <= bbox.Zmax)
}

package main

import (
	lidario "cesium_tiler/lasread"
	"cesium_tiler/structs"
	"cesium_tiler/structs/octree"
	"fmt"
	"log"
	"time"
)

var xOffset float64 = 0
var yOffset float64 = 0
var zOffset float64 = 0

// Starts the tiling process
func RunTiler(opts *structs.TilerOptions) error {
	defer timeTrack(time.Now(),"tiler")



	fileName := "C:\\Users\\bonfi\\Desktop\\las\\output\\2019-JAN-07_Montesilvano3 Track_B_2019-02-25_11h30_01_331.las"
	// fileName := "C:\\Users\\bonfi\\Desktop\\las\\output\\Chunk 4.las"

	pts, err := readLas(fileName)
	if err != nil {
		return err
	}
	OctTree := octree.NewOctTree(opts)
	err = OctTree.AddItems(pts)
	if err != nil {
		return err
	}

	OctTree.BuildTree()
	OctTree.PrintStructure()
	return nil
}

func readLas(file string) ([]octree.OctElement, error) {
	var lf *lidario.LasFile
	var err error
	lf, err = lidario.NewLasFileForTiler(file)
	if err != nil {
		fmt.Println(err)
	}
	defer lf.Close()
	return lf.GetOctElements(), nil
}


func timeTrack(start time.Time, name string) {
	elapsed := time.Since(start)
	log.Printf("%s took %s", name, elapsed)
}

package octree

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

// Represents an OctTree of OctElements and contains informations needed
// to propagate points in the tree.
type OctTree struct {
	itemsToAdd                         []OctElement
	RootNode                           OctNode
	Built                              bool
	minX, maxX, minY, maxY, minZ, maxZ float64
	Opts                               *TilerOptions
}

// Builds an empty OctTree initializing its properties to the correct defaults
func NewOctTree(opts *TilerOptions) *OctTree {
	return &OctTree{
		itemsToAdd: make([]OctElement, 0),
		Built:      false,
		minX:       math.MaxFloat64,
		minY:       math.MaxFloat64,
		minZ:       math.MaxFloat64,
		maxX:       -1 * math.MaxFloat64,
		maxY:       -1 * math.MaxFloat64,
		maxZ:       -1 * math.MaxFloat64,
		Opts:       opts,
	}
}

func (octTree *OctTree) recomputeBoundsFromElement(element *OctElement) {
	octTree.minX = math.Min(float64(element.X), octTree.minX)
	octTree.minY = math.Min(float64(element.Y), octTree.minY)
	octTree.minZ = math.Min(float64(element.Z), octTree.minZ)
	octTree.maxX = math.Max(float64(element.X), octTree.maxX)
	octTree.maxY = math.Max(float64(element.Y), octTree.maxY)
	octTree.maxZ = math.Max(float64(element.Z), octTree.maxZ)
}

// Adds a splice of pointers to OctElement instances to the OctTree inner list of items to be initialized
func (octTree *OctTree) AddItems(items []OctElement) error {
	if octTree.Built {
		return errors.New("cannot add items to a Built octree")
	}
	octTree.itemsToAdd = append(octTree.itemsToAdd, items...)
	for _, e := range items {
		octTree.recomputeBoundsFromElement(&e)
	}
	octTree.Built = false
	return nil
}

// Builds the hierarchical tree structure propagating the added items according to the TilerOptions provided
// during initialization
func (octTree *OctTree) BuildTree() error {
	if octTree.Built {
		return errors.New("octree already Built")
	}
	octNode := NewOctNode(NewBoundingBox(octTree.minX, octTree.maxX, octTree.minY, octTree.maxY, octTree.minZ, octTree.maxZ), octTree.Opts, 1, nil)
	octTree.RootNode = *octNode
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(octTree.itemsToAdd), func(i, j int) { octTree.itemsToAdd[i], octTree.itemsToAdd[j] = octTree.itemsToAdd[j], octTree.itemsToAdd[i] })

	var wg sync.WaitGroup
	wg.Add(len(octTree.itemsToAdd))

	N := 64
	sem := make(chan struct{}, N)

	for i := 0; i < len(octTree.itemsToAdd); i++ {
		sem <- struct{}{}
		go func(i int) {
			defer wg.Done()
			octTree.RootNode.AddOctElement(&octTree.itemsToAdd[i])
			defer func() {
				<-sem
			}()
		}(i)
	}

	wg.Wait()

	octTree.itemsToAdd = nil
	octTree.Built = true

	return nil
}

func (octTree *OctTree) PrintStructure() {
	if octTree.Built {
		octTree.RootNode.PrintStructure()
	}
}

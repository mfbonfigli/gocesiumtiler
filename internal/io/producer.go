package io

import (
	"github.com/mfbonfigli/gocesiumtiler/internal/octree"
	"sync"
)

type Producer interface {
	Produce(work chan *WorkUnit, wg *sync.WaitGroup, node octree.INode)
}
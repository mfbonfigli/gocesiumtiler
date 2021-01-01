package io

import "sync"

type Consumer interface {
	Consume(workchan chan *WorkUnit, errchan chan error, waitGroup *sync.WaitGroup)
}

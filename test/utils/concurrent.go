package utils

import (
	"sync"

	"github.com/onsi/gomega"
)

type Concurrently struct {
	wg        sync.WaitGroup
	errorChan chan error
}

func NewConcurrently() *Concurrently {
	return &Concurrently{
		errorChan: make(chan error),
		wg:        sync.WaitGroup{},
	}
}

func (c *Concurrently) Run(fn func() error) {
	c.wg.Add(1)

	go func() {
		defer c.wg.Done()
		if err := fn(); err != nil {
			select {
			case c.errorChan <- err:
			default:
			}
		}
	}()
}

func (c *Concurrently) Wait() {
	c.wg.Wait()
	select {
	case err := <-c.errorChan:
		gomega.Expect(err).ToNot(gomega.HaveOccurred())
	default:
	}
}

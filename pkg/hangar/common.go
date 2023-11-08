package hangar

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	// Use 2 routine to handle error messages.
	errorHandlerWorkerNum = 2
)

type common struct {
	// images is the image list.
	images []string
	// imageSpecSet example: map["os"]map["linux"]true
	imageSpecSet map[string]map[string]bool
	// timeout when copy image
	timeout time.Duration
	// workers is the number of wroker
	workers int
	// waitGroup is a WaitGroup to wait for all workers finished
	waitGroup *sync.WaitGroup
	// errorWaitGroup is a WaitGroup to wait for all error routine finished
	errorWaitGroup *sync.WaitGroup
	// objectCh is a channel for sending object to worker
	objectCh chan any
	// objectCtx is the context for handle object
	objectCtx context.Context
	// errorCh is a channel to receive error message
	errorCh chan error
	// errorCtx is the context for handle error message
	errorCtx context.Context
	// failedImageList stores the images failed to copy (thread-unsafe)
	failedImageSet map[string]bool
	// failedImageListMutex is a mutex for read/write of failedImageList
	failedImageListMutex *sync.RWMutex
}

type CommonOpts struct {
	Images  []string
	Arch    []string
	OS      []string
	Variant []string
	Timeout time.Duration
	Workers int
}

func newCommon(o *CommonOpts) *common {
	c := &common{
		images: make([]string, len(o.Images)),

		imageSpecSet: map[string]map[string]bool{
			"os":      make(map[string]bool),
			"arch":    make(map[string]bool),
			"variant": make(map[string]bool),
		},

		timeout:        o.Timeout,
		workers:        o.Workers,
		waitGroup:      &sync.WaitGroup{},
		errorWaitGroup: &sync.WaitGroup{},

		objectCh: make(chan any),
		errorCh:  make(chan error),

		failedImageSet:       make(map[string]bool),
		failedImageListMutex: &sync.RWMutex{},
	}

	copy(c.images, o.Images)
	for i := 0; i < len(o.OS); i++ {
		c.imageSpecSet["os"][o.OS[i]] = true
	}
	for i := 0; i < len(o.Arch); i++ {
		c.imageSpecSet["arch"][o.Arch[i]] = true
	}
	for i := 0; i < len(o.Variant); i++ {
		c.imageSpecSet["variant"][o.Variant[i]] = true
	}

	return c
}

func (c *common) initWorker(ctx context.Context, f func(context.Context, any)) {
	c.objectCtx = ctx
	for i := 0; i < c.workers && i < len(c.images); i++ {
		c.waitGroup.Add(1)
		go c.workerFunc(i, f)
	}
}

func (c *common) workerFunc(id int, f func(context.Context, any)) {
	defer c.waitGroup.Done()
	for {
		select {
		case <-c.objectCtx.Done():
			logrus.Infof("Worker [%d] stopped gracefully: %v",
				id, c.objectCtx.Err())
			return
		case obj, ok := <-c.objectCh:
			if !ok {
				logrus.Debugf("Worker channel closed")
				return
			}
			if obj == nil {
				continue
			}
			f(c.objectCtx, obj)
		}
	}
}

func (c *common) initErrorHandler(ctx context.Context) {
	c.errorCtx = ctx
	c.errorWaitGroup.Add(errorHandlerWorkerNum)
	for i := 0; i < errorHandlerWorkerNum; i++ {
		go func() {
			defer c.errorWaitGroup.Done()
			for {
				select {
				case <-ctx.Done():
					logrus.Debugf("Error handler stopped gracefully: %v", ctx.Err())
					return
				case err, ok := <-c.errorCh:
					if !ok {
						logrus.Debugf("Error handler channel closed")
						return
					}
					logrus.Error(err)
				}
			}
		}()
	}
}

func (c *common) recordFailedImage(name string) {
	c.failedImageListMutex.Lock()
	c.failedImageSet[name] = true
	c.failedImageListMutex.Unlock()
}

func (c *common) handleError(err error) error {
	if err == nil {
		return nil
	}
	select {
	case c.errorCh <- err:
	case <-c.errorCtx.Done():
		// If context canceled, skip recording error messages.
	}
	return c.errorCtx.Err()
}

func (c *common) handleObject(obj any) error {
	if obj == nil {
		return nil
	}
	select {
	case c.objectCh <- obj:
	case <-c.objectCtx.Done():
		// If context canceled, skip sending object to worker.
	}
	return c.errorCtx.Err()
}

func (c *common) waitWorkers() {
	close(c.objectCh)
	// Waiting for all images were copied
	c.waitGroup.Wait()
	close(c.errorCh)
	// Waiting for all error messages were handled properly
	c.errorWaitGroup.Wait()
}

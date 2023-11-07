package hangar

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
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
	// errorCh is a channel to receive error message
	errorCh chan error
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
	for i := 0; i < c.workers && i < len(c.images); i++ {
		c.waitGroup.Add(1)
		go c.workerFunc(ctx, f)
	}
}

func (c *common) workerFunc(ctx context.Context, f func(context.Context, any)) {
	defer c.waitGroup.Done()
	for {
		select {
		case <-ctx.Done():
			logrus.Warnf("worker routine killed gracefully: %v", ctx.Err())
			return
		case obj, ok := <-c.objectCh:
			if !ok {
				logrus.Debugf("channel closed, release worker")
				return
			}
			if obj == nil {
				continue
			}
			f(ctx, obj)
		}
	}
}

func (c *common) initErrorHandler(ctx context.Context) {
	c.errorWaitGroup.Add(1)
	go func() {
		defer c.errorWaitGroup.Done()
		for {
			select {
			case err, ok := <-c.errorCh:
				if !ok {
					logrus.Debugf("error channel closed, release error routine")
					return
				}
				if ctx.Err() == nil {
					logrus.Errorf("%v", err)
				}
			}
		}
	}()
}

func (c *common) recordFailedImage(name string) {
	c.failedImageListMutex.Lock()
	c.failedImageSet[name] = true
	c.failedImageListMutex.Unlock()
}

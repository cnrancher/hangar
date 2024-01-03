package hangar

import (
	"context"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
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
	// failedImageListName is the file name of the failed image list
	failedImageListName string
	// systemContext
	systemContext *types.SystemContext
	// policy
	policy *signature.Policy
}

type CommonOpts struct {
	Images              []string
	Arch                []string
	OS                  []string
	Variant             []string
	Timeout             time.Duration
	Workers             int
	FailedImageListName string
	SystemContext       *types.SystemContext
	Policy              *signature.Policy
}

func newCommon(o *CommonOpts) (*common, error) {
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
		failedImageListName:  o.FailedImageListName,

		systemContext: utils.CopySystemContext(o.SystemContext),
		policy:        nil,
	}
	var err error
	policy, err := utils.CopyPolicy(o.Policy)
	if err != nil {
		return nil, fmt.Errorf("failed to copy policy: %w", err)
	}
	c.policy = policy
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

	return c, nil
}

func (c *common) SaveFailedImages() error {
	if len(c.failedImageSet) == 0 {
		return nil
	}
	file, err := os.Create(c.failedImageListName)
	if err != nil {
		return fmt.Errorf("failed to create file %q: %w",
			c.failedImageListName, err)
	}
	defer file.Close()
	for i := range c.failedImageSet {
		_, err = file.WriteString(fmt.Sprintf("%s\n", i))
		if err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}
	logrus.Infof("Failed image list exported to %q", c.failedImageListName)
	return nil
}

func (c *common) initWorker(ctx context.Context, f func(context.Context, any)) {
	c.objectCtx = ctx
	maxWorkerNum := c.workers
	if len(c.images) > 0 && len(c.images) < maxWorkerNum {
		maxWorkerNum = len(c.images)
		logrus.Debugf("Reset worker num %d", maxWorkerNum)
	}
	for i := 0; i < maxWorkerNum; i++ {
		c.waitGroup.Add(1)
		go c.workerFunc(i, f)
		logrus.Debugf("Created worker id %v", i)
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

// layerManager is for managing image layer cache.
type layerManager struct {
	mutex        *sync.RWMutex
	layersRefMap map[string]int
	cacheDir     string
}

func newLayerManager(index *archive.Index) (*layerManager, error) {
	tmpDir, err := os.MkdirTemp(utils.CacheDir(), "*")
	if err != nil {
		return nil, fmt.Errorf("mkdir temp: %w", err)
	}
	m := &layerManager{
		mutex:        &sync.RWMutex{},
		layersRefMap: make(map[string]int),
		cacheDir:     tmpDir,
	}
	for _, img := range index.List {
		for _, spec := range img.Images {
			for _, layer := range spec.Layers {
				m.layersRefMap[layer.Encoded()]++
			}
			if spec.Config != "" {
				m.layersRefMap[spec.Config.Encoded()]++
			}
			m.layersRefMap[spec.Digest.Encoded()]++
		}
	}
	return m, nil
}

func (m *layerManager) getImageLayers(img *archive.ImageSpec) []string {
	var data = make([]string, 0, len(img.Layers)+2)
	for _, l := range img.Layers {
		data = append(data, l.Encoded())
	}
	data = append(data, img.Digest.Encoded())
	if img.Config != "" {
		data = append(data, img.Config.Encoded())
	}
	return data
}

func (m *layerManager) decompressLayer(
	img *archive.ImageSpec, ar *archive.Reader,
) error {
	for _, layer := range m.getImageLayers(img) {
		p := path.Join(archive.SharedBlobDir, "sha256", layer)
		err := ar.Decompress(p, m.blobDir())
		if err != nil {
			return fmt.Errorf("failed to decompress [%v]: %w", p, err)
		}
	}
	return nil
}

// clean deletes blobs in shared directory.
func (m *layerManager) clean(img *archive.ImageSpec) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	layers := m.getImageLayers(img)
	for i := 0; i < len(layers); i++ {
		layer := layers[i]
		ref, ok := m.layersRefMap[layer]
		if !ok {
			logrus.Warnf(
				"failed to cleanup [%v]: layer not exists in ref map", layer)
			continue
		}
		if ref > 0 {
			m.layersRefMap[layer]--
		}
		if m.layersRefMap[layer] == 0 {
			m.layersRefMap[layer]--
			p := path.Join(m.blobDir(), layer)
			if _, err := os.Stat(p); err != nil {
				logrus.Warnf("failed to cleanup [%v]: stat %v", p, err)
			}
			if err := os.RemoveAll(p); err != nil {
				logrus.Warnf("failed to cleanup [%v]: %v", p, err)
			}
		}
	}
}

func (m *layerManager) cleanAll() {
	if err := os.RemoveAll(m.cacheDir); err != nil {
		logrus.Warnf("failed to cleanup [%v]: %v", m.cacheDir, err)
	}
}

func (m *layerManager) sharedBlobDir() string {
	return path.Join(m.cacheDir, archive.SharedBlobDir)
}

func (m *layerManager) blobDir() string {
	return path.Join(m.cacheDir, archive.SharedBlobDir, "sha256")
}

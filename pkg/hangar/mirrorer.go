package hangar

import (
	"context"
	"fmt"
	"strings"

	"github.com/cnrancher/hangar/pkg/destination"
	"github.com/cnrancher/hangar/pkg/hangar/imagelist"
	"github.com/cnrancher/hangar/pkg/source"
	"github.com/cnrancher/hangar/pkg/types"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
)

// Mirrorer mirrors multipule images between image registries.
type Mirrorer struct {
	*common

	// Override the registry of source image to be copied
	SourceRegistry string
	// Override the registry of the copied destination image
	DestinationRegistry string
	// Override the project of source image to be copied
	SourceProject string
	// Override the project of the copied destination image
	DestinationProject string
}

type MirrorerOpts struct {
	CommonOpts

	SourceRegistry      string
	DestinationRegistry string
	SourceProject       string
	DestinationProject  string
}

func NewMirrorer(o *MirrorerOpts) *Mirrorer {
	m := &Mirrorer{
		SourceRegistry:      o.SourceRegistry,
		DestinationRegistry: o.DestinationRegistry,
		SourceProject:       o.SourceProject,
		DestinationProject:  o.DestinationProject,
	}
	m.common = newCommon(&o.CommonOpts)
	return m
}

func (m *Mirrorer) copy(ctx context.Context) {
	m.common.initErrorHandler(ctx)
	m.common.initWorker(ctx, m.worker)
	for _, line := range m.common.images {
		switch imagelist.Detect(line) {
		case imagelist.TypeDefault:
			m.copyImageListTypeDefault(line)
		case imagelist.TypeMirror:
			m.copyImageListTypeMirror(line)
		default:
			logrus.Warnf("Ignore image list line %q: invalid format", line)
		}
	}

	close(m.common.objectCh)
	// Waiting for all images copied
	m.common.waitGroup.Wait()
	close(m.common.errorCh)
	// Waiting for all error messages were handled properly
	m.common.errorWaitGroup.Wait()
}

// Run mirror images from source to destination registry.
func (m *Mirrorer) Run(ctx context.Context) error {
	m.copy(ctx)
	if len(m.failedImageList) != 0 {
		return fmt.Errorf("some images failed to copy: \n%s",
			strings.Join(m.failedImageList, "\n"))

	}
	return nil
}

func (m *Mirrorer) copyImageListTypeDefault(line string) {
	object := &copyObject{
		image: line,
		// id:     i,
		copier: nil,
	}
	sourceRegistry := utils.GetRegistryName(line)
	if m.SourceRegistry != "" {
		sourceRegistry = m.SourceRegistry
	}
	sourceProject := utils.GetProjectName(line)
	if m.SourceProject != "" {
		sourceProject = m.SourceProject
	}
	src, err := source.NewSource(&source.Option{
		Type:     types.TypeDocker,
		Registry: sourceRegistry,
		Project:  sourceProject,
		Name:     utils.GetImageName(line),
		Tag:      utils.GetImageTag(line),
	})
	if err != nil {
		m.errorCh <- fmt.Errorf("failed to init source image: %v", err)
		m.recordFailedImage(line)
		return
	}
	object.source = src
	destProject := utils.GetProjectName(line)
	if m.DestinationProject != "" {
		destProject = m.DestinationProject
	}
	dest, err := destination.NewDestination(&destination.Option{
		Type:     types.TypeDocker,
		Registry: m.DestinationRegistry,
		Project:  destProject,
		Name:     utils.GetImageName(line),
		Tag:      utils.GetImageTag(line),
	})
	if err != nil {
		m.errorCh <- fmt.Errorf("failed to init dest image: %v", err)
		m.recordFailedImage(line)
		return
	}
	object.destination = dest
	m.common.objectCh <- object
}

func (m *Mirrorer) copyImageListTypeMirror(line string) {
	object := &copyObject{
		image: line,
		// id:     i,
		copier: nil,
	}

	spec, _ := imagelist.GetMirrorSpec(line)
	if len(spec) != 3 {
		m.errorCh <- fmt.Errorf("ignore line %q in image list: invalid format", line)
		m.recordFailedImage(line)
		return
	}
	sourceRegistry := utils.GetRegistryName(spec[0])
	if m.SourceRegistry != "" {
		sourceRegistry = m.SourceRegistry
	}
	sourceProject := utils.GetProjectName(spec[0])
	if m.SourceProject != "" {
		sourceProject = m.SourceProject
	}
	src, err := source.NewSource(&source.Option{
		Type:     types.TypeDocker,
		Registry: sourceRegistry,
		Project:  sourceProject,
		Name:     utils.GetImageName(spec[0]),
		Tag:      spec[2],
	})
	if err != nil {
		m.errorCh <- fmt.Errorf("failed to init source image: %v", err)
		m.recordFailedImage(line)
		return
	}
	object.source = src
	destProject := utils.GetProjectName(spec[1])
	if m.DestinationProject != "" {
		destProject = m.DestinationProject
	}
	dest, err := destination.NewDestination(&destination.Option{
		Type:     types.TypeDocker,
		Registry: m.DestinationRegistry,
		Project:  destProject,
		Name:     utils.GetImageName(spec[1]),
		Tag:      spec[2],
	})
	if err != nil {
		m.errorCh <- fmt.Errorf("failed to init dest image: %v", err)
		m.recordFailedImage(line)
		return
	}
	object.destination = dest
	m.common.objectCh <- object
}

func (m *Mirrorer) worker(ctx context.Context, id int) {
	defer m.common.waitGroup.Done()
	for {
		select {
		case <-ctx.Done():
			logrus.Debugf("worker stopped: %v", ctx.Err())
			return
		case obj, ok := <-m.common.objectCh:
			if !ok {
				logrus.Debugf("channel closed, release worker")
				return
			}
			if obj == nil {
				continue
			}

			var (
				copyContext context.Context
				cancel      context.CancelFunc
				err         error
			)
			if obj.timeout > 0 {
				copyContext, cancel = context.WithTimeout(ctx, obj.timeout)
			} else {
				copyContext, cancel = context.WithCancel(ctx)
			}

			err = obj.source.Init(copyContext)
			if err != nil {
				m.common.errorCh <- err
				m.common.recordFailedImage(obj.source.ReferenceNameWithoutTransport())
				cancel()
				continue
			}
			err = obj.destination.Init(copyContext)
			if err != nil {
				m.common.errorCh <- err
				m.common.recordFailedImage(obj.source.ReferenceNameWithoutTransport())
				cancel()
				continue
			}
			err = obj.source.Copy(copyContext, obj.destination, m.common.imageSpecSet)
			if err != nil {
				m.common.errorCh <- err
				m.common.recordFailedImage(obj.source.ReferenceNameWithoutTransport())
				cancel()
				continue
			}

			logrus.Infof("[%v] => [%v]",
				obj.source.ReferenceNameWithoutTransport(),
				obj.destination.ReferenceNameWithoutTransport())
			// Cancel the context after copy image
			cancel()
		}
	}
}

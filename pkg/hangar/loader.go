package hangar

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/cnrancher/hangar/pkg/destination"
	"github.com/cnrancher/hangar/pkg/hangar/archive"
	"github.com/cnrancher/hangar/pkg/manifest"
	"github.com/cnrancher/hangar/pkg/source"
	"github.com/cnrancher/hangar/pkg/types"
	"github.com/cnrancher/hangar/pkg/utils"
	imagetypes "github.com/containers/image/v5/types"
	"github.com/sirupsen/logrus"
)

// loadObject is the object sending to worker pool when loading image
type loadObject struct {
	image   *archive.Image
	timeout time.Duration
	id      int
}

// layerManager is for managing image layer cache.
type layerManager struct {
	mutex        *sync.RWMutex
	layersRefMap map[string]int
	cacheDir     string
}

func newLayerManager(index *archive.Index) (*layerManager, error) {
	tmpDir, err := os.MkdirTemp(archive.CacheDir(), "*")
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
	var data []string = make([]string, 0, len(img.Layers)+2)
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

func (m *layerManager) clean(img *archive.ImageSpec) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	layers := m.getImageLayers(img)
	for i := 0; i < len(layers); i++ {
		if m.layersRefMap[layers[i]] > 0 {
			m.layersRefMap[layers[i]]--
		}
		if m.layersRefMap[layers[i]] <= 0 {
			m.layersRefMap[layers[i]]--
			p := path.Join(m.blobDir(), layers[i])
			err := os.RemoveAll(p)
			if err != nil {
				logrus.Warnf("failed to cleanup [%v]: %v", p, err)
			}
		}
	}
}

func (m *layerManager) sharedBlobDir() string {
	return path.Join(m.cacheDir, archive.SharedBlobDir)
}

func (m *layerManager) blobDir() string {
	return path.Join(m.cacheDir, archive.SharedBlobDir, "sha256")
}

// Loader loads images from hangar archive file to registry server.
type Loader struct {
	*common

	// ar is the archive reader.
	ar *archive.Reader
	// arMutex is the mutex for archive reader.
	arMutex *sync.RWMutex
	// index is the archive index.
	index *archive.Index
	// layerManager manages the layers
	layerManager *layerManager

	// Specify the destination image registry.
	DestinationRegistry string
	// Specify the destination image project.
	DestinationProject string
	// Directory is the source archive directory
	Directory string
	// SharedBlobDirPath is the directory to save the shared blobs
	SharedBlobDirPath string
	// ArchiveName is the archive file name to be load
	ArchiveName string
}

type LoaderOpts struct {
	CommonOpts

	// Specify the destination image registry.
	DestinationRegistry string
	// Specify the destination image project.
	DestinationProject string
	// Directory is the source archive directory
	Directory string
	// SharedBlobDirPath is the directory to save the shared blobs
	SharedBlobDirPath string
	// ArchiveName is the archive file name to be load
	ArchiveName string
	// Use HTTPS and verify certificate
	TlsVerify bool
}

func NewLoader(o *LoaderOpts) (*Loader, error) {
	l := &Loader{
		ar:           nil,
		arMutex:      &sync.RWMutex{},
		index:        archive.NewIndex(),
		layerManager: nil,

		DestinationRegistry: o.DestinationRegistry,
		DestinationProject:  o.DestinationProject,
		Directory:           o.Directory,
		SharedBlobDirPath:   o.SharedBlobDirPath,
		ArchiveName:         o.ArchiveName,
	}
	if l.SharedBlobDirPath == "" {
		l.SharedBlobDirPath = archive.SharedBlobDir
	}
	l.common = newCommon(&o.CommonOpts)

	l.arMutex.Lock()
	defer l.arMutex.Unlock()
	var err error
	l.ar, err = archive.NewReader(l.ArchiveName)
	if err != nil {
		return nil, fmt.Errorf("failed to create archive reader: %w", err)
	}
	b, err := l.ar.Index()
	if err != nil {
		return nil, fmt.Errorf("ar.Index: %w", err)
	}
	err = json.Unmarshal(b, l.index)
	if err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	if len(l.index.List) == 0 {
		logrus.Warnf("No images in %q", o.ArchiveName)
	}
	lm, err := newLayerManager(l.index)
	if err != nil {
		return nil, fmt.Errorf("failed to init layer manager: %w", err)
	}
	l.layerManager = lm

	return l, nil
}

func (l *Loader) copy(ctx context.Context) {
	l.common.initErrorHandler(ctx)
	l.common.images = make([]string, len(l.index.List))
	l.common.initWorker(ctx, l.worker)
	for i, image := range l.index.List {
		object := &loadObject{
			id:    i + 1,
			image: image,
		}
		l.handleObject(object)
	}
	l.waitWorkers()
	if err := l.ar.Close(); err != nil {
		logrus.Errorf("failed to close archive reader: %v", err)
	}
}

func (l *Loader) newLoadCacheDir() (string, error) {
	cd, err := os.MkdirTemp(archive.CacheDir(), "*")
	if err != nil {
		return "", fmt.Errorf("os.MkdirTemp: %w", err)
	}
	return cd, nil
}

// Run loads images from hangar archive to destination image registry
func (l *Loader) Run(ctx context.Context) error {
	l.copy(ctx)
	if len(l.failedImageSet) != 0 {
		for i := range l.failedImageSet {
			fmt.Printf("%v\n", i)
		}
		return fmt.Errorf("some images failed to load")
	}
	return nil
}

func (l *Loader) worker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*loadObject)
	if !ok {
		logrus.Errorf("skip object type(%T), data %v", o, o)
		return
	}

	imageName := obj.image.Source + ":" + obj.image.Tag
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
	defer cancel()
	// Use defer to handle error message.
	defer func() {
		if err != nil {
			l.handleError(err)
			l.recordFailedImage(imageName)
		}
	}()

	// Init destination image spec.
	destinationRegistry := utils.GetRegistryName(imageName)
	if l.DestinationRegistry != "" {
		destinationRegistry = l.DestinationRegistry
	}
	destinationProject := utils.GetProjectName(imageName)
	if l.DestinationProject != "" {
		destinationProject = l.DestinationProject
	}
	dest, err := destination.NewDestination(&destination.Option{
		Type:     types.TypeDocker,
		Registry: destinationRegistry,
		Project:  destinationProject,
		Name:     utils.GetImageName(imageName),
		Tag:      obj.image.Tag,
		SystemContext: &imagetypes.SystemContext{
			DockerInsecureSkipTLSVerify: imagetypes.NewOptionalBool(l.common.tlsVerify),
			OCIInsecureSkipTLSVerify:    l.common.tlsVerify,
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to init destination image: %w", err)
		return
	}
	if err = dest.Init(copyContext); err != nil {
		return
	}
	// Init manifest Builder.
	var builder *manifest.Builder
	builder, err = manifest.NewBuilder(&manifest.BuilderOpts{
		ReferenceName: dest.ReferenceName(),
		SystemContext: &imagetypes.SystemContext{
			DockerInsecureSkipTLSVerify: imagetypes.NewOptionalBool(l.common.tlsVerify),
			OCIInsecureSkipTLSVerify:    l.common.tlsVerify,
		},
	})
	if err != nil {
		err = fmt.Errorf("failed to create manifest builder: %w", err)
		return
	}
	for _, img := range obj.image.Images {
		if img.Digest == "" {
			logrus.WithFields(logrus.Fields{"IMG": obj.id}).
				Warnf("Skip invalid image [%v] [%v] [%v]",
					imageName, img.Arch, img.OS)
			continue
		}

		imgRef := "docker://" + obj.image.Source + "@" + img.Digest.String()
		var tmpDir string
		tmpDir, err = l.ar.DecompressImageTmp(
			&img, l.common.imageSpecSet, l.layerManager.blobDir())

		// Register defer function to clean-up cache dir.
		defer func(d string, img archive.ImageSpec) {
			if d != "" {
				os.RemoveAll(d)
			}
			l.layerManager.clean(&img)
		}(tmpDir, img)

		if err != nil {
			err = fmt.Errorf(
				"failed to decompress image [%v]: %w", imgRef, err)
			return
		}
		if err = l.layerManager.decompressLayer(&img, l.ar); err != nil {
			return
		}
		var src *source.Source
		src, err = source.NewSource(&source.Option{
			Type:      types.TypeOci,
			Directory: tmpDir,
			SystemContext: &imagetypes.SystemContext{
				OCISharedBlobDirPath: l.layerManager.sharedBlobDir(),
			},
		})
		if err != nil {
			err = fmt.Errorf("failed to init source image: %w", err)
			continue
		}
		if err = src.Init(copyContext); err != nil {
			return
		}
		if err = src.Copy(copyContext, dest, l.common.imageSpecSet); err != nil {
			return
		}

		var mi *manifest.ManifestImage
		mi, err = manifest.NewManifestImage(
			ctx, imgRef, &imagetypes.SystemContext{})
		if err != nil {
			err = fmt.Errorf("failed to create manifest image: %w", err)
			return
		}
		builder.Add(mi)
	}

	if builder.Images() == 0 {
		err = fmt.Errorf("failed to load [%v]: some images failed to load", imageName)
		return
	}
	if err = builder.Push(ctx); err != nil {
		err = fmt.Errorf("failed to push manifest: %w", err)
		return
	}

	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Infof("Loaded [%v]", imageName)
}

package hangar

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/cnrancher/hangar/pkg/hangar/imagelist"
	"github.com/cnrancher/hangar/pkg/image/scan"
	"github.com/cnrancher/hangar/pkg/image/source"
	"github.com/cnrancher/hangar/pkg/image/types"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
)

type scanObject struct {
	image   string
	source  *source.Source
	timeout time.Duration
	id      int
}

type Scanner struct {
	*common

	reportMu *sync.Mutex
	report   *scan.Report
	// Override the registry
	Registry string
	// Override the project
	Project string
}

// Scanner implements functions of Hangar interface
var _ Hangar = &Scanner{}

type ScannerOpts struct {
	CommonOpts

	Report   *scan.Report
	Registry string
	Project  string
}

func NewScanner(o *ScannerOpts) (*Scanner, error) {
	s := &Scanner{
		reportMu: &sync.Mutex{},
		report:   o.Report,
		Registry: o.Registry,
		Project:  o.Project,
	}
	if s.report == nil {
		s.report = scan.NewReport()
	}
	var err error
	s.common, err = newCommon(&o.CommonOpts)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Scanner) scan(ctx context.Context) {
	s.common.initErrorHandler(ctx)
	s.common.initWorker(ctx, s.worker)
	for i, line := range s.common.images {
		var (
			object *scanObject
			err    error
		)
		switch imagelist.Detect(line) {
		case imagelist.TypeDefault:
		default:
			logrus.Warnf("Ignore image list line %q: invalid format", line)
			continue
		}
		object = &scanObject{
			id:      i + 1,
			image:   line,
			timeout: s.timeout,
		}
		registry := utils.GetRegistryName(line)
		if s.Registry != "" {
			registry = s.Registry
		}
		project := utils.GetProjectName(line)
		if s.Project != "" {
			project = s.Project
		}
		src, err := source.NewSource(&source.Option{
			Type:          types.TypeDocker,
			Registry:      registry,
			Project:       project,
			Name:          utils.GetImageName(line),
			Tag:           utils.GetImageTag(line),
			SystemContext: s.systemContext,
		})
		if err != nil {
			s.handleError(fmt.Errorf("failed to init source image: %w", err))
			s.recordFailedImage(line)
			continue
		}
		object.source = src
		if err = s.handleObject(object); err != nil {
			s.handleError(err)
			s.recordFailedImage(line)
		}
	}
	s.waitWorkers()
}

func (s *Scanner) worker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*scanObject)
	if !ok {
		logrus.Errorf("skip object type(%T), data %v", o, o)
		return
	}
	var (
		scanContext context.Context
		cancel      context.CancelFunc
		err         error
	)
	if obj.timeout > 0 {
		scanContext, cancel = context.WithTimeout(ctx, obj.timeout)
	} else {
		scanContext, cancel = context.WithCancel(ctx)
	}
	defer func() {
		cancel()
		if err != nil {
			s.handleError(fmt.Errorf("error occurred when scan [%v]: %w",
				obj.source.ReferenceNameWithoutTransport(), err))
			s.common.recordFailedImage(obj.image)
		}
	}()

	err = obj.source.Init(scanContext)
	if err != nil {
		err = fmt.Errorf("failed to init [%v]: %w",
			obj.source.ReferenceName(), err)
		return
	}
	logrus.WithFields(logrus.Fields{"IMG": obj.id}).
		Infof("Scanning [%v]", obj.source.ReferenceNameWithoutTransport())
	var result *scan.Result
	result, err = obj.source.Scan(scanContext, &source.ScanOptions{
		Set: s.common.imageSpecSet,
	})
	if err != nil {
		if errors.Is(err, utils.ErrNoAvailableImage) {
			logrus.WithFields(logrus.Fields{"IMG": obj.id}).
				Warnf("Skip scan image [%v]: %v",
					obj.source.ReferenceNameWithoutTransport(), err)
			err = nil
			return
		}
		err = fmt.Errorf("failed to scan [%v]: %w",
			obj.source.ReferenceName(), err)
		return
	}
	if !result.Pass() {
		logrus.WithFields(logrus.Fields{"IMG": obj.id}).
			Warnf("Vulnerabilities detected on image [%v]", obj.source.ReferenceNameWithoutTransport())
	}
	s.reportMu.Lock()
	s.report.Append(result)
	s.reportMu.Unlock()
}

func (s *Scanner) Run(ctx context.Context) error {
	s.scan(ctx)
	if len(s.failedImageSet) != 0 {
		v := make([]string, 0, len(s.failedImageSet))
		for i := range s.failedImageSet {
			v = append(v, i)
		}
		logrus.Errorf("Scan failed image list: \n%v", strings.Join(v, "\n"))
		return ErrScanFailed
	}
	return nil
}

func (s *Scanner) Validate(ctx context.Context) error {
	panic("Scanner does not support validation mode yet")
}

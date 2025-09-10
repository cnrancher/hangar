package hangar

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/cnrancher/hangar/pkg/hangar/imagelist"
	"github.com/cnrancher/hangar/pkg/image/source"
	"github.com/cnrancher/hangar/pkg/image/types"
	"github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
)

// inspectObject is the object for sending to worker pool when inspecting image
type inspectObject struct {
	image   string
	source  *source.Source
	timeout time.Duration
	id      int
}

type Inspector struct {
	*common
	report []inspectorReport

	reportFile string
	format     string
	autoYes    bool

	// Override the registry of source image to be copied
	Registry string
	// Override the project of source image to be copied
	Project string
}

type InspectorOpts struct {
	CommonOpts

	ReportFile   string
	ReportFormat string
	AutoYes      bool

	// Override the registry of source image to be copied
	Registry string
	// Override the project of source image to be copied
	Project string
}

type inspectorReport struct {
	Image     string   `json:"image" yaml:"image"`
	Platforms []string `json:"platforms" yaml:"platforms"`
}

func NewInspector(o *InspectorOpts) (*Inspector, error) {
	s := &Inspector{
		reportFile: o.ReportFile,
		format:     o.ReportFormat,
		autoYes:    o.AutoYes,

		Registry: o.Registry,
		Project:  o.Project,
	}
	var err error
	s.common, err = newCommon(&o.CommonOpts)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Inspector) inspect(ctx context.Context) {
	s.common.initErrorHandler(ctx)
	s.common.initWorker(ctx, s.worker)
	for i, img := range s.common.images {
		switch imagelist.Detect(img) {
		case imagelist.TypeDefault:
		default:
			logrus.Warnf("Ignore image list line %q: invalid format", img)
			continue
		}
		object := &inspectObject{
			id:      i + 1,
			image:   img,
			timeout: s.timeout,
		}
		registry := utils.GetRegistryName(img)
		if s.Registry != "" {
			registry = s.Registry
		}
		project := utils.GetProjectName(img)
		if s.Project != "" {
			project = s.Project
		}
		src, err := source.NewSource(&source.Option{
			Type:          types.TypeDocker,
			Registry:      registry,
			Project:       project,
			Name:          utils.GetImageName(img),
			Tag:           utils.GetImageTag(img),
			SystemContext: s.systemContext,
		})
		object.source = src
		if err != nil {
			s.handleError(fmt.Errorf("failed to init image: %w", err))
			s.recordFailedImage(img)
			continue
		}
		if err = s.handleObject(object); err != nil {
			s.handleError(fmt.Errorf("failed to handle object: %w", err))
			s.recordFailedImage(img)
		}
	}
	s.waitWorkers()
}

// Run save images from registry server into local directory / hangar archive.
func (s *Inspector) Run(ctx context.Context) error {
	s.inspect(ctx)
	var errs = []error{}
	if len(s.failedImageSet) != 0 {
		v := make([]string, 0, len(s.failedImageSet))
		for i := range s.failedImageSet {
			v = append(v, i)
		}
		logrus.Errorf("Inspect failed image list: \n%v", strings.Join(v, "\n"))
		errs = append(errs, ErrInspectFailed)
	}
	if err := s.saveReport(ctx); err != nil {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

func (s *Inspector) Validate(_ context.Context) error {
	return nil
}

func (s *Inspector) worker(ctx context.Context, o any) {
	if o == nil {
		return
	}
	obj, ok := o.(*inspectObject)
	if !ok {
		logrus.Errorf("skip object type(%T), data %v", o, o)
		return
	}

	var (
		inspectContext context.Context
		cancel         context.CancelFunc
		err            error
	)
	if obj.timeout > 0 {
		inspectContext, cancel = context.WithTimeout(ctx, obj.timeout)
	} else {
		inspectContext, cancel = context.WithCancel(ctx)
	}
	defer func() {
		if err != nil {
			s.handleError(NewError(obj.id, err, nil, nil))
			s.recordFailedImage(obj.image)
		}
		cancel()
	}()

	err = obj.source.Init(inspectContext)
	if err != nil {
		err = fmt.Errorf("failed to init image %v: %w",
			obj.image, err)
		return
	}
	img := obj.source.ReferenceNameWithoutTransport()
	platforms := obj.source.Platforms(true)
	s.report = append(s.report, inspectorReport{
		Image:     img,
		Platforms: platforms,
	})

	if len(platforms) == 0 {
		logrus.WithFields(logrus.Fields{"IMG": obj.id}).Warnf("Skip [%v]: no platforms found",
			img)
		return
	}
	message := fmt.Sprintf("Image [%v]: %v",
		img, strings.Join(platforms, ","))
	logrus.WithFields(logrus.Fields{"IMG": obj.id}).Info(message)
}

func (s *Inspector) saveReport(ctx context.Context) error {
	var report string
	suffix := "txt"
	switch s.format {
	case "json":
		report = utils.ToJSON(s.report)
		suffix = s.format
	case "yaml":
		report = utils.ToYAML(s.report)
		suffix = s.format
	default:
		report = index2Report(s.report)
	}

	if s.reportFile == "" {
		s.reportFile = fmt.Sprintf("inspect-report.%v", suffix)
	}
	if err := utils.CheckFileExistsPrompt(ctx, s.reportFile, s.autoYes); err != nil {
		return err
	}
	f, err := os.OpenFile(s.reportFile, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to create report %q: %w", s.reportFile, err)
	}
	defer f.Close()
	if _, err := f.WriteString(report + "\n"); err != nil {
		return fmt.Errorf("failed to write report %q: %w", s.reportFile, err)
	}

	logrus.Infof("Report saved to %q", s.reportFile)
	return nil
}

func index2Report(report []inspectorReport) string {
	b := strings.Builder{}
	for i, report := range report {
		p := strings.Join(report.Platforms, ",")
		if p == "" {
			p = "unknown"
		}
		b.WriteString(fmt.Sprintf("%4d | %s | %s\n",
			i+1, report.Image, p))
	}
	return b.String()
}

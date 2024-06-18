package scan

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"slices"
	"time"

	"github.com/aquasecurity/trivy/pkg/sbom/spdx"
	"github.com/aquasecurity/trivy/pkg/types"
	"github.com/opencontainers/go-digest"
	"github.com/spdx/tools-golang/spdx/v2/common"
	gospdx "github.com/spdx/tools-golang/spdx/v2/v2_3"
)

var (
	severityName = []string{
		"UNKNOWN",
		"LOW",
		"MEDIUM",
		"HIGH",
		"CRITICAL",
	}
)

type Severity int

const (
	SeverityUnknown Severity = iota
	SeverityLow
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

func NewSeverity(s string) Severity {
	if i := slices.Index(severityName, s); i >= 0 {
		return Severity(i)
	}
	return SeverityUnknown
}

type Report struct {
	Time    time.Time `json:"time,omitempty" yaml:"time,omitempty"`
	Results []*Result `json:"results,omitempty" yaml:"result,omitempty"`
}

type Result struct {
	Reference string         `json:"reference,omitempty" yaml:"reference,omitempty"`
	Images    []*ImageResult `json:"images,omitempty" yaml:"images,omitempty"`
}

func NewResult(ref string, images []*ImageResult) *Result {
	return &Result{
		Reference: ref,
		Images:    images,
	}
}

// Pass detects if the image results have vulnerabilities
func (r *Result) Pass() bool {
	if len(r.Images) == 0 {
		return true
	}
	for _, image := range r.Images {
		if len(image.Vulnerabilities) > 0 {
			return false
		}
	}
	return true
}

type ImageResult struct {
	Digest          digest.Digest    `json:"digest,omitempty" yaml:"digest,omitempty"`
	Platform        Platform         `json:"platform,omitempty" yaml:"platform,omitempty"`
	SBOM_SPDX       *gospdx.Document `json:"spdx,omitempty" yaml:"spdx,omitempty"`
	Vulnerabilities []Vulnerability  `json:"vulnerabilities,omitempty" yaml:"vulnerabilities,omitempty"`
}

type Platform struct {
	Arch       string   `json:"arch,omitempty" yaml:"arch,omitempty"`
	OS         string   `json:"os,omitempty" yaml:"os,omitempty"`
	OSVersion  string   `json:"osVersion,omitempty" yaml:"osVersion,omitempty"`
	OSFeatures []string `json:"osFeatures,omitempty" yaml:"osFeatures,omitempty"`
	Variant    string   `json:"variant,omitempty" yaml:"variant,omitempty"`
}

type Vulnerability struct {
	Title            string   `json:"title" yaml:"title"`
	ID               string   `json:"id" yaml:"id"`
	Severity         Severity `json:"-" yaml:"-"`
	SeverityString   string   `json:"severity" yaml:"severity"`
	PkgName          string   `json:"package" yaml:"package"`
	InstalledVersion string   `json:"installed" yaml:"installed"`
	FixedVersion     string   `json:"fixed" yaml:"fixed"`
	PrimaryURL       string   `json:"url" yaml:"url"`
}

func NewReport() *Report {
	return &Report{
		Time:    time.Now(),
		Results: make([]*Result, 0),
	}
}

func (r *Report) Append(result *Result) {
	if result == nil {
		return
	}
	r.Results = append(r.Results, result)
}

func (r *Report) WriteCSV(f io.Writer) error {
	line := []string{
		"image",     // 0
		"arch",      // 1
		"os",        // 2
		"package",   // 3
		"title",     // 4
		"id",        // 5
		"severity",  // 6
		"installed", // 7
		"fixed",     // 8
		"url",       // 9
	}
	cw := csv.NewWriter(f)
	defer cw.Flush()
	if err := cw.Write(line); err != nil {
		return err
	}
	if len(r.Results) == 0 {
		return nil
	}
	for _, result := range r.Results {
		if len(result.Images) == 0 {
			continue
		}
		reference := result.Reference
		for _, image := range result.Images {
			if len(image.Vulnerabilities) == 0 {
				continue
			}
			for _, v := range image.Vulnerabilities {
				line = []string{
					reference,           // 0
					image.Platform.Arch, // 1
					image.Platform.OS,   // 2
					v.PkgName,           // 3
					v.Title,             // 4
					v.ID,                // 5
					v.SeverityString,    // 6
					v.InstalledVersion,  // 7
					v.FixedVersion,      // 8
					v.PrimaryURL,        // 9
				}
				if err := cw.Write(line); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *Report) WriteSPDX_CSV(f io.Writer) error {
	line := []string{
		"image",       // 0
		"arch",        // 1
		"os",          // 2
		"package",     // 3
		"license",     // 4
		"versionInfo", // 5
		"supplier",    // 6
		"originator",  // 7
		"SPDXID",      // 8
	}
	cw := csv.NewWriter(f)
	defer cw.Flush()
	if err := cw.Write(line); err != nil {
		return err
	}
	if len(r.Results) == 0 {
		return nil
	}
	for _, result := range r.Results {
		if len(result.Images) == 0 {
			continue
		}
		reference := result.Reference
		for _, image := range result.Images {
			s := image.SBOM_SPDX
			if len(s.Packages) == 0 {
				continue
			}
			for _, p := range s.Packages {
				if p.PackageSupplier == nil {
					p.PackageSupplier = &common.Supplier{}
				}
				if p.PackageOriginator == nil {
					p.PackageOriginator = &common.Originator{}
				}
				supplier, _ := p.PackageSupplier.MarshalJSON()
				originator, _ := p.PackageOriginator.MarshalJSON()
				line = []string{
					reference,                       // 0
					image.Platform.Arch,             // 1
					image.Platform.OS,               // 2
					p.PackageName,                   // 3
					p.PackageLicenseDeclared,        // 4
					p.PackageVersion,                // 5
					string(supplier),                // 6
					string(originator),              // 7
					string(p.PackageSPDXIdentifier), // 8
				}
				if err := cw.Write(line); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (r *Result) Append(image *ImageResult) {
	if image == nil {
		return
	}
	r.Images = append(r.Images, image)
}

func NewImageResult(
	ctx context.Context, report *types.Report, format string, opt *ScanOption,
) (*ImageResult, error) {
	image := &ImageResult{
		Digest:          opt.Digest,
		Platform:        opt.Platform,
		SBOM_SPDX:       nil,
		Vulnerabilities: nil,
	}
	switch format {
	case "spdx-json", "spdx-csv":
		// Only generate SPDX for SBOM outputs.
		var err error
		m := spdx.NewMarshaler("")
		image.SBOM_SPDX, err = m.MarshalReport(ctx, *report)
		if err != nil {
			return nil, fmt.Errorf("spdx Marshal: %w", err)
		}
		// Modify the creator tool info to trivy without version info.
		for _, c := range image.SBOM_SPDX.CreationInfo.Creators {
			if c.CreatorType == "Tool" {
				c.Creator = "trivy"
			}
		}
		return image, nil
	}

	if len(report.Results) == 0 {
		return image, nil
	}

	for _, result := range report.Results {
		if len(result.Vulnerabilities) == 0 {
			continue
		}
		for _, v := range result.Vulnerabilities {
			v1 := Vulnerability{
				Title:            v.Title,
				ID:               v.VulnerabilityID,
				Severity:         NewSeverity(v.Severity),
				SeverityString:   v.Severity,
				PkgName:          v.PkgName,
				InstalledVersion: v.InstalledVersion,
				FixedVersion:     v.FixedVersion,
				PrimaryURL:       v.PrimaryURL,
			}
			image.Vulnerabilities = append(image.Vulnerabilities, v1)
		}
	}
	return image, nil
}

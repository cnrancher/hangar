package signv2

import (
	"encoding/csv"
	"fmt"
	"io"
	"time"

	"github.com/opencontainers/go-digest"
)

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

// Pass detects if the image signature verifation passed
func (r *Result) Pass() bool {
	if len(r.Images) == 0 {
		return true
	}
	for _, image := range r.Images {
		if image.Payload == "" {
			return false
		}
	}
	return true
}

type ImageResult struct {
	Digest    digest.Digest `json:"digest" yaml:"digest"`
	MediaType string        `json:"mediaType" yaml:"mediaType"`
	Platform  Platform      `json:"platform" yaml:"platform"`

	TLogVerified             bool   `json:"tlogVerified" yaml:"tlogVerified"`
	CertificateSubject       string `json:"certificateSubject,omitempty" yaml:"certificateSubject,omitempty"`
	CertificateIssuer        string `json:"certificateIssuer,omitempty" yaml:"certificateIssuer,omitempty"`
	GithubWorkflowTrigger    string `json:"githubWorkflowTrigger,omitempty" yaml:"githubWorkflowTrigger,omitempty"`
	GithubWorkflowSha        string `json:"githubWorkflowSha,omitempty" yaml:"githubWorkflowSha,omitempty"`
	GithubWorkflowName       string `json:"githubWorkflowName,omitempty" yaml:"githubWorkflowName,omitempty"`
	GithubWorkflowRepository string `json:"githubWorkflowRepository,omitempty" yaml:"githubWorkflowRepository,omitempty"`
	GithubWorkflowRef        string `json:"githubWorkflowRef,omitempty" yaml:"githubWorkflowRef,omitempty"`
	Payload                  string `json:"payload" yaml:"payload"`
}

type Platform struct {
	Arch       string   `json:"arch,omitempty" yaml:"arch,omitempty"`
	OS         string   `json:"os,omitempty" yaml:"os,omitempty"`
	OSVersion  string   `json:"osVersion,omitempty" yaml:"osVersion,omitempty"`
	OSFeatures []string `json:"osFeatures,omitempty" yaml:"osFeatures,omitempty"`
	Variant    string   `json:"variant,omitempty" yaml:"variant,omitempty"`
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
		"image",                    // 0
		"arch",                     // 1
		"os",                       // 2
		"payload",                  // 3
		"digest",                   // 4
		"tlogVerify",               // 5
		"mediaType",                // 6
		"certificateIssuer",        // 7
		"certificateSubject",       // 8
		"githubWorkflowName",       // 9
		"githubWorkflowRef",        // 10
		"githubWorkflowRepository", // 11
		"githubWorkflowSha",        // 12
		"githubWorkflowTrigger",    // 13
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
			line = []string{
				reference,                             // 0
				image.Platform.Arch,                   // 1
				image.Platform.OS,                     // 2
				image.Payload,                         // 3
				image.Digest.String(),                 // 4
				fmt.Sprintf("%v", image.TLogVerified), // 5
				image.MediaType,                       // 6
				image.CertificateIssuer,               // 7
				image.CertificateSubject,              // 8
				image.GithubWorkflowName,              // 9
				image.GithubWorkflowRef,               // 10
				image.GithubWorkflowRepository,        // 11
				image.GithubWorkflowSha,               // 12
				image.GithubWorkflowTrigger,           // 13
			}
			if err := cw.Write(line); err != nil {
				return err
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

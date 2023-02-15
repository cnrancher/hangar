package mirror

import (
	"fmt"

	u "github.com/cnrancher/hangar/pkg/utils"
	"github.com/sirupsen/logrus"
)

func (m *Mirror) StartSave() error {
	if m == nil {
		return fmt.Errorf("StartSave: %w", u.ErrNilPointer)
	}
	if m.Mode != MODE_SAVE {
		return fmt.Errorf("StartSave: mirror is not in SAVE mode")
	}
	logrus.WithField("M_ID", m.MID).
		Infof("SOURCE: [%v] TAG: [%v]", m.Source, m.Tag)
	logrus.WithField("M_ID", m.MID).Debug("Start Save")

	var err error
	// Get Absolute path of saved directory & ensure dir exists
	if m.Directory, err = u.GetAbsPath(m.Directory); err != nil {
		return fmt.Errorf("StartSave: %w", err)
	}
	if err = u.EnsureDirExists(m.Directory); err != nil {
		return fmt.Errorf("StartSave: %w", err)
	}
	// Init image list from source
	if err = m.initImageList(); err != nil {
		return fmt.Errorf("StartSave: %w", err)
	}

	// Save images into local dir
	for _, img := range m.images {
		if err := img.Save(); err != nil {
			logrus.WithFields(logrus.Fields{"M_ID": m.MID}).Error(err)
		}
	}
	if m.Saved() != m.ImageNum() {
		img := []string{}
		for i := range m.images {
			img = append(img, m.images[i].Source)
		}
		return fmt.Errorf("some images failed to save: %v", img)
	}
	logrus.WithField("M_ID", m.MID).Infof("SAVED [%s:%s]", m.Source, m.Tag)

	return nil
}

func (m *Mirror) Saved() int {
	var num int = 0
	for _, img := range m.images {
		if img.Saved {
			num++
		}
	}
	return num
}

package mirror

import (
	"fmt"

	u "cnrancher.io/image-tools/utils"
	"github.com/sirupsen/logrus"
)

func (m *Mirror) StartSave() error {
	if m == nil {
		return fmt.Errorf("StartSave: %w", u.ErrNilPointer)
	}
	if m.mode != MODE_SAVE {
		return fmt.Errorf("StartSave: mirrorer is not in SAVE mode")
	}
	logrus.WithField("M_ID", m.mID).Debug("Start Save")

	absDir, err := u.GetAbsPath(m.directory)
	if err != nil {
		return fmt.Errorf("StartSave: %w", err)
	}
	m.directory = absDir

	if err := u.EnsureDirExists(m.directory); err != nil {
		return fmt.Errorf("StartSave: %w", err)
	}

	if err := m.initImageList(); err != nil {
		return fmt.Errorf("StartSave: %w", err)
	}

	for _, img := range m.images {
		if err := img.Save(); err != nil {
			logrus.WithFields(logrus.Fields{"M_ID": m.mID}).Error(err.Error())
		}
	}

	return nil
}

func (m *Mirror) Saved() int {
	var num int = 0
	for _, img := range m.images {
		if img.Saved() {
			num++
		}
	}
	return num
}

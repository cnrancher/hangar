package mirror

import (
	"fmt"

	"github.com/sirupsen/logrus"
)

func (m *Mirror) StartLoad() error {
	if m.directory == "" {
		return fmt.Errorf("StartLoad: directory is empty string")
	}
	if m.mode != MODE_LOAD {
		return fmt.Errorf("StartSave: mirrorer is not in LOAD mode")
	}
	for _, img := range m.images {
		if err := img.Load(); err != nil {
			return fmt.Errorf("StartLoad: %w", err)
		}
	}

	logrus.WithField("M_ID", m.mID).
		Info("Creating dest manifest list...")
	if err := m.updateDestManifest(); err != nil {
		return fmt.Errorf("StartLoad: %w", err)
	}

	logrus.WithField("M_ID", m.mID).
		Infof("Successfully loaded %s:%s.",
			m.destination, m.tag)

	return nil
}

func (m *Mirror) Loaded() int {
	var num int = 0
	for _, img := range m.images {
		if img.Loaded() {
			num++
		}
	}
	return num
}

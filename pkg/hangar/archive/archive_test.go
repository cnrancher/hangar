package archive

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_CompareIndexVersion(t *testing.T) {
	index := NewIndex()

	index.Version = "v0.0.1"
	err := CompareIndexVersion(index)
	assert.NotNil(t, err)
	t.Logf("Error message: %v", err)

	index.Version = IndexVersion
	err = CompareIndexVersion(index)
	assert.Nil(t, err)

	index.Version = "v99.99.99"
	err = CompareIndexVersion(index)
	assert.Nil(t, err)
}

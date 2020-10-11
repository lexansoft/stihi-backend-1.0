package cache_level2

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestIsStihiContent(t *testing.T) {
	jsonMeta := "{\"app\":\"stihi-io\",\"editor\":\"html\",\"image\":\"\",\"tags\":[\"stihi-io\",\"ru--religioznaya-lirika\",\"test\"]}"

	assert.True(t, IsStihiContent(jsonMeta), "Detect right stihi.io content")
}
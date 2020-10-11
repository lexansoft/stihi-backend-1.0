package tests

import (
	"github.com/stretchr/testify/assert"
	"gitlab.com/stihi/stihi-backend/app/random"
	"testing"
)

func TestRandomStringFunc(t *testing.T) {
	loopsCount := 10000
	strSize := 4
	for i := 0; i < loopsCount; i++ {
		go func() {
			oldStr := random.String(strSize)
			newStr := random.String(strSize)
			assert.NotEqual(t, oldStr, newStr, "Two random strings should be different")
		}()
	}
}

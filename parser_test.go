package templateparser

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestDecode(t *testing.T) {
	a := assert.New(t)

	parser, err := NewParser()
	if !a.NoError(err) {
		return
	}
	err = parser.DecodeByFilePath("sample.yaml", nil)
	if !a.NoError(err) {
		return
	}

}

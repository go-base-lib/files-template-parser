package templateparser

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestDecode(t *testing.T) {
	a := assert.New(t)

	parser, err := NewParser()
	if !a.NoError(err) {
		return
	}
	err = parser.SetOutput(os.Stdout).DecodeByFilePath("sample.yaml", &ProjectInfo{
		Name: "测试工程",
	})
	if !a.NoError(err) {
		return
	}
}

func TestParse(t *testing.T) {
	a := assert.New(t)

	parser, err := NewParserByWorkPath("templates/01-java/out")
	if !a.NoError(err) {
		return
	}

	if p, err := parser.ParseProjectTemplateInfoByFilePath("/home/slx/works/01-go-base-lib/files-template-parser/templates/01-java/01-gradle-springboot.yaml"); !a.NoError(err) {
		return
	} else {
		fmt.Println(p)
	}

}

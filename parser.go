package templateparser

import (
	"bytes"
	"errors"
	"fmt"
	"gopkg.in/yaml.v3"
	"io"
	"os"
	"path/filepath"
	"strings"
)

type Parser struct {
	TemplateInfo *ProjectTemplateInfo
	WorkerPath   string
}

func NewParserByWorkPath(workerPath string) (*Parser, error) {
	_ = os.RemoveAll(workerPath)
	if err := os.MkdirAll(workerPath, 0777); err != nil {
		return nil, errors.New("工作目录创建失败: " + err.Error())
	}

	return &Parser{
		WorkerPath: workerPath,
	}, nil
}

func NewParser() (*Parser, error) {
	return NewParserByWorkPath("_tmp")
}

// Decode 解析通过二进制
func (p *Parser) Decode(content []byte, projectInfo *ProjectInfo) error {
	return p.DecodeByReader(bytes.NewReader(content), projectInfo)
}

// DecodeByFilePath 解析通过文件路径
func (p *Parser) DecodeByFilePath(filePath string, projectInfo *ProjectInfo) error {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
	if err != nil {
		return errors.New("文件打开失败: " + err.Error())
	}
	defer file.Close()

	return p.DecodeByReader(file, projectInfo)
}

// DecodeByReader 解析通过reader
func (p *Parser) DecodeByReader(reader io.Reader, projectInfo *ProjectInfo) error {
	decoder := yaml.NewDecoder(reader)

	result := &ProjectTemplateInfo{}
	if err := decoder.Decode(&result); err != nil {
		return err
	}

	p.TemplateInfo = result

	return p.parseProjectInfo(projectInfo)
}

// parseProjectInfo 解析工程信息
func (p *Parser) parseProjectInfo(projectInfo *ProjectInfo) error {
	projectInfo = settingProjectInfo(projectInfo)
	thisInfo := &ThisInfo{
		templateData: p.TemplateInfo,
		projectInfo:  projectInfo,
	}
	passData := make(map[string]interface{})
	passData["Project"] = projectInfo
	passData["top"] = p.TemplateInfo
	passData["this"] = thisInfo
	//region 解析env
	thisInfo.Type = ThisTypeEnvs
	if err := parseOrderFieldMap(p.TemplateInfo.Envs, passData, thisInfo, nil); err != nil {
		return err
	}
	//endregion

	//region 解析var
	thisInfo.Type = ThisTypeVars
	if err := parseOrderFieldMap(p.TemplateInfo.Vars, passData, thisInfo, nil); err != nil {
		return err
	}
	//endregion

	//region 解析动态变量
	thisInfo.Type = ThisTypeRemoteVars
	if err := p.parseOrderRemoteVarInfoMap(passData, thisInfo); err != nil {
		return err
	}
	//endregion

	//region 解析静态模板
	thisInfo.Type = ThisTypeTemplates
	thisInfo.Name = "templates"
	if err := p.parserTemplate(p.TemplateInfo.Templates, passData, thisInfo); err != nil {
		return err
	}
	//endregion

	return nil
}

func (p *Parser) parserTemplate(templates map[string]*TemplateFileInfo, data map[string]interface{}, thisInfo *ThisInfo) error {
	for k := range templates {
		v := templates[k]
		thisInfo.Data = v
		if v.Range != "" {
			v.Range = strings.TrimSpace(v.Range)
			v.Range = `{{ pathRange .this ` + v.Range + " }}"
			if _, _, err := getStrByTemplate(v.Range, data, thisInfo); err != nil {
				return err
			}
		}

		pathRange := thisInfo.pathRange
		thisInfo.pathRange = nil
		if pathRange != nil {
			if err := rangeInterface(pathRange, data, func() error {
				return p.writeTemplateContentToTemplateFile(k, v, data, thisInfo)
			}, 0); err != nil {
				return err
			}
			continue
		}
		if err := p.writeTemplateContentToTemplateFile(k, v, data, thisInfo); err != nil {
			return err
		}
	}

	return nil
}

func (p *Parser) writeTemplateContentToTemplateFile(pathTemplate string, fileTemplateInfo *TemplateFileInfo, data map[string]interface{}, thisInfo *ThisInfo) error {
	defer thisInfo.clearWriteData()

	pr, _, err := getStrByTemplate(pathTemplate, data, thisInfo)
	if err != nil {
		return err
	}

	//pr = p.WorkerPath + "/" + pr
	prSplit := strings.Split(pr, "/")

	filePath, _, err := getStrByTemplate(filepath.Join(prSplit...), data, thisInfo)
	filePath = filepath.Join(p.WorkerPath, filePath)
	if err != nil {
		return err
	}

	if fileTemplateInfo.IsDir {
		if err = os.MkdirAll(filePath, 0777); err != nil {
			return errors.New(fmt.Sprintf("创建目录[%s]失败: %s", filePath, err.Error()))
		}
	} else {
		_ = os.MkdirAll(filepath.Dir(filePath), 0777)
	}

	cr, _, err := getBytesByTemplate(fileTemplateInfo.Content, data, thisInfo)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return errors.New(fmt.Sprintf("打开文件[%s]失败: %s", filePath, err.Error()))
	}
	defer file.Close()
	if len(thisInfo.writeFileList) == 0 {
		if _, err = file.Write(cr); err != nil {
			return errors.New(fmt.Sprintf("向文件[%s]写入内容失败: %s", filePath, err.Error()))
		}
		return nil
	}

	for {
		writeFileLen := len(thisInfo.writeFileList)
		if writeFileLen == 0 {
			break
		}

		index := bytes.Index(cr, writeSplitBytes)
		if index == -1 {
			return errors.New("表达式与预期不否，已查找到二进制数据，但未识别标识符")
		}

		if _, err = file.Write(cr[:index]); err != nil {
			return errors.New(fmt.Sprintf("向文件[%s]写入内容失败: %s", filePath, err.Error()))
		}

		if err = thisInfo.writeData(file); err != nil {
			return errors.New(fmt.Sprintf("向文件[%s]写入内容失败: %s", filePath, err.Error()))
		}

		cr = cr[index+writeSplitLen:]
	}

	if len(cr) > 0 {
		if _, err = file.Write(cr); err != nil {
			return errors.New(fmt.Sprintf("向文件[%s]写入内容失败: %s", filePath, err.Error()))
		}
	}
	return nil
}

// parseOrderRemoteVarInfoMap 解析动态变量
func (p *Parser) parseOrderRemoteVarInfoMap(data map[string]interface{}, thisInfo *ThisInfo) (err error) {
	remoteVars := p.TemplateInfo.RemoteVars
	if remoteVars == nil {
		return nil
	}
	keys := remoteVars.Keys()
	if len(keys) == 0 {
		return
	}

	for _, k := range keys {
		thisInfo.Name = k
		val, _ := remoteVars.Get(k)
		if err = val.Parse(data, thisInfo); err != nil {
			return
		}
		if err = val.Req.Do(); err != nil {
			return
		}
	}
	return nil
}

// parseOrderFieldMap 解析排序Map
func parseOrderFieldMap(fieldMap *OrderFieldMap, data map[string]interface{}, thisInfo *ThisInfo, callBakFn func(k, v string) error) (err error) {
	if fieldMap == nil {
		return nil
	}
	keys := fieldMap.Keys()
	if len(keys) == 0 {
		return nil
	}

	for _, k := range keys {
		thisInfo.Name = k
		thisInfo.Data = fieldMap
		v, _ := fieldMap.Get(k)
		v, _, err = getStrByTemplate(v, data, thisInfo)
		if err != nil {
			return
		}
		fieldMap.Set(k, v)
		if callBakFn != nil {
			if err = callBakFn(k, v); err != nil {
				return
			}
		}
	}

	return nil
}

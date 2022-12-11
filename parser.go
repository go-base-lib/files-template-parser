package templateparser

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/go-base-lib/logs"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type Parser struct {
	TemplateInfo *ProjectTemplateInfo
	WorkerPath   string

	bufferWriter *bufio.Writer
	logBlockName string
	breakLog     bool
}

func NewParserByWorkPath(workerPath string) (*Parser, error) {
	_ = os.RemoveAll(workerPath)
	if err := os.MkdirAll(workerPath, 0777); err != nil {
		return nil, errors.New("工作目录创建失败: " + err.Error())
	}

	return &Parser{
		WorkerPath:   workerPath,
		bufferWriter: bufio.NewWriter(io.Discard),
	}, nil
}

func NewParser() (*Parser, error) {
	return NewParserByWorkPath("_tmp")
}

func (p *Parser) LogSuspend() *Parser {
	p.breakLog = true
	return p
}

func (p *Parser) LogRestore() *Parser {
	p.breakLog = false
	return p
}

func (p *Parser) Log(blockName, str string, data ...any) *Parser {
	if p.breakLog {
		return p
	}
	p.logBlockName = blockName
	if !strings.HasSuffix(str, "\n") {
		str += "\n"
	}
	_, _ = p.bufferWriter.WriteString(fmt.Sprintf("["+blockName+"] -> "+str, data...))
	_ = p.bufferWriter.Flush()
	return p
}

func (p *Parser) SetLogBlockName(blockName string) *Parser {
	p.logBlockName = blockName
	return p
}

func (p *Parser) LogWithPrevBlockName(str string, data ...any) *Parser {
	return p.Log(p.logBlockName, str, data...)
}

func (p *Parser) SetOutput(output io.Writer) *Parser {
	p.bufferWriter = bufio.NewWriter(output)
	return p
}

func (p *Parser) ParseProjectTemplateInfo(content []byte) (*ProjectTemplateInfo, error) {
	return p.ParseProjectTemplateInfoByReader(bytes.NewReader(content))
}

func (p *Parser) ParseProjectTemplateInfoByFilePath(filePath string) (*ProjectTemplateInfo, error) {
	file, err := os.OpenFile(filePath, os.O_RDONLY, 0666)
	if err != nil {
		return nil, errors.New("文件打开失败: " + err.Error())
	}
	defer file.Close()
	return p.ParseProjectTemplateInfoByReader(file)
}

func (p *Parser) ParseProjectTemplateInfoByReader(reader io.Reader) (*ProjectTemplateInfo, error) {

	decoder := yaml.NewDecoder(reader)

	result := &ProjectTemplateInfo{}
	if err := decoder.Decode(&result); err != nil {
		return nil, err
	}

	if len(result.Import) != 0 {
		importTemplateInfo := &ProjectTemplateInfo{}
		for _, str := range result.Import {
			if err := p.parserImport(importTemplateInfo, str); err != nil {
				return nil, err
			}
		}

		importTemplateInfo.Shell = result.Shell
		importTemplateInfo.Executes = result.Executes
		p.mergeProjectTemplateInfo(importTemplateInfo, result)
		return importTemplateInfo, nil
	}

	return result, nil
}

func (p *Parser) mergeProjectTemplateInfo(dest *ProjectTemplateInfo, src *ProjectTemplateInfo) {
	if dest.Envs == nil {
		dest.Envs = src.Envs
	} else if src.Envs != nil && src.Envs.m != nil {
		for _, key := range src.Envs.Keys() {
			v, _ := src.Envs.Get(key)
			dest.Envs.Set(key, v)
		}
	}

	if dest.Vars == nil {
		dest.Vars = src.Vars
	} else if src.Vars != nil && src.Vars.m != nil {
		for _, key := range src.Vars.Keys() {
			v, _ := src.Vars.Get(key)
			dest.Vars.Set(key, v)
		}
	}

	if dest.RemoteVars == nil {
		dest.RemoteVars = src.RemoteVars
	} else if src.RemoteVars != nil && src.RemoteVars.m != nil {
		for _, key := range src.RemoteVars.Keys() {
			v, _ := src.RemoteVars.Get(key)
			dest.RemoteVars.Set(key, v)
		}
	}

	if len(dest.Templates) == 0 {
		dest.Templates = src.Templates
	} else {
		for k := range src.Templates {
			dest.Templates[k] = src.Templates[k]
		}
	}
}

func (p *Parser) parserImport(prevTemplateInfo *ProjectTemplateInfo, currentTemplatePath string) error {
	var (
		err error

		currentTemplateInfo *ProjectTemplateInfo
	)
	if strings.HasPrefix(currentTemplatePath, "http://") || strings.HasPrefix(currentTemplatePath, "https://") {
		p.Log("import", currentTemplatePath)
		var resp *http.Response
		if resp, err = globalSkipVerifyCertHttpClient.Get(currentTemplatePath); err != nil {
			return err
		}
		defer resp.Body.Close()

		if currentTemplateInfo, err = p.ParseProjectTemplateInfoByReader(resp.Body); err != nil {
			return err
		}
	} else {
		currentTemplatePath = filepath.Join(p.WorkerPath, currentTemplatePath)
		p.Log("import", currentTemplatePath)
		if currentTemplateInfo, err = p.ParseProjectTemplateInfoByFilePath(currentTemplatePath); err != nil {
			return err
		}
	}

	p.mergeProjectTemplateInfo(prevTemplateInfo, currentTemplateInfo)

	return nil
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
	//decoder := yaml.NewDecoder(reader)

	//result := &ProjectTemplateInfo{}
	//if err := decoder.Decode(&result); err != nil {
	//	return err
	//}

	projectTemplateInfo, err := p.ParseProjectTemplateInfoByReader(reader)
	if err != nil {
		return err
	}

	return p.DecodeByProjectTemplateInfo(projectTemplateInfo, projectInfo)
}

func (p *Parser) DecodeByProjectTemplateInfo(templateInfo *ProjectTemplateInfo, projectInfo *ProjectInfo) error {
	p.TemplateInfo = templateInfo
	return p.parseProjectInfo(projectInfo)
}

// parseProjectInfo 解析工程信息
func (p *Parser) parseProjectInfo(projectInfo *ProjectInfo) error {
	projectInfo = settingProjectInfo(projectInfo)
	cacheDirPath := filepath.Join(p.WorkerPath, ".__temp__.")
	_ = os.MkdirAll(cacheDirPath, 0777)
	defer func() {
		_ = os.RemoveAll(cacheDirPath)
	}()
	thisInfo := &ThisInfo{
		templateData: p.TemplateInfo,
		projectInfo:  projectInfo,
		cacheDirPath: cacheDirPath,
	}
	passData := make(map[string]interface{})
	passData["Project"] = projectInfo
	passData["top"] = p.TemplateInfo
	passData["this"] = thisInfo
	//region 解析env
	logs.Debugln("正在解析环境变量(envs)...")
	p.SetLogBlockName("envs")
	thisInfo.Type = ThisTypeEnvs
	if err := p.parseOrderFieldMap(p.TemplateInfo.Envs, passData, thisInfo, nil, nil); err != nil {
		return err
	}
	//endregion

	//region 解析var
	logs.Debugln("正在解析自定义变量(vars)...")
	p.SetLogBlockName("vars")
	thisInfo.Type = ThisTypeVars
	if err := p.parseOrderFieldMap(p.TemplateInfo.Vars, passData, thisInfo, nil, nil); err != nil {
		return err
	}
	//endregion

	//region 解析动态变量
	logs.Debugln("正在解析远程变量(remoteVars)...")
	p.SetLogBlockName("remoteVars")
	thisInfo.Type = ThisTypeRemoteVars
	if err := p.parseOrderRemoteVarInfoMap(passData, thisInfo); err != nil {
		return err
	}
	//endregion

	shell := DefaultShellConfig.current
	if p.TemplateInfo.Shell.current != "" {
		shell = p.TemplateInfo.Shell.current
	}

	//region 全局pre命令执行器

	if p.TemplateInfo.Executes != nil {
		thisInfo.Type = ThisTypeExecutePre
		thisInfo.Name = string(ThisTypeExecutePre)
		p.SetLogBlockName("executes-pre")
		if err := p.TemplateInfo.Executes.ExecPre(p, shell, passData, thisInfo); err != nil {
			return err
		}

	}

	//endregion

	//region 解析静态模板
	logs.Debugln("正在解析模板(templates)...")
	thisInfo.Type = ThisTypeTemplates
	thisInfo.Name = "templates"
	p.SetLogBlockName("templates")
	if err := p.parserTemplate(p.TemplateInfo.Templates, passData, thisInfo); err != nil {
		return err
	}
	//endregion

	//region 全局Post命令执行器

	if p.TemplateInfo.Executes != nil {
		thisInfo.Type = ThisTypeExecutePost
		thisInfo.Name = string(ThisTypeExecutePost)
		p.SetLogBlockName("executes-post")
		if err := p.TemplateInfo.Executes.ExecPost(p, shell, passData, thisInfo); err != nil {
			return err
		}

	}

	//endregion

	return nil
}

func (p *Parser) parserTemplate(templates map[string]*TemplateFileInfo, data map[string]interface{}, thisInfo *ThisInfo) error {
	for k := range templates {
		v := templates[k]
		if v.Ignore {
			p.LogWithPrevBlockName("${%s} => ignore", k)
			continue
		}
		thisInfo.Data = v
		if v.Range != "" {
			v.Range = strings.TrimSpace(v.Range)
			v.Range = `{{ pathRange .this (` + v.Range + ") }}"
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
		p.LogWithPrevBlockName("${%s} => create dir: %s", pathTemplate, filePath)
		if err = os.MkdirAll(filePath, 0777); err != nil {
			return errors.New(fmt.Sprintf("创建目录[%s]失败: %s", filePath, err.Error()))
		}
		return nil
	}
	_ = os.MkdirAll(filepath.Dir(filePath), 0777)

	if fileTemplateInfo.Content == "" && fileTemplateInfo.Path == "" {
		return fmt.Errorf("文件[%s]缺失内容描述", filePath)
	}

	file, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return errors.New(fmt.Sprintf("打开文件[%s]失败: %s", filePath, err.Error()))
	}
	defer file.Close()

	if fileTemplateInfo.Path != "" {
		filePath, _, err = getStrByTemplate(fileTemplateInfo.Path, data, thisInfo)
		if err != nil {
			return err
		}

		f, err := os.OpenFile(filePath, os.O_RDONLY, 0655)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err = io.Copy(file, f); err != nil {
			return err
		}

		thisInfo.writeFileList = nil
		p.LogWithPrevBlockName("${%s} => copy file: %s", pathTemplate, filePath)
		return nil
	}

	cr, _, err := getBytesByTemplate(fileTemplateInfo.Content, data, thisInfo)
	if err != nil {
		return err
	}

	if len(thisInfo.writeFileList) == 0 {
		if _, err = file.Write(cr); err != nil {
			return errors.New(fmt.Sprintf("向文件[%s]写入内容失败: %s", filePath, err.Error()))
		}
		p.LogWithPrevBlockName("${%s} => write content to: %s", pathTemplate, filePath)
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
	p.LogWithPrevBlockName("${%s} => write content and bytes data to: %s", pathTemplate, filePath)
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
		if err = val.Parse(data, thisInfo, p); err != nil {
			return
		}
		if err = val.Req.Do(p); err != nil {
			return
		}

		if marshal, err := json.Marshal(val.Response.Data); err != nil {
			p.LogWithPrevBlockName("${%s}: result data => %#v", k, val.Response.Data)
		} else {
			p.LogWithPrevBlockName("${%s}: result data => %s", k, marshal)
		}
	}
	return nil
}

// parseOrderFieldMap 解析排序Map
func (p *Parser) parseOrderFieldMap(fieldMap *OrderFieldMap, data map[string]interface{}, thisInfo *ThisInfo, callBakWithStrFn func(k string, v string) error, callBackWithStrSliceFn func(k string, v []string) error) (err error) {
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

		switch _v := v.(type) {
		case string:
			if v, _, err = getStrByTemplate(_v, data, thisInfo); err != nil {
				return
			}
			if callBakWithStrFn != nil {
				if err = callBakWithStrFn(k, _v); err != nil {
					p.LogWithPrevBlockName("%s parse err: %s", k, err.Error())
					return
				}
			}
		case []string:
			for i := range _v {
				if _v[i], _, err = getStrByTemplate(_v[i], data, thisInfo); err != nil {
					p.LogWithPrevBlockName("%s parse err: %s", k, err.Error())
					return
				}
			}
			if callBackWithStrSliceFn != nil {
				if err = callBackWithStrSliceFn(k, _v); err != nil {
					return
				}
			}
		default:
			return errors.New("不支持的变量类型")
		}

		p.LogWithPrevBlockName("${%s}: %s", k, v)
		fieldMap.Set(k, v)
	}

	return nil
}

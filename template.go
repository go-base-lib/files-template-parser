package templateparser

import (
	"bytes"
	"github.com/Masterminds/sprig/v3"
	"strings"
	"text/template"
)

var textTemplate *template.Template

const (
	writeSplit    = "_._^__^_._"
	writeSplitLen = len(writeSplit)
)

var (
	writeSplitBytes = []byte(writeSplit)
)

var (
	// templateFnSubstrByFlag 模板方法截取字符串通过标识
	templateFnSubstrByFlag = func(pos int, flag, str string) string {
		endStr := ""
		if pos < 0 {
		Start:
			lastIndex := strings.LastIndex(str, flag)
			if lastIndex == -1 {
				return ""
			}
			if endStr != "" {
				endStr = str[lastIndex+1:] + "." + endStr
			} else {
				endStr = str[lastIndex+1:] + endStr
			}
			str = str[:lastIndex]
			pos += 1
			if pos < 0 {
				goto Start
			}
		} else {
		StartIndex:
			index := strings.Index(str, flag)
			if index == -1 {
				return ""
			}
			if endStr != "" {
				endStr += "."
			}
			endStr += str[0:index]
			str = str[index+1:]
			pos -= 1
			if pos > 0 {
				goto StartIndex
			}
		}
		return endStr
	}
	// templateFnEnv 获取环境变量
	templateFnEnv = func(envName string, thisInfo *ThisInfo) string {
		return thisInfo.Env(envName)
	}
	// templateFnVar 获取变量
	templateFnVar = func(varName string, thisInfo *ThisInfo) any {
		return thisInfo.Var(varName)
	}
	// templateFnRemoteVar 解析远程参数
	templateFnRemoteVar = func(varName string, thisInfo *ThisInfo) *RemoteVarInfo {
		return thisInfo.RemoteVar(varName)
	}
	// templateFnRemoteVarResponse 获取远程参数的响应
	templateFnRemoteVarResponse = func(varName string, thisInfo *ThisInfo) *ResponseInfo {
		remoteVar := thisInfo.RemoteVar(varName)
		if remoteVar == nil {
			return nil
		}
		return remoteVar.Response
	}
	// templateFnWriteBytes 写入二进制
	templateFnWriteBytes = func(d []byte, thisInfo *ThisInfo) string {
		thisInfo.addWriteData(d)
		return writeSplit
	}
	// 路径循环
	templateFnPathRange = func(thisInfo *ThisInfo, data ...interface{}) string {
		thisInfo.pathRange = data
		return ""
	}
)

func init() {
	funcMap := sprig.TxtFuncMap()
	funcMap["substrByFlag"] = templateFnSubstrByFlag
	funcMap["env"] = templateFnEnv
	funcMap["var"] = templateFnVar
	funcMap["remoteVar"] = templateFnRemoteVar
	funcMap["remoteVarResponse"] = templateFnRemoteVarResponse
	funcMap["writeBytes"] = templateFnWriteBytes
	funcMap["pathRange"] = templateFnPathRange
	textTemplate = template.New("base").Funcs(funcMap)
}

type AnyString interface{ ~string }

func getStrByTemplate[K AnyString](str K, data map[string]interface{}, thisInfo *ThisInfo) (K, interface{}, error) {
	_str := string(str)
	_str = strings.TrimSpace(_str)
	if len(_str) == 0 {
		return str, nil, nil
	}
	parse, err := textTemplate.Parse(_str)
	if err != nil {
		return "", nil, err
	}

	buffer := &bytes.Buffer{}
	if err = parse.Execute(buffer, data); err != nil {
		return "", nil, err
	}

	err = thisInfo.error()
	if err != nil {
		return K(buffer.String()), nil, err
	}

	return K(buffer.String()), thisInfo.getReturnData(), err
}

func getBytesByTemplate(str string, data map[string]interface{}, thisInfo *ThisInfo) ([]byte, interface{}, error) {
	parse, err := textTemplate.Parse(str)
	if err != nil {
		return nil, nil, err
	}

	buffer := &bytes.Buffer{}
	if err = parse.Execute(buffer, data); err != nil {
		return nil, nil, err
	}

	err = thisInfo.error()
	if err != nil {
		return buffer.Bytes(), nil, err
	}

	return buffer.Bytes(), thisInfo.getReturnData(), err
}

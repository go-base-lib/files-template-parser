package templateparser

import (
	"errors"
	"fmt"
	"github.com/iancoleman/orderedmap"
	"gopkg.in/yaml.v3"
	"strings"
)

type ThisType string

const (
	ThisTypeEnvs       ThisType = "env"
	ThisTypeVars       ThisType = "vars"
	ThisTypeRemoteVars ThisType = "remoteVars"
	ThisTypeTemplates  ThisType = "templates"
)

type TemplateFileInfo struct {
	// Path 文件路径
	Path string `yaml:"path,omitempty"`
	// Content 文件内容
	Content string `yaml:"content,omitempty"`
	// IsDir 是否为目录
	IsDir bool `yaml:"isDir,omitempty"`
	// Range 循环表达式
	Range string `yaml:"range,omitempty"`
	// Comment 注释
	Comment string `yaml:"comment,omitempty"`
	// Ignore 忽略
	Ignore bool `yaml:"ignore,omitempty"`
}

type ResponseInfo struct {
	ExitCode            string
	ExitMsg             string
	ResponseRawFilePath string
	Data                interface{}
	Metadata            interface{}
}

type OrderFieldMap struct {
	m *orderedmap.OrderedMap
}

func (o *OrderFieldMap) UnmarshalYAML(value *yaml.Node) error {
	r := orderedmap.New()
	contentLen := len(value.Content)
	switch value.Kind {
	case yaml.MappingNode:
		if contentLen%2 != 0 {
			return errors.New(fmt.Sprintf("行: %d, 列: %d, 错误的Map类型", value.Line, value.Column))
		}
		for i := 0; i < contentLen; i += 2 {
			key := value.Content[i]
			val := value.Content[i+1]

			//if val.Kind != key.Kind {
			//	res := new(any)
			//	if err := val.Decode(&res); err != nil {
			//		panic(err)
			//	}
			//	r.Set(key.Value, res)
			//	continue
			//}
			if val.Kind == yaml.SequenceNode {
				values := make([]string, 0, len(value.Content))
				for i := range val.Content {
					v := val.Content[i]
					if v.Kind != yaml.ScalarNode {
						return fmt.Errorf("行%d: 不支持的数据类型", v.Line)
					}
					values = append(values, v.Value)
				}
				r.Set(key.Value, values)
				continue
			}

			if val.Kind != yaml.ScalarNode {
				return fmt.Errorf("行%d: 不支持的数据类型", val.Line)
			}

			r.Set(key.Value, val.Value)
		}

	case yaml.SequenceNode:
		for i := 0; i < contentLen; i++ {
			val := value.Content[i]
			if val.Tag != "!!str" {
				return errors.New(fmt.Sprintf("行: %d, 列: %d, 错误的数据类型,应为: key=value 格式", value.Line, value.Column))
			}
			valStr := val.Value
			valSplit := strings.Split(valStr, "=")
			k := valSplit[0]
			v := ""
			if len(valSplit) == 2 {
				v = valSplit[1]
			}
			r.Set(k, v)
		}

	default:
		return errors.New(fmt.Sprintf("行: %d, 列: %d, 错误的Map类型", value.Line, value.Column))
	}
	o.m = r
	return nil
}

func (o *OrderFieldMap) Get(key string) (res any, ok bool) {
	v, b := o.m.Get(key)
	if !b || v == nil {
		return "", b
	}

	if res, ok = v.(string); ok {
		return
	}

	res, ok = v.([]string)
	return
}

func (o *OrderFieldMap) Set(key string, info any) {
	o.m.Set(key, info)
}

func (o *OrderFieldMap) Delete(key string) {
	o.m.Delete(key)
}

func (o *OrderFieldMap) Keys() []string {
	return o.m.Keys()
}

// SortKeys Sort the map keys using your sort func
func (o *OrderFieldMap) SortKeys(sortFunc func(keys []string)) {
	o.m.SortKeys(sortFunc)
}

// Sort the map using your sort func
func (o *OrderFieldMap) Sort(lessFunc func(a *orderedmap.Pair, b *orderedmap.Pair) bool) {
	o.m.Sort(lessFunc)
}

type OrderRemoteVarInfoMap struct {
	m *orderedmap.OrderedMap
}

func (o *OrderRemoteVarInfoMap) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return errors.New(fmt.Sprintf("行: %d, 列: %d, 错误的数据类型", value.Line, value.Column))
	}

	contentLen := len(value.Content)
	if contentLen%2 != 0 {
		return errors.New(fmt.Sprintf("行: %d, 列: %d, 错误的Map类型", value.Line, value.Column))
	}

	r := orderedmap.New()
	for i := 0; i < contentLen; i += 2 {
		key := value.Content[i].Value
		valContent := value.Content[i+1]
		if valContent.Tag == "!!null" {
			r.Set(key, nil)
			continue
		}

		var val *RemoteVarParser
		if err := valContent.Decode(&val); err != nil {
			return err
		}
		r.Set(key, val)
	}

	o.m = r
	return nil
}

func (o *OrderRemoteVarInfoMap) Get(key string) (*RemoteVarParser, bool) {
	v, b := o.m.Get(key)
	if !b || v == nil {
		return nil, b
	}

	return v.(*RemoteVarParser), b
}

func (o *OrderRemoteVarInfoMap) Set(key string, info *RemoteVarParser) {
	o.m.Set(key, info)
}

func (o *OrderRemoteVarInfoMap) Delete(key string) {
	o.m.Delete(key)
}

func (o *OrderRemoteVarInfoMap) Keys() []string {
	return o.m.Keys()
}

// SortKeys Sort the map keys using your sort func
func (o *OrderRemoteVarInfoMap) SortKeys(sortFunc func(keys []string)) {
	o.m.SortKeys(sortFunc)
}

// Sort Sort the map using your sort func
func (o *OrderRemoteVarInfoMap) Sort(lessFunc func(a *orderedmap.Pair, b *orderedmap.Pair) bool) {
	o.m.Sort(lessFunc)
}

// SupportRemoteReqType 支持远程请求类别
type SupportRemoteReqType string

func (s *SupportRemoteReqType) ToLower() {
	*s = SupportRemoteReqType(strings.ToLower(string(*s)))
}

const (
	SupportRemoteReqTypeHttp  SupportRemoteReqType = "http"
	SupportRemoteReqTypeHttps SupportRemoteReqType = "https"
)

// HttpUploadFileFormInfo http文件上传表单内容
type HttpUploadFileFormInfo struct {
	// Files 文件
	Files *OrderFieldMap `yaml:"files,omitempty"`
	// Data 数据
	Data *OrderFieldMap `yaml:"data,omitempty"`
}

// RemoteVarInfo 动态变量信息
type RemoteVarInfo struct {
	// Type 类型
	Type SupportRemoteReqType `yaml:"type,omitempty"`
	// Url 请求路径
	Url string `json:"url,omitempty"`
	// Method 请求方式
	Method string `yaml:"method,omitempty"`
	// Headers 请求头
	Headers *OrderFieldMap `yaml:"headers,omitempty"`
	// RequestUploadFiles 请求上传文件信息
	RequestUploadFiles *HttpUploadFileFormInfo `yaml:"requestUploadFiles,omitempty"`
	// RequestParams 请求参数
	RequestParams *OrderFieldMap `yaml:"requestParams,omitempty"`
	// RequestFormData 表单请求数据
	RequestFormData *OrderFieldMap `yaml:"requestFormData,omitempty"`
	// RequestBody 请求身体
	RequestBody string `yaml:"requestBody,omitempty"`
	// ResponseJudge 响应结果判断, 模板返回true/string, true: 正确, 其他: 错误信息
	ResponseJudge string `yaml:"responseJudge,omitempty"`
	// ResponseParser 内置响应数据解析器: json | text | base64 | hex
	ResponseParser string `yaml:"responseParser,omitempty"`
	// PostResponseParser 内置解析器无法满足时使用的自定义响应解析器
	PostResponseParser string `yaml:"postResponseParser,omitempty"`
	// SkipHttpsVerifyCert 跳过https的证书认证
	SkipHttpsVerifyCert bool `yaml:"skipHttpsVerifyCert,omitempty"`
	// Req 请求接口
	Req RequestInterface `yaml:"-"`
	// Response 请求响应数据
	Response *ResponseInfo `yaml:"-"`
	// Name 变量名称
	//Name string `yaml:"-"`
}

type RemoteVarParser struct {
	*RemoteVarInfo
	line   int
	column int
}

func (d *RemoteVarParser) Parse(data map[string]interface{}, thisInfo *ThisInfo, p *Parser) (err error) {
	if d.Url == "" {
		return errors.New(fmt.Sprintf("行: %d, 列: %d, 缺失url(请求路径)", d.line, d.column))
	}

	thisInfo.Data = d.RemoteVarInfo

	if d.Type, _, err = getStrByTemplate(d.Type, data, thisInfo); err != nil {
		return err
	}

	d.Type.ToLower()

	d.Url, _, err = getStrByTemplate(d.Url, data, thisInfo)
	if err != nil {
		return
	}

	d.Method = strings.ToUpper(d.Method)
	if d.Method == "" {
		d.Method = "GET"
	}

	switch d.Type {
	case SupportRemoteReqTypeHttp:
		fallthrough
	case SupportRemoteReqTypeHttps:
		if err := createHttpRequestByVar(d.RemoteVarInfo, data, thisInfo, p); err != nil {
			return err
		}
	default:
		return errors.New(fmt.Sprintf(fmt.Sprintf("行: %d, 列: %d, 不支持的type(获取类型): %s", d.line, d.column, d.Type)))
	}
	return nil
}

func (d *RemoteVarParser) UnmarshalYAML(value *yaml.Node) error {
	var r *RemoteVarInfo
	if err := value.Decode(&r); err != nil {
		return err
	}
	d.line = value.Line
	d.column = value.Column
	d.RemoteVarInfo = r
	return nil
}

type ExecuteInfo struct {
	Post []string `yaml:"post,omitempty"`
	Pre  []string `yaml:"pre,omitempty"`
}

type ProjectComments struct {
	Envs       map[string]string `yaml:"envs,omitempty"`
	Vars       map[string]string `yaml:"vars,omitempty"`
	RemoteVars map[string]string `yaml:"remoteVars,omitempty"`
}

type ProjectTemplateInfo struct {
	// Import 导入
	Import []string `yaml:"import,omitempty"`
	// Env 环境变量
	Envs *OrderFieldMap `yaml:"envs,omitempty"`
	// Vars 变量
	Vars *OrderFieldMap `yaml:"vars,omitempty"`
	// RemoteVars 动态变量
	RemoteVars *OrderRemoteVarInfoMap `yaml:"remoteVars,omitempty"`
	// Templates 静态模板
	Templates map[string]*TemplateFileInfo `yaml:"templates,omitempty"`
	// Execute 执行器, 在模板创建完成之后执行
	Execute *ExecuteInfo `yaml:"execute,omitempty"`
	// Comments 注释
	Comments *ProjectComments `yaml:"comments,omitempty"`
}

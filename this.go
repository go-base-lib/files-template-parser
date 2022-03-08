package templateparser

import (
	"errors"
	"io"
)

type ThisInfo struct {
	// projectInfo 工程信息
	projectInfo *ProjectInfo
	// templateData 模板数据
	templateData *ProjectTemplateInfo
	// Type 类别
	Type ThisType
	// Name 名称
	Name string
	// Data 数据
	Data interface{}
	// writeFileList 写入文件列表
	writeFileList [][]byte
	// 路径循环条件
	pathRange []interface{}
	// err 错误
	err error
	// returnData 响应数据
	returnData interface{}
}

// getReturnData 获取返回值
func (t *ThisInfo) getReturnData() interface{} {
	r := t.returnData
	t.returnData = nil
	return r
}

// Return 设置返回值
func (t *ThisInfo) Return(data interface{}) string {
	t.returnData = data
	return ""
}

// error 获取错误信息
func (t *ThisInfo) error() error {
	err := t.err
	t.err = nil
	return err
}

// Error 设置错误
func (t *ThisInfo) Error(errStr string) string {
	t.err = errors.New(errStr)
	return ""
}

// Env 获取环境变量
func (t *ThisInfo) Env(name string) string {
	v, _ := t.templateData.Envs.Get(name)
	return v
}

// Var 获取变量
func (t *ThisInfo) Var(name string) string {
	v, _ := t.templateData.Vars.Get(name)
	return v
}

// RemoteVar 获取远程变量
func (t *ThisInfo) RemoteVar(name string) *RemoteVarInfo {
	v, b := t.templateData.RemoteVars.Get(name)
	if !b {
		return nil
	}
	return v.RemoteVarInfo
}

// addWriteData 添加写入数据
func (t *ThisInfo) addWriteData(d []byte) {
	if t.writeFileList == nil {
		t.writeFileList = make([][]byte, 0, 8)
	}
	t.writeFileList = append(t.writeFileList, d)
}

// clearWriteData 清空写入数据
func (t *ThisInfo) clearWriteData() {
	if t.writeFileList != nil {
		t.writeFileList = nil
	}
}

// writeData 写入数据
func (t *ThisInfo) writeData(writer io.Writer) error {
	if len(t.writeFileList) == 0 {
		return nil
	}

	writer.Write(t.writeFileList[0])
	t.writeFileList = t.writeFileList[1:]
	return nil
}

// VersionName 获取版本名称
func (t *ThisInfo) VersionName() string {
	return t.projectInfo.Version.Name
}

// CurrentDepends 获取当前普通依赖列表
func (t *ThisInfo) CurrentDepends(groupName ...string) []*ProjectDependInfo {
	if len(groupName) == 0 {
		keys := t.projectInfo.Depends.Current.Keys()
		if len(keys) == 0 {
			return make([]*ProjectDependInfo, 0, 0)
		}
		groupName = []string{keys[0]}
	}

	var result []*ProjectDependInfo
	for _, n := range groupName {
		groupInfo, ok := t.projectInfo.Depends.Current.Get(n)
		if !ok {
			continue
		}

		if result == nil {
			result = make([]*ProjectDependInfo, 0, len(groupInfo.Depends.Default))
		}
		result = append(result, groupInfo.Depends.Default...)
	}

	return result
}

// CurrentPluginDepends 获取当前插件依赖
func (t *ThisInfo) CurrentPluginDepends(groupName ...string) []*ProjectDependInfo {
	if len(groupName) == 0 {
		keys := t.projectInfo.Depends.Current.Keys()
		if len(keys) == 0 {
			return make([]*ProjectDependInfo, 0, 0)
		}
		groupName = []string{keys[0]}
	}

	var result []*ProjectDependInfo
	for _, n := range groupName {
		groupInfo, ok := t.projectInfo.Depends.Current.Get(n)
		if !ok {
			continue
		}

		if result == nil {
			result = make([]*ProjectDependInfo, 0, len(groupInfo.Depends.Plugin))
		}
		result = append(result, groupInfo.Depends.Plugin...)
	}

	return result
}

// DependGroup 获取依赖分组信息
func (t *ThisInfo) DependGroup(groupName string) *ProjectDependsGroupInfo {
	groupInfo, ok := t.projectInfo.Depends.Group.Get(groupName)
	if !ok {
		return nil
	}
	return groupInfo
}

// ProductName 产品名称
func (t *ThisInfo) ProductName() string {
	return t.projectInfo.Product.Name
}

// ModuleName 模块名称
func (t *ThisInfo) ModuleName() string {
	return t.projectInfo.Name
}

// Product 产品信息
func (t *ThisInfo) Product() *ProjectProductInfo {
	return t.projectInfo.Product
}

// Version 版本信息
func (t *ThisInfo) Version() *ProjectVersionInfo {
	return t.projectInfo.Version
}

// ModuleDesc 模块描述
func (t *ThisInfo) ModuleDesc() string {
	return t.projectInfo.Desc
}

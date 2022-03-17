package templateparser

import "github.com/iancoleman/orderedmap"

type OrderProjectDependGroupMap struct {
	m *orderedmap.OrderedMap
}

func (o *OrderProjectDependGroupMap) Get(key string) (*ProjectDependsGroupInfo, bool) {
	v, b := o.m.Get(key)
	if !b || v == nil {
		return nil, b
	}

	return v.(*ProjectDependsGroupInfo), b
}

func (o *OrderProjectDependGroupMap) Set(key string, info *ProjectDependsGroupInfo) {
	o.m.Set(key, info)
}

func (o *OrderProjectDependGroupMap) Delete(key string) {
	o.m.Delete(key)
}

func (o *OrderProjectDependGroupMap) Keys() []string {
	return o.m.Keys()
}

// SortKeys Sort the map keys using your sort func
func (o *OrderProjectDependGroupMap) SortKeys(sortFunc func(keys []string)) {
	o.m.SortKeys(sortFunc)
}

// Sort Sort the map using your sort func
func (o *OrderProjectDependGroupMap) Sort(lessFunc func(a *orderedmap.Pair, b *orderedmap.Pair) bool) {
	o.m.Sort(lessFunc)
}

type ProjectInfo struct {
	// Product 产品信息
	Product *ProjectProductInfo `json:"product,omitempty"`
	// Version 版本信息
	Version *ProjectVersionInfo `json:"version,omitempty"`
	// Depends 依赖信息
	Depends *ProjectDependsContainerInfo `json:"depends,omitempty"`
	// Modules 模块信息
	Modules map[string]*ProjectModule `json:"modules,omitempty"`
	// Name 模块名称
	Name string `json:"name,omitempty"`
	// Desc 模块描述
	Desc string `json:"desc,omitempty"`
}

// ProjectModule 工程模块信息
type ProjectModule struct {
	// Name 模块名称
	Name string `json:"name,omitempty"`
	// Desc 描述
	Desc string `json:"desc,omitempty"`
}

// ProjectDependsContainerInfo 依赖外层包裹
type ProjectDependsContainerInfo struct {
	// Current 当前依赖列表
	Current *OrderProjectDependGroupMap `json:"current,omitempty"`
	// Group 分组信息
	Group *OrderProjectDependGroupMap `json:"group,omitempty"`
}

type ProjectDependsGroupInfo struct {
	// Name 名称
	Name string `json:"name,omitempty"`
	// Desc 描述
	Desc string `json:"desc,omitempty"`
	// Depends 依赖
	Depends *ProjectDependsInfo `json:"depends,omitempty"`
}

// ProjectDependsInfo 工程依赖信息
type ProjectDependsInfo struct {
	// Plugin 插件依赖
	Plugin []*ProjectDependInfo `json:"plugin,omitempty"`
	// Default 普通依赖
	Default []*ProjectDependInfo `json:"default,omitempty"`
}

type ProjectDependInfo struct {
	// Name 依赖名称
	Name string `json:"name,omitempty"`
	// Version 当前版本信息
	Version string `json:"version,omitempty"`
	// Desc 描述
	Desc string `json:"desc,omitempty"`
	// Metadata 原始数据
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ProjectVersionInfo 版本信息
type ProjectVersionInfo struct {
	// Name 版本名称
	Name string
	// Desc 版本描述
	Desc string
}

// ProjectProductInfo 产品信息
type ProjectProductInfo struct {
	// Name 名称
	Name string `json:"name,omitempty"`
	// CodeName 代号
	CodeName string `json:"codeName,omitempty"`
	// Type 类别
	Type string `json:"type,omitempty"`
	// Desc 描述
	Desc string `json:"desc,omitempty"`
	// Background 背景
	Background string `json:"background,omitempty"`
}

// settingProjectInfo 设置工程信息
func settingProjectInfo(p *ProjectInfo) *ProjectInfo {
	if p == nil {
		p = &ProjectInfo{}
	}
	if p.Product == nil {
		p.Product = &ProjectProductInfo{}
	}

	if p.Version == nil {
		p.Version = &ProjectVersionInfo{}
	}

	if p.Depends == nil {
		p.Depends = &ProjectDependsContainerInfo{}
	}

	if p.Depends.Current == nil {
		p.Depends.Current = NewOrderProjectDependGroupMap()
	}

	if p.Depends.Group == nil {
		p.Depends.Group = NewOrderProjectDependGroupMap()
	}

	if p.Modules == nil {
		p.Modules = make(map[string]*ProjectModule, 0)
	}

	return p
}

func NewOrderProjectDependGroupMap() *OrderProjectDependGroupMap {
	return &OrderProjectDependGroupMap{
		m: orderedmap.New(),
	}
}

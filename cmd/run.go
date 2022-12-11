package main

import (
	"encoding/json"
	"flag"
	"fmt"
	templateparser "github.com/devloperPlatform/devplatform-project-template-parser"
	"os"
	"path/filepath"
)

func main() {
	templateFileName := flag.String("template", "", "要解析的文件模板地址")
	projectJsonInfo := flag.String("projectinfo", "", "要设置的工程信息")
	workPath := flag.String("workpath", "", "工作路径, 默认为模板文件所在目录的out目录")

	flag.Parse()

	if *templateFileName == "" {
		_, _ = os.Stderr.WriteString("要解析的模板地址不能为空")
		return
	}

	if _templateFilePath, err := filepath.Abs(*templateFileName); err != nil {
		_, _ = os.Stderr.WriteString("获取模板文件的绝对路径失败: " + err.Error())
		return
	} else {
		*templateFileName = _templateFilePath
	}

	if stat, err := os.Stat(*templateFileName); err != nil || stat.IsDir() {
		_, _ = os.Stderr.WriteString("模板文件路径非文件或不存在")
		return
	}

	if workPath == nil || *workPath == "" {
		*workPath = filepath.Join(filepath.Dir(*templateFileName), "out")
	}

	var projectInfo *templateparser.ProjectInfo
	if projectJsonInfo != nil && *projectJsonInfo != "" {
		if err := json.Unmarshal([]byte(*projectJsonInfo), &projectInfo); err != nil {
			_, _ = os.Stderr.WriteString("解析工程信息失败: " + err.Error())
			return
		}
	}

	if projectInfo == nil {
		projectInfo = &templateparser.ProjectInfo{}
	}

	marshal, _ := json.Marshal(projectInfo)

	fmt.Printf("解析信息输出:\n模板地址: %s\n工作路径: %s\n工程信息: %s\n\n------------------------------------------------\n\n", *templateFileName, *workPath, marshal)

	parser, err := templateparser.NewParserByWorkPath(*workPath)
	if err != nil {
		_, _ = os.Stderr.WriteString(err.Error())
		return
	}

	if err = parser.SetOutput(os.Stdout).DecodeByFilePath(*templateFileName, projectInfo); err != nil {
		_, _ = os.Stderr.WriteString(err.Error())
		return
	}

}

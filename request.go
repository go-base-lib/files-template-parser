package templateparser

import (
	"bytes"
	"crypto/tls"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

var globalSkipVerifyCertHttpClient = &http.Client{
	Transport: &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	},
}

type RequestInterface interface {
	// Do 请求
	Do(p *Parser) error
}

type httpRequest struct {
	varInfo  *RemoteVarInfo
	req      *http.Request
	client   *http.Client
	data     map[string]interface{}
	thisInfo *ThisInfo
}

func (h *httpRequest) Do(p *Parser) error {
	res, err := h.client.Do(h.req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	d := &ResponseInfo{
		ExitCode: strconv.Itoa(res.StatusCode),
		ExitMsg:  res.Status,
		Metadata: res,
	}

	p.LogWithPrevBlockName("${%s}: response status code: %d, status text: %s", h.thisInfo.Name, res.StatusCode, res.Status)

	if h.varInfo.ResponseJudge != "" {
		h.thisInfo.Data = &d
		if _, _, err = getStrByTemplate(h.varInfo.ResponseJudge, h.data, h.thisInfo); err != nil {
			return errors.New(err.Error())
		}
	} else if res.StatusCode != 200 {
		return errors.New(res.Status)
	}

	resStoreDir := filepath.Join(h.thisInfo.cacheDirPath, "remoteVars", h.thisInfo.Name, "http")
	_ = os.MkdirAll(resStoreDir, 0777)

	resFilePath := filepath.Join(resStoreDir, "_response.raw")
	resFile, err := os.OpenFile(resFilePath, os.O_CREATE|os.O_RDWR, 0655)
	if err != nil {
		return fmt.Errorf("remoteVars[%s]: 创建http远程请求响应缓存失败: %w", h.thisInfo.Name, err)
	}
	defer resFile.Close()

	if _, err = io.Copy(resFile, res.Body); err != nil {
		return fmt.Errorf("remoteVars[%s]: 保存远程响应结果失败: %w", h.thisInfo.Name, err)
	}

	//resDataBytes, err := io.ReadAll(res.Body)
	//if err != nil {
	//	return err
	//}

	if _, err = resFile.Seek(0, 0); err != nil {
		return fmt.Errorf("remoteVars[%s]: 复原文件指针失败: %w", h.thisInfo.Name, err)
	}

	var resData interface{} = resFilePath
	if h.varInfo.ResponseParser != "" {
		var resDataBytes []byte
		if resDataBytes, err = io.ReadAll(resFile); err != nil {
			return fmt.Errorf("remoteVars[%s]: 读取完整内容失败: %w", h.thisInfo.Name, err)
		}

		split := strings.Split(h.varInfo.ResponseParser, "|")
		for _, s := range split {
			s = strings.TrimSpace(strings.ToLower(s))
			switch s {
			case "text":
				resData = string(resDataBytes)
				p.LogWithPrevBlockName("${%s}: response text parser => %s", h.thisInfo.Name, resData)
			case "json":
				var _d interface{}
				if err = json.Unmarshal(resDataBytes, &_d); err != nil {
					return err
				}
				resData = &_d
				p.LogWithPrevBlockName("${%s}: response json parser => %s", h.thisInfo.Name, resDataBytes)
			case "hex":
				resDataBytes, err = hex.DecodeString(string(resDataBytes))
				if err != nil {
					return err
				}
				resData = resDataBytes
				p.LogWithPrevBlockName("${%s}: response hex parser => %s", h.thisInfo.Name, resData)
			case "base64":
				resDataBytes, err = base64.StdEncoding.DecodeString(string(resDataBytes))
				if err != nil {
					return err
				}
				resData = resDataBytes
				p.LogWithPrevBlockName("${%s}: response base64 parser => %s", h.thisInfo.Name, resData)
			}
		}
	}

	d.Data = resData

	if h.varInfo.PostResponseParser != "" {
		h.thisInfo.Data = d
		_, d.Data, err = getStrByTemplate(h.varInfo.PostResponseParser, h.data, h.thisInfo)
		if err != nil {
			return err
		}
	}

	h.varInfo.Response = d
	return nil
}

func createHttpRequestByVar(varInfo *RemoteVarInfo, data map[string]interface{}, thisInfo *ThisInfo, p *Parser) error {
	if !strings.HasPrefix(varInfo.Url, string(varInfo.Type)) {
		varInfo.Url = fmt.Sprintf("%s://%s", varInfo.Type, varInfo.Url)
	}
	p.LogWithPrevBlockName("${%s}: type => %s", thisInfo.Name, varInfo.Type)
	p.LogWithPrevBlockName("${%s}: method => %s", thisInfo.Name, varInfo.Method)

	var requestBody io.Reader
	if varInfo.RequestParams != nil && len(varInfo.RequestParams.Keys()) > 0 {
		buf := &bytes.Buffer{}
		if err := p.parseOrderFieldMap(varInfo.RequestParams, data, thisInfo, func(k string, v string) error {
			buf.WriteString(k)
			buf.WriteRune('=')
			buf.WriteString(v)
			buf.WriteRune('&')
			return nil
		}, func(k string, v []string) error {
			for i := range v {
				buf.WriteString(k)
				buf.WriteString("[]=")
				buf.WriteString(v[i])
				buf.WriteRune('&')
			}
			return nil
		}); err != nil {
			return err
		}
		str := buf.String()
		if len(str) > 0 {
			str = str[:len(str)-1]
		}
		varInfo.Url += "?" + str
	}

	p.LogWithPrevBlockName("${%s}: url => %s", thisInfo.Name, varInfo.Url)

	if varInfo.RequestBody != "" {
		thisInfo.Data = varInfo
		reqBody, _, err := getStrByTemplate(varInfo.RequestBody, data, thisInfo)
		if err != nil {
			return err
		}
		p.LogWithPrevBlockName("${%s}: request body => %s", thisInfo.Name, reqBody)
		requestBody = strings.NewReader(reqBody)
	} else if varInfo.RequestFormData != nil && len(varInfo.RequestFormData.Keys()) > 0 {
		vals := make(url.Values)
		if err := p.parseOrderFieldMap(varInfo.RequestFormData, data, thisInfo, func(k string, v string) error {
			vals[k] = []string{v}
			return nil
		}, func(k string, v []string) error {
			vals[k] = v
			return nil
		}); err != nil {
			return err
		}
		formData := vals.Encode()

		p.LogWithPrevBlockName("${%s}: request form data => %s", thisInfo.Name, formData)

		requestBody = strings.NewReader(formData)
	} else if varInfo.RequestUploadFiles != nil {
		files := varInfo.RequestUploadFiles.Files
		datas := varInfo.RequestUploadFiles.Data

		filesLen := len(files.Keys())
		datasLen := len(datas.Keys())

		if filesLen > 0 || datasLen > 0 {
			buf := &bytes.Buffer{}
			fileWriter := multipart.NewWriter(buf)
			if filesLen > 0 {
				p.LogSuspend()
				if err := p.parseOrderFieldMap(files, data, thisInfo, func(k string, v string) error {
					p.LogRestore()
					defer p.LogSuspend()
					filename := filepath.Base(v)
					p.LogWithPrevBlockName("${%s}: upload file, name: %s, file path => %s", thisInfo.Name, filename, v)
					file, err := fileWriter.CreateFormFile(k, filename)
					if err != nil {
						return err
					}
					if err = copyFile2Writer(v, file); err != nil {
						return err
					}
					return nil
				}, func(k string, v []string) error {
					p.LogRestore()
					defer p.LogSuspend()
					for i := range v {
						_v := v[i]
						filename := filepath.Base(_v)
						p.LogWithPrevBlockName("${%s}: upload file, name: %s, file path => %s", thisInfo.Name, filename, v)
						file, err := fileWriter.CreateFormFile(k, filename)
						if err != nil {
							return err
						}
						if err = copyFile2Writer(_v, file); err != nil {
							return err
						}
					}
					return nil
				}); err != nil {
					return err
				}
				p.LogRestore()
			}

			if datasLen > 0 {
				p.LogSuspend()
				if err := p.parseOrderFieldMap(datas, data, thisInfo, func(k string, v string) error {
					p.LogRestore()
					defer p.LogSuspend()

					p.LogWithPrevBlockName("${%s}: upload file With data: ${%s} => %s", thisInfo.Name, k, v)
					return fileWriter.WriteField(k, v)
				}, func(k string, v []string) error {
					p.LogRestore()
					defer p.LogSuspend()

					for i := range v {
						_v := v[i]
						_k := fmt.Sprintf("%s[%d]", k, i)
						if err := fileWriter.WriteField(_k, _v); err != nil {
							return err
						}
						p.LogWithPrevBlockName("${%s}: upload file With data: ${%s} => %v", thisInfo.Name, _k, _v)
					}

					return nil
				}); err != nil {
					return err
				}
				p.LogRestore()
			}

			requestBody = buf
		}

	}

	req, err := http.NewRequest(varInfo.Method, varInfo.Url, requestBody)
	if err != nil {
		return err
	}

	req.Header.Set("User-Agent", "teamwork-lib-file-template-parser")
	p.LogSuspend()
	defer p.LogRestore()
	if err = p.parseOrderFieldMap(varInfo.RequestParams, data, thisInfo, func(k string, v string) error {
		req.Header.Add(k, v)
		return nil
	}, func(k string, v []string) error {
		for i := range v {
			req.Header.Add(k, v[i])
		}
		return nil
	}); err != nil {
		p.LogWithPrevBlockName("${%s}: parse header error: %s", thisInfo.Name, err.Error())
		return err
	}
	p.LogRestore()
	marshal, _ := json.Marshal(req.Header)
	p.LogWithPrevBlockName("${%s}: header => %s", thisInfo.Name, marshal)

	httpClient := http.DefaultClient

	if varInfo.Type == SupportRemoteReqTypeHttps && varInfo.SkipHttpsVerifyCert {
		httpClient = globalSkipVerifyCertHttpClient
	}

	varInfo.Req = &httpRequest{
		varInfo:  varInfo,
		req:      req,
		client:   httpClient,
		data:     data,
		thisInfo: thisInfo,
	}

	return nil
}

func copyFile2Writer(src string, dest io.Writer) error {
	file, err := os.OpenFile(src, os.O_RDONLY, 0666)
	if err != nil {
		return errors.New("打开文件(" + src + ")失败, err => " + err.Error())
	}
	defer file.Close()

	_, err = io.Copy(dest, file)
	if err != nil {
		return errors.New("拷贝文件(" + src + ")失败 => " + err.Error())
	}
	return nil
}

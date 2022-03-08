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
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type RequestInterface interface {
	// Do 请求
	Do() error
}

type httpRequest struct {
	varInfo  *RemoteVarInfo
	req      *http.Request
	client   *http.Client
	data     map[string]interface{}
	thisInfo *ThisInfo
}

func (h *httpRequest) Do() error {
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

	if h.varInfo.ResponseJudge != "" {
		h.thisInfo.Data = &d
		if _, _, err = getStrByTemplate(h.varInfo.ResponseJudge, h.data, h.thisInfo); err != nil {
			return errors.New(err.Error())
		}

	} else if res.StatusCode != 200 {
		return errors.New(res.Status)
	}

	resDataBytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return err
	}

	var resData interface{} = resDataBytes
	if h.varInfo.ResponseParser != "" {
		split := strings.Split(h.varInfo.ResponseParser, "|")
		for _, s := range split {
			s = strings.TrimSpace(strings.ToLower(s))
			switch s {
			case "text":
				resData = string(resDataBytes)
			case "json":
				var _d interface{}
				if err = json.Unmarshal(resDataBytes, &_d); err != nil {
					return err
				}
				resData = &_d
			case "hex":
				resDataBytes, err = hex.DecodeString(string(resDataBytes))
				if err != nil {
					return err
				}
				resData = resDataBytes
			case "base64":
				resDataBytes, err = base64.StdEncoding.DecodeString(string(resDataBytes))
				if err != nil {
					return err
				}
				resData = resDataBytes
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

func createHttpRequestByVar(varInfo *RemoteVarInfo, data map[string]interface{}, thisInfo *ThisInfo) error {
	if !strings.HasPrefix(varInfo.Url, string(varInfo.Type)) {
		varInfo.Url = fmt.Sprintf("%s://%s", varInfo.Type, varInfo.Url)
	}

	headers := make(map[string]string)
	if err := parseOrderFieldMap(varInfo.RequestParams, data, thisInfo, func(k, v string) error {
		headers[k] = v
		return nil
	}); err != nil {
		return err
	}

	var requestBody io.Reader
	if varInfo.RequestParams != nil && len(varInfo.RequestParams.Keys()) > 0 {
		buf := &bytes.Buffer{}
		if err := parseOrderFieldMap(varInfo.RequestParams, data, thisInfo, func(k, v string) error {
			buf.WriteString(k)
			buf.WriteRune('=')
			buf.WriteString(v)
			buf.WriteRune('&')
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

	if varInfo.RequestBody != "" {
		thisInfo.Data = varInfo
		reqBody, _, err := getStrByTemplate(varInfo.RequestBody, data, thisInfo)
		if err != nil {
			return err
		}

		requestBody = strings.NewReader(reqBody)
	} else if varInfo.RequestFormData != nil && len(varInfo.RequestFormData.Keys()) > 0 {
		vals := make(url.Values)
		if err := parseOrderFieldMap(varInfo.RequestFormData, data, thisInfo, func(k, v string) error {
			vals[k] = []string{v}
			return nil
		}); err != nil {
			return err
		}
		requestBody = strings.NewReader(vals.Encode())
	} else if varInfo.RequestUploadFiles != nil {
		files := varInfo.RequestUploadFiles.Files
		datas := varInfo.RequestUploadFiles.Data

		filesLen := len(files.Keys())
		datasLen := len(datas.Keys())

		if filesLen > 0 || datasLen > 0 {
			buf := &bytes.Buffer{}
			fileWriter := multipart.NewWriter(buf)
			if filesLen > 0 {
				if err := parseOrderFieldMap(files, data, thisInfo, func(k, v string) error {
					filename := filepath.Base(v)
					file, err := fileWriter.CreateFormFile(k, filename)
					if err != nil {
						return err
					}
					if err = copyFile2Writer(v, file); err != nil {
						return err
					}
					return nil
				}); err != nil {
					return err
				}
			}

			if datasLen > 0 {
				if err := parseOrderFieldMap(datas, data, thisInfo, func(k, v string) error {
					return fileWriter.WriteField(k, v)
				}); err != nil {
					return err
				}
			}

			requestBody = buf
		}

	}

	req, err := http.NewRequest(varInfo.Method, varInfo.Url, requestBody)
	if err != nil {
		return err
	}

	httpClient := http.DefaultClient

	if varInfo.Type == SupportRemoteReqTypeHttps && varInfo.SkipHttpsVerifyCert {
		httpClient = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					InsecureSkipVerify: true,
				},
			},
		}
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

package pkg

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
)

func GetJson(Url string, response interface{}) (err error) {
	var data []byte
	resp, err := http.Get(Url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if bytes.Contains(data, []byte("errcode")) {
		var werr = &Error{}
		_ = json.Unmarshal(data, werr)
		if werr.ErrCode != 0 {
			return werr
		}
	}
	if response != nil {
		return json.Unmarshal(data, response)
	} else {
		return nil
	}
}

func PostSchema(kind cryptKind, url string, schema, response interface{}) error {
	schemaBuf := bytes.NewBuffer(nil)
	var encoder encoder
	switch kind {
	case KindJson:
		enc := json.NewEncoder(schemaBuf)
		enc.SetEscapeHTML(false)
		encoder = enc
	case KindXml:
		enc := xml.NewEncoder(schemaBuf)
		encoder = enc
	}
	err := encoder.Encode(schema)
	if err != nil {
		return err
	} else {
		return PostData(kind, url, schemaBuf.Bytes(), response)
	}
}

// 解析类型
type cryptKind int

const (
	KindJson cryptKind = iota
	KindXml
)

// json/xml 统一接口
type decoder interface {
	Decode(v interface{}) error
}

type encoder interface {
	Encode(v interface{}) error
}

func PostData(kind cryptKind, url string, data []byte, response interface{}) error {
	var contentType string
	var resDecoder func(data []byte) decoder
	switch kind {
	case KindJson:
		contentType = "application/json;charset=utf-8"
		resDecoder = func(data []byte) decoder {
			return json.NewDecoder(bytes.NewReader(data))
		}
	case KindXml:
		contentType = "application/xml;charset=utf-8"
		resDecoder = func(data []byte) decoder {
			return xml.NewDecoder(bytes.NewReader(data))
		}
	}
	resp, err := http.Post(url, contentType, bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if bytes.Contains(data, []byte("errcode")) {
		var werr = &Error{}
		resDecoder(data).Decode(werr)
		if werr.ErrCode != 0 {
			return werr
		}
	}
	if response != nil {
		return resDecoder(data).Decode(response)
	} else {
		return nil
	}
}

func UploadFile(uri string, data []byte, fieldName, fileName string, values url.Values, res interface{}) error {
	buf := &bytes.Buffer{}
	mulWriter := multipart.NewWriter(buf)
	fileWriter, err := mulWriter.CreateFormFile(fieldName, fileName)
	if err != nil {
		return err
	}
	_, err = fileWriter.Write(data)
	if err != nil {
		return err
	}
	for key, vs := range values {
		for _, value := range vs {
			partWriter, err := mulWriter.CreateFormField(key)
			if err != nil {
				return err
			} else {
				_, err = partWriter.Write([]byte(value))
				if err != nil {
					return err
				}
			}
		}
	}
	contentType := mulWriter.FormDataContentType()
	_ = mulWriter.Close()
	resp, err := http.Post(uri, contentType, buf)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	data, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if bytes.Contains(data, []byte("errcode")) {
		e := &Error{}
		_ = json.Unmarshal(data, e)
		if e.ErrCode != 0 {
			return e
		}
	}
	return json.Unmarshal(data, res)
}

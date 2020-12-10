package util

import (
	"Gbase8sCluster/entity"
	//"encoding/json"
	"errors"
	"fmt"
	"github.com/json-iterator/go"
	"github.com/kirinlabs/HttpRequest"
	"time"
)

type HttpClient struct {
	httpReq *HttpRequest.Request
}

func NewHttpClient() *HttpClient {
	req := HttpRequest.NewRequest()
	req.SetHeaders(map[string]string{
		"Content-Type": "application/json",
	})
	return &HttpClient{
		req,
	}
}

func (h *HttpClient) SetTimeout(timeout time.Duration) *HttpClient {
	h.httpReq.SetTimeout(timeout)
	return h
}

func (h *HttpClient) Get(url string) (*entity.ResponseData, error) {
	if resp, err := h.httpReq.Get(url); err != nil {
		return nil, err
	} else {
		if resp.StatusCode() != 200 {
			return nil, errors.New(fmt.Sprintf("status code %d", resp.StatusCode()))
		}
		if byteRet, err := resp.Body(); err != nil {
			return nil, err
		} else {
			var reponseData entity.ResponseData
			if err := jsoniter.Unmarshal(byteRet, &reponseData); err != nil {
				return nil, err
			}
			return &reponseData, nil
		}
	}
}

func (h *HttpClient) Post(url string, data interface{}) (*entity.ResponseData, error) {
	body, err := jsoniter.Config{EscapeHTML: false, ValidateJsonRawMessage: true}.Froze().Marshal(data)
	if err != nil {
		return nil, err
	}
	//ssss := "{\"cmd\":\"source /env.sh && onstat -g rss\"}"
	//log.Info(ssss)
	//log.Info(string(body))
	//if v, ok := data.(string); ok {
	//	log.Info(v)
	//}
	if resp, err := h.httpReq.Post(url, body); err != nil {
		return nil, err
	} else {
		if resp.StatusCode() != 200 {
			return nil, errors.New(fmt.Sprintf("status code %d", resp.StatusCode()))
		}
		if byteRet, err := resp.Body(); err != nil {
			return nil, err
		} else {
			var reponseData entity.ResponseData
			if err := jsoniter.Unmarshal(byteRet, &reponseData); err != nil {
				return nil, err
			}
			return &reponseData, nil
		}
	}
}

package utils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// 用于解析太平洋API返回的JSON结构
type IPLocation struct {
	IP     string `json:"ip"`
	Pro    string `json:"pro"`  // 省份
	City   string `json:"city"` // 城市
	Addr   string `json:"addr"` // 完整地址描述
	Region string `json:"region"`
	ISP    string `json:"isp"` // 运营商
}

func queryIPLocation(ip string) (*IPLocation, error) {
	// 构造太平洋网络IP查询API的URL[citation:5][citation:8]
	url := fmt.Sprintf("http://whois.pconline.com.cn/ipJson.jsp?ip=%s&json=true", ip)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("请求API失败: %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	var location IPLocation
	if err := json.Unmarshal(body, &location); err != nil {
		return nil, fmt.Errorf("解析JSON失败: %w", err)
	}

	return &location, nil
}

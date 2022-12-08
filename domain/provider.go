package domain

import (
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// DomainProvider 从三方信息提供商获取子域名数据
type DomainProvider struct {
	*DomainModule
	*http.Client
	*DomainProviderOption
}

type DomainProviderOption struct {
	Proxy string
}

func (dp *DomainProvider) Parse(option interface{}) {
	dp.DomainProviderOption, _ = option.(*DomainProviderOption)
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	if dp.Proxy != "" {
		uri, _ := url.Parse(dp.Proxy)
		transport.Proxy = http.ProxyURL(uri)
	}
	dp.Client = &http.Client{
		Transport: transport,
	}
}

// UseDNSDumpster 使用DNSDumpster子域名扫描结果
func (dp *DomainProvider) UseDNSDumpster() {
	api := "https://dnsdumpster.com/"
	req, _ := http.NewRequest("GET", api, nil)
	req.Header.Add("Referer", "https://dnsdumpster.com")
	res, err := dp.Client.Do(req)
	if err != nil {
		Log.Warn("DNSDumpster失败")
		return
	}
	for _, c := range res.Cookies() {
		if c.Name == "csrftoken" {
			data := fmt.Sprintf("csrfmiddlewaretoken=%s&targetip=%s&user=free", c.Value, dp.RootDomain)
			req2, _ := http.NewRequest("POST", api, strings.NewReader(data))
			dp.doRequest(req2)
		}
	}
}

// UseCertSpotter 利用证书透明度查询公开的子域名
func (dp *DomainProvider) UseCertSpotter() {
	api := fmt.Sprintf("https://api.certspotter.com/v1/issuances?domain=%s&include_subdomains=true&expand=dns_names", dp.RootDomain)
	req, _ := http.NewRequest("GET", api, nil)
	n, _ := dp.doRequest(req)
	Log.Debugf("使用CertSpotter发现了%d域名", n)
}

// UseIP138 使用IP138获取子域名
func (dp *DomainProvider) UseIP138() {
	api := fmt.Sprintf("https://site.ip138.com/%s/domain.htm", dp.RootDomain)
	req, _ := http.NewRequest("GET", api, nil)
	req.Header.Add("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.5359.95 Safari/537.36")
	n, _ := dp.doRequest(req)
	Log.Debugf("使用IP138发现了%d域名", n)
}

// UseQianxun 使用千寻在线子域名扫描结果
func (dp *DomainProvider) UseQianXun() {
	for i := 1; ; i++ {
		api := fmt.Sprintf("https://www.dnsscan.cn/dns.html?keywords=%s&page=%d", dp.RootDomain, i)
		data := fmt.Sprintf("ecmsfrom=&show=&num=&classid=0&keywords=%s", dp.RootDomain)
		req, _ := http.NewRequest("POST", api, strings.NewReader(data))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Add("Referer", "http://www.dnsscan.cn/dns.html")
		n, _ := dp.doRequest(req)
		if n == 0 {
			break
		}
	}
}

func (dp *DomainProvider) doRequest(req *http.Request) (int, *http.Response) {
	res, err := dp.Client.Do(req)
	if err != nil {
		Log.Warn(err)
		return 0, res
	}
	content, err := io.ReadAll(res.Body)
	if err != nil {
		Log.Warn(err)
		return 0, res
	}

	sr := make(SimpleDomainResult)
	pattern := strings.ReplaceAll(dp.RootDomain, ".", "\\.")
	re := regexp.MustCompile("[-\\w\\.]+" + pattern)
	all := re.FindAllString(string(content), -1)
	for _, v := range all {
		if v == "."+dp.RootDomain {
			continue
		}
		if _, ok := sr[v]; ok {
			continue
		}
		sr[v] = struct{}{}
	}
	n := len(sr)
	if n > 0 {
		dp.SubdomainResultChan <- sr
	}
	return n, res
}

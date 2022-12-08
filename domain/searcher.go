package domain

import (
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

// DomainSearcher 通过搜索引擎搜集子域名
type DomainSearcher struct {
	*DomainModule
	*http.Client
	*DomainSearcherOption
}

// DomainSearcherOption 用户配置项
type DomainSearcherOption struct {
	Headers     map[string]string
	Proxy       string
	BaiduCookie string
	BingCookie  string
	FofaEmail   string
	FofaKey     string
}

func (ds *DomainSearcher) Parse(option interface{}) {
	ds.DomainSearcherOption, _ = option.(*DomainSearcherOption)
	transport := &http.Transport{TLSClientConfig: &tls.Config{InsecureSkipVerify: true}}
	if ds.Proxy != "" {
		uri, _ := url.Parse(ds.Proxy)
		transport.Proxy = http.ProxyURL(uri)
	}
	if ds.Headers == nil {
		defaultHeaders := map[string]string{
			"User-Agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/108.0.5359.95 Safari/537.36",
			"Accept":     "*/*",
		}
		ds.Headers = defaultHeaders
	}

	ds.Client = &http.Client{
		Transport: transport,
	}
}

func (ds *DomainSearcher) UseGoogleAPI() {

}

// Use360So 使用360So搜索引擎
func (ds *DomainSearcher) Use360So() {

}

// UseQuakeAPI 使用360夸克搜索引擎的API
func (ds *DomainSearcher) UseQuakeAPI() {

}

// UseFofaAPI 使用Fofa的API搜集子域名
func (ds *DomainSearcher) UseFofaAPI() {
	qbase64 := base64.StdEncoding.EncodeToString([]byte("domain=" + ds.RootDomain))
	size := 1000
	for page := 1; ; page++ {
		api := fmt.Sprintf("https://fofa.info/api/v1/search/all?full=true&key=%v&email=%v&qbase64=%v&page=%v&size=%v", ds.FofaKey, ds.FofaEmail, qbase64, page, size)
		req, _ := http.NewRequest("GET", api, nil)
		ds.doSearch(req)
	}
}

func (ds *DomainSearcher) UseBingAPI() {
	// q: 关键词 offset: 跳过的数量 count: 每页显示数量

}

// useBingBefore 使用必应搜索时需要先获取cookie
func (ds *DomainSearcher) useBingBefore() {

}

// UseBing 使用必应搜索引擎
func (ds *DomainSearcher) UseBing() {
	if ds.BingCookie == "" {
		ds.useBingBefore()
	}

	// q: 关键词 first: 跳过的数量 count: 每页显示数量
	q := fmt.Sprintf("site:.%v", ds.RootDomain)
	count := 50
	for first := 0; ; first += count {
		api := fmt.Sprintf("https://www.bing.com/search?q=%v&first=%v&count=%v", q, first, count)
		req, _ := http.NewRequest("GET", api, nil)
		for k, v := range ds.Headers {
			req.Header.Add(k, v)
		}
		if ds.BingCookie != "" {
			req.Header.Add("Cookie", ds.BaiduCookie)
		}
		ds.doSearch(req)
	}
}

// UseBaidu 使用百度搜索引擎
func (ds *DomainSearcher) UseBaidu() {
	// wd: 关键词 pn: 跳过的数量 rn: 每页显示数量
	wd := fmt.Sprintf("site:.%v", ds.RootDomain)
	rn := 50
	cookies := []*http.Cookie{}
	for pn := 0; ; pn += rn {
		api := fmt.Sprintf("https://www.baidu.com/s?wd=%v&pn=%v&rn=%v", wd, pn, rn)
		req, _ := http.NewRequest("GET", api, nil)
		for k, v := range ds.Headers {
			req.Header.Add(k, v)
		}
		if ds.BaiduCookie != "" {
			req.Header.Add("Cookie", ds.BaiduCookie)
		}
		for _, c := range cookies {
			req.AddCookie(c)
		}
		n, res := ds.doSearch(req)
		if n == 0 {
			break
		}
		if len(res.Cookies()) > 0 {
			cookies = append(cookies, res.Cookies()...)
		}
	}
}

func (ds *DomainSearcher) doSearch(req *http.Request) (int, *http.Response) {
	res, err := ds.Client.Do(req)
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
	pattern := strings.ReplaceAll(ds.RootDomain, ".", "\\.")
	re := regexp.MustCompile("[-\\w\\.]+" + pattern)
	all := re.FindAllString(string(content), -1)
	for _, v := range all {
		if v == "."+ds.RootDomain {
			continue
		}
		if _, ok := sr[v]; ok {
			continue
		}
		sr[v] = struct{}{}
	}
	n := len(sr)
	if n > 0 {
		ds.SubdomainResultChan <- sr
	}
	return n, res
}

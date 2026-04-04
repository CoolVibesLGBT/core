package helpers

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

var UserAgents = []string{
	// Desktop - Chrome, Firefox, Edge, Safari
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_2) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Edge/120.0.0.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:121.0) Gecko/20100101 Firefox/121.0",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:120.0) Gecko/20100101 Firefox/120.0",

	// Mobile - iOS, Android
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPad; CPU OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (Linux; Android 14; Pixel 8 Pro) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 14; Samsung Galaxy S24) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 13; Xiaomi 13) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Mobile/15E148 Safari/604.1",

	// Tablets
	"Mozilla/5.0 (Linux; Android 14; Samsung Galaxy Tab S9) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/121.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Linux; Android 12; Lenovo Tab P12) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Linux; Android 11; Huawei MatePad) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/119.0.0.0 Safari/537.36",

	// Gaming consoles
	"Mozilla/5.0 (PlayStation 5 4.03) AppleWebKit/605.1.15 (KHTML, like Gecko)",
	"Mozilla/5.0 (Nintendo Switch; WebApp) AppleWebKit/601.7 (KHTML, like Gecko) Version/9.0.0.0 Safari/601.7",

	// Bots / Crawlers
	"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
	"Mozilla/5.0 (compatible; Bingbot/2.0; +http://www.bing.com/bingbot.htm)",
	"Mozilla/5.0 (compatible; YandexBot/3.0; +http://yandex.com/bots)",
	"Mozilla/5.0 (compatible; DuckDuckBot/1.0; +http://duckduckgo.com/duckduckbot.html)",
	"Mozilla/5.0 (compatible; Baiduspider/2.0; +http://www.baidu.com/search/spider.html)",
	"Mozilla/5.0 (compatible; Sogou web spider/4.0; +http://www.sogou.com/docs/help/webmasters.htm)",
	"Mozilla/5.0 (compatible; Exabot/3.0; +http://www.exabot.com/go/robot)",

	// Older / Misc browsers
	"Mozilla/5.0 (Windows NT 6.1; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.159 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/14.1.2 Safari/605.1.15",
	"Mozilla/5.0 (Windows NT 10.0; WOW64; rv:102.0) Gecko/20100101 Firefox/102.0",

	// Smart TVs
	"Mozilla/5.0 (SMART-TV; Linux; Tizen 6.0) AppleWebKit/537.36 (KHTML, like Gecko) SamsungBrowser/15.0 TV Safari/537.36",
	"Mozilla/5.0 (Linux; Android 10; SHIELD Android TV) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/92.0.4515.159 Safari/537.36",

	// Kindle / e-readers
	"Mozilla/5.0 (Linux; U; Android 14; en-US; Kindle Fire HD 10) AppleWebKit/537.36 (KHTML, like Gecko) Silk/123.4.567 like Chrome/121.0.0.0",

	// Misc / niche
	"Mozilla/5.0 (BlackBerry; U; BlackBerry 10.3; en-US) AppleWebKit/537.35+ (KHTML, like Gecko) Version/10.3.1.2576 Mobile Safari/537.35+",
	"Mozilla/5.0 (webOS/1.4.5; U; en-US) AppleWebKit/532.2 (KHTML, like Gecko) Version/1.0 Safari/532.2",
}

func ApplyBrowserHeaders(req *http.Request) {
	req.Header.Set("User-Agent", RandomUserAgent())
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	//req.Header.Set("Accept-Encoding", "gzip, deflate, br")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Upgrade-Insecure-Requests", "1")
}

func RandomUserAgent() string {
	return UserAgents[rand.Intn(len(UserAgents))]
}

type ProxyConfig struct {
	Proxies []string `json:"proxies"`
}

func LoadProxies(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg ProxyConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return cfg.Proxies, nil
}

var proxyList []string

func InitProxies(path string) error {
	p, err := LoadProxies(path)
	if err != nil {
		return err
	}
	proxyList = p
	return nil
}

func RandomProxy() (string, error) {
	if len(proxyList) == 0 {
		return "", errors.New("proxy required")
	}
	return proxyList[rand.Intn(len(proxyList))], nil
}

func CloseQuietly(c io.Closer) {
	if err := c.Close(); err != nil {
		log.Printf("close error: %v", err)
	}
}

func DownloadFile(urlAddr, filePath string) error {

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	client := &http.Client{
		Timeout:   1000 * time.Second,
		Transport: transport,
	}

	parsedURL, err := url.Parse(urlAddr)
	if err != nil {
		sleep := time.Duration(rand.Intn(15)+15) * time.Second
		time.Sleep(sleep)
		return err
	}

	req, err := http.NewRequest("GET", parsedURL.String(), nil)
	if err != nil {
		sleep := time.Duration(rand.Intn(15)+15) * time.Second
		time.Sleep(sleep)
		return err
	}

	ApplyBrowserHeaders(req)
	req.Header.Set("User-Agent", RandomUserAgent())

	resp, err := client.Do(req)
	if err != nil {
		sleep := time.Duration(rand.Intn(15)+15) * time.Second
		time.Sleep(sleep)
		return err
	}
	defer CloseQuietly(resp.Body)

	if resp.StatusCode != http.StatusOK {
		sleep := time.Duration(rand.Intn(10)+25) * time.Second
		time.Sleep(sleep)
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filePath)
	if err != nil {
		sleep := time.Duration(rand.Intn(10)+10) * time.Second
		time.Sleep(sleep)
		return err
	}
	defer CloseQuietly(out)

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		sleep := time.Duration(rand.Intn(10)+10) * time.Second
		time.Sleep(sleep)
		return err
	}

	sleep := time.Duration(rand.Intn(10)+15) * time.Second
	time.Sleep(sleep)

	return nil
}

func DownloadFileWithProxy(urlAddr, filePath string, proxyAddr string) error {
	proxyURL := func(_ *http.Request) (*url.URL, error) {
		if proxyAddr == "" {
			return nil, nil
		}
		if !strings.Contains(proxyAddr, "://") {
			proxyAddr = "http://" + proxyAddr
		}
		return url.Parse(proxyAddr)
	}

	transport := &http.Transport{
		Proxy: proxyURL,
	}

	client := &http.Client{
		Timeout:   50 * time.Second,
		Transport: transport,
	}

	req, err := http.NewRequest("GET", urlAddr, nil)
	if err != nil {
		return err
	}

	ApplyBrowserHeaders(req)
	req.Header.Set("User-Agent", RandomUserAgent())

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer CloseQuietly(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("bad status: %s", resp.Status)
	}

	out, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer CloseQuietly(out)

	_, err = io.Copy(out, resp.Body)
	return err
}

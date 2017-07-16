package invite

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type CarrierInfo struct {
	SMS string
	MMS string
}

var (
	SMS = regexp.MustCompile(`SMS.*<td.*?>(.*?)<`)
	MMS = regexp.MustCompile(`MMS.*<td.*?>(.*?)<`)
)

func addHeader(req *http.Request, hdrs map[string]string) {
	for k, v := range hdrs {
		req.Header.Add(k, v)
	}
}

type cookie struct {
	*http.Cookie
	assigned time.Time
}

func Lookup(cw *http.Cookie, countryCode, phoneNum string) CarrierInfo {
	const urlStr = `http://freecarrierlookup.com/getcarrier.php`
	v := url.Values{
		"cc":       []string{countryCode},
		"phonenum": []string{phoneNum},
	}
	req, err := http.NewRequest("POST", urlStr, strings.NewReader(v.Encode()))
	addHeader(req, map[string]string{
		`Content-Type`: "application/x-www-form-urlencoded",
		`Cookie`:       cw.String(),
	})
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	defer resp.Body.Close()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	b, _ := ioutil.ReadAll(resp.Body)
	ci := CarrierInfo{
		SMS: string(SMS.FindSubmatch(b)[1]),
		MMS: string(MMS.FindSubmatch(b)[1]),
	}
	return ci
}

type cache struct {
	size int
	sync.Mutex
	c []cookie
}

func NewCache(size int) *cache {
	if size < 1 {
		return nil
	}
	return &cache{
		size: size,
		c:    []cookie{},
	}
}

func (che *cache) Add(cke *http.Cookie) {
	che.Lock()
	defer che.Unlock()
	che.c = append(che.c, cookie{cke, time.Now()})
	sort.Slice(che.c, func(i, j int) bool {
		return che.c[i].assigned.Before(che.c[j].assigned)
	})
	if diff := len(che.c) - che.size; diff > 0 {
		che.c = che.c[diff:]
	}
}

func (che *cache) Get() *http.Cookie {
	che.Lock()
	defer che.Unlock()
	if len(che.c) < che.size {
		return nil
	}
	return che.c[len(che.c)-1].Cookie
}

func NewCookie() *http.Cookie {
	const urlRoot = `http://freecarrierlookup.com`
	resp, err := http.Get(urlRoot)
	defer resp.Body.Close()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	var cw *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "PHPSESSID" {
			cw = c
			break
		}
	}
	if cw == nil {
		fmt.Println("Did not find cookie PHPSESSID")
		os.Exit(1)
	}
	return cw
}

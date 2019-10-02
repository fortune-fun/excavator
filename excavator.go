package excavator

import (
	"crypto/sha256"
	"crypto/tls"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-xorm/xorm"
	"github.com/gocolly/colly"
	"github.com/godcong/go-trait"
)

var log = trait.NewZapFileSugar("excavator.log")
var db *xorm.Engine
var debug = false

const tmpFile = "tmp"

// Step ...
type Step string

// excavator run step status ...
const (
	StepAll       Step = "all"
	StepRadical        = "radical"
	StepCharacter      = "character"
)

// Excavator ...
type Excavator struct {
	Workspace   string `json:"workspace"`
	URL         string `json:"url"`
	HTML        string `json:"html"`
	skip        []string
	db          *xorm.Engine
	radicalType RadicalType
}

// DB ...
func (exc *Excavator) DB() *xorm.Engine {
	return exc.db
}

// SetDB ...
func (exc *Excavator) SetDB(db *xorm.Engine) {
	exc.db = db
}

type ExArgs func(exc *Excavator)

func URLArgs(url string) ExArgs {
	return func(exc *Excavator) {
		exc.URL = url
	}
}

// New ...
func New(radicalType RadicalType, args ...ExArgs) *Excavator {
	exc := &Excavator{radicalType: radicalType, Workspace: tmpFile}
	for _, arg := range args {
		arg(exc)
	}
	return exc
}

func radicalUrl(url string, radicalType RadicalType) string {
	switch radicalType {
	case RadicalTypeHanChengPinyin:
		url += HanChengPinyin
	case RadicalTypeHanChengBushou:
		url += HanChengBushou
	case RadicalTypeHanChengBihua:
	case RadicalTypeKangXiPinyin:
	case RadicalTypeKangXiBushou:
	case RadicalTypeKangXiBihua:
	}
	return url
}

// init ...
func (exc *Excavator) init() {
	if exc.db == nil {
		exc.db = InitMysql("localhost:3306", "root", "111111")
	}
}

// Run ...
func (exc *Excavator) Run() error {
	log.Info("excavator run")
	exc.init()
	//switch exc.step {
	//case StepAll:
	//case StepRadical:
	//	go exc.parseRadical(exc.radical)
	//case StepCharacter:
	//	go exc.findRadical(exc.radical)
	//	go exc.parseCharacter(exc.radical, exc.character)
	//}

	return nil
}
func (exc *Excavator) findRadical(characters chan<- *RadicalCharacter) {
	defer func() {
		characters <- nil
	}()
	i, e := exc.db.Count(RadicalCharacter{})
	if e != nil || i == 0 {
		log.Error(e)
		return
	}
	log.With("total", i).Info("total char")
	for x := int64(0); x < i; x += 500 {
		rc := new([]RadicalCharacter)
		e := exc.db.Limit(500, int(x)).Find(rc)
		if e != nil {
			log.Error(e)
			continue
		}
		for i := range *rc {
			characters <- &(*rc)[i]
		}
	}
}

func (exc *Excavator) parseRadical(characters chan<- *RadicalCharacter) {
	defer func() {
		characters <- nil
	}()
	c := colly.NewCollector()
	c.OnHTML("a[href][data-action]", func(element *colly.HTMLElement) {
		da := element.Attr("data-action")
		log.With("value", da).Info("data action")
		if da == "" {
			return
		}
		r, e := exc.parseAJAX(exc.URL, strings.NewReader(fmt.Sprintf("wd=%s", da)))
		if e != nil {
			return
		}
		for _, tmp := range *(*[]RadicalUnion)(r) {
			for i := range tmp.RadicalCharacterArray {
				rc := tmp.RadicalCharacterArray[i]
				e := exc.saveRadicalCharacter(&tmp.RadicalCharacterArray[i])
				if e != nil {
					log.Error(e)
					continue
				}
				characters <- &rc
			}
		}
		log.With("value", r).Info("radical")
	})
	c.OnResponse(func(response *colly.Response) {
		log.Info(string(response.Body))
	})
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})
	e := c.Visit(exc.URL)
	if e != nil {
		log.Error(e)
	}
	return
}

func (exc *Excavator) parseAJAX(url string, body io.Reader) (r *Radical, e error) {
	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: tr}
	//body := strings.NewReader(`wd=%E4%B9%99`)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}
	//req.Header = exc.Header()

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return UnmarshalRadical(bytes)
}

//ParseDocument get the url result body
func (exc *Excavator) parseDocument(url string) (doc *goquery.Document, e error) {
	var reader io.Reader
	hash := SHA256(url)
	log.Infof("hash:%s,url:%s", hash, url)
	if !exc.IsExist(hash) {
		// Request the HTML page.
		res, err := http.Get(url)
		if err != nil {
			return nil, err
		}
		defer res.Body.Close()
		if res.StatusCode != 200 {
			return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
		}
		reader = res.Body
		file, e := os.OpenFile(exc.getFilePath(hash), os.O_RDWR|os.O_CREATE|os.O_SYNC, os.ModePerm)
		if e != nil {
			return nil, e

		}
		written, e := io.Copy(file, reader)
		if e != nil {
			return nil, e
		}
		log.Infof("read %s | %d ", hash, written)
		_ = file.Close()
	}
	reader, e = os.Open(exc.getFilePath(hash))
	if e != nil {
		return nil, e
	}
	// Load the HTML document
	return goquery.NewDocumentFromReader(reader)
}

// IsExist ...
func (exc *Excavator) IsExist(name string) bool {
	_, e := os.Open(name)
	return e == nil || os.IsExist(e)
}

// GetPath ...
func (exc *Excavator) getFilePath(s string) string {
	if exc.Workspace == "" {
		exc.Workspace, _ = os.Getwd()
	}
	log.With("workspace", exc.Workspace, "temp", tmpFile, "file", s).Info("file path")
	return filepath.Join(exc.Workspace, tmpFile, s)
}

/*URL 拼接地址 */
func URL(prefix string, uris ...string) string {
	end := len(prefix)
	if end > 1 && prefix[end-1] == '/' {
		prefix = prefix[:end-1]
	}

	var url = []string{prefix}
	for _, v := range uris {
		url = append(url, TrimSlash(v))
	}
	return strings.Join(url, "/")
}

// TrimSlash ...
func TrimSlash(s string) string {
	if size := len(s); size > 1 {
		if s[size-1] == '/' {
			s = s[:size-1]
		}
		if s[0] == '/' {
			s = s[1:]
		}
	}
	return s
}

func (exc *Excavator) parseCharacter(characters <-chan *RadicalCharacter, char chan<- *Character) {
	defer func() {
		char <- nil
	}()
	c := colly.NewCollector()
	var ch *Character
	c.OnRequest(func(r *colly.Request) {
		fmt.Println("Visiting", r.URL)
	})
	c.OnResponse(func(response *colly.Response) {
		log.Info(response.StatusCode)
	})
	c.OnHTML(`div[class=info] > p[class=mui-ellipsis]`, func(element *colly.HTMLElement) {
		e := parseKangXiCharacter(element, ch)
		log.Infof("%+v", ch)
		if e != nil {
			log.Error(e)
		}
	})
	c.OnHTML(`div > ul.hanyu-cha-info.mui-clearfix`, func(element *colly.HTMLElement) {
		e := parseDictInformation(element, ch)
		log.Infof("%+v", ch)
		if e != nil {
			log.Error(e)
		}
	})
	c.OnHTML(`div > ul.hanyu-cha-ul`, func(element *colly.HTMLElement) {
		e := parseDictComment(element, ch)
		if e != nil {
			log.Error(e)
		}
	})
	c.OnScraped(func(response *colly.Response) {

	})
	for {
		select {
		case cr := <-characters:
			if cr == nil {
				goto END
			}
			ch = new(Character)
			ch.Ch = cr.Zi
			if ch.Radical == "" {
				ch.Radical = cr.BuShou
			}
			e := c.Visit(URL(exc.URL, cr.URL))
			if e != nil {
				log.With("radical", cr.BuShou).Error(e)
				continue
			}
			session := exc.db.NewSession()
			e = ch.InsertIfNotExist(session)
			//_, e = exc.db.InsertOne(ch)
			if e != nil {
				log.With("radical", cr.BuShou).Error(e)
				session.Close()
				continue
			}
			session.Close()
			char <- ch
		}
	}
END:
}

func (exc *Excavator) saveRadicalCharacter(characters *RadicalCharacter) (e error) {
	i, e := exc.db.Where("url = ?", characters.URL).Count(RadicalCharacter{})
	if e != nil {
		return e
	}
	if i == 0 {
		_, e = exc.db.InsertOne(characters)
	}
	log.With("url", characters.URL).Info("exist")
	return
}

// SHA256 ...
func SHA256(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(s)))
}

func bushouHeader() http.Header {
	header := make(http.Header)
	header.Set("Cookie", "hy_so_4=%255B%257B%2522zi%2522%253A%2522%25E8%2592%258B%2522%252C%2522url%2522%253A%252234%252FKOKORNKOCQXVILXVB%252F%2522%252C%2522py%2522%253A%2522ji%25C7%258Eng%252C%2522%252C%2522bushou%2522%253A%2522%25E8%2589%25B9%2522%252C%2522num%2522%253A%252217%2522%257D%255D; ASP.NET_SessionId=zilmx52mwtr3xsq5i212pd5a; UM_distinctid=16c2efb5e9e134-0cfc801ee6ae06-353166-1fa400-16c2efb5e9f3c8; CNZZDATA1267010321=1299014713-1564151968-%7C1564151968; Hm_lvt_cd7ed86134e0e3138a7cf1994e6966c8=1564156322; Hm_lpvt_cd7ed86134e0e3138a7cf1994e6966c8=1564156322")
	header.Set("Origin", "http://hy.httpcn.com")
	header.Set("Accept-Encoding", "gzip, deflate")
	header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,ja;q=0.7,zh-TW;q=0.6")
	header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.142 Mobile Safari/537.36")
	header.Set("Content-Type", "application/x-www-form-urlencoded")
	header.Set("Accept", "application/json")
	header.Set("Referer", "http://hy.httpcn.com/bushou/kangxi/")
	header.Set("X-Requested-With", "XMLHttpRequest")
	header.Set("Connection", "keep-alive")
	return header
}

func pinyinHeader() http.Header {
	header := make(http.Header)
	header.Set("Cookie", "hy_so_4=%255B%257B%2522zi%2522%253A%2522%25E8%2592%258B%2522%252C%2522url%2522%253A%252234%252FKOKORNKOCQXVILXVB%252F%2522%252C%2522py%2522%253A%2522ji%25C7%258Eng%252C%2522%252C%2522bushou%2522%253A%2522%25E8%2589%25B9%2522%252C%2522num%2522%253A%252217%2522%257D%255D; ASP.NET_SessionId=zilmx52mwtr3xsq5i212pd5a; UM_distinctid=16c2efb5e9e134-0cfc801ee6ae06-353166-1fa400-16c2efb5e9f3c8; CNZZDATA1267010321=1299014713-1564151968-%7C1564151968; Hm_lvt_cd7ed86134e0e3138a7cf1994e6966c8=1564156322; Hm_lpvt_cd7ed86134e0e3138a7cf1994e6966c8=1564156322")
	header.Set("Origin", "http://hy.httpcn.com")
	header.Set("Accept-Encoding", "gzip, deflate")
	header.Set("Accept-Language", "zh-CN,zh;q=0.9,en;q=0.8,ja;q=0.7,zh-TW;q=0.6")
	header.Set("User-Agent", "Mozilla/5.0 (Linux; Android 6.0; Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/75.0.3770.142 Mobile Safari/537.36")
	header.Set("Content-Type", "application/x-www-form-urlencoded")
	header.Set("Accept", "application/json")
	header.Set("Referer", "http://hy.httpcn.com/pinyin/kangxi/")
	header.Set("X-Requested-With", "XMLHttpRequest")
	header.Set("Connection", "keep-alive")
	return header
}

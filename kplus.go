package kplus

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

const dateFormat = "20060102"
const priceUrl = "https://secure6216m.sakura.ne.jp:9802/csvex/webdav/kabu.plus/csv/japan-all-stock-prices/daily/japan-all-stock-prices_%s.csv"

type KPlus struct {
	user string
	pass string
}

func New(u, p string) *KPlus {
	return &KPlus{
		user: u,
		pass: p,
	}
}

func (k *KPlus) GetPrices(t time.Time) ([]byte, error) {
	b, err := get(k.user, k.pass, t)
	if err != nil {
		return nil, err
	}
	defer b.Close()

	return ioutil.ReadAll(transform.NewReader(b, japanese.ShiftJIS.NewDecoder()))
}

func (k *KPlus) GetPricesToday() ([]byte, error) {
	return k.GetPrices(time.Now())
}

func (k *KPlus) GetKdbPrices(t time.Time) ([]byte, error) {
	b, err := get(k.user, k.pass, t)
	if err != nil {
		return nil, err
	}
	defer b.Close()

	r := csv.NewReader(transform.NewReader(b, japanese.ShiftJIS.NewDecoder()))

	buf := new(bytes.Buffer)
	w := csv.NewWriter(buf)

	first := true
	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if first {
			first = false
			_ = w.Write([]string{"コード", "銘柄名", "市場", "始値", "高値", "安値", "終値", "出来高", "売買代金"})
			continue
		}
		if err := w.Write(convertToKdb(record)); err != nil {
			return nil, err
		}
		w.Flush()
	}
	return buf.Bytes(), nil
}

func get(user, pass string, t time.Time) (io.ReadCloser, error) {
	url := fmt.Sprintf(priceUrl, t.Format(dateFormat))
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(user, pass)

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		defer resp.Body.Close()
		return nil, fmt.Errorf(resp.Status)
	}

	return resp.Body, nil
}

func convertToKdb(s []string) []string {
	return []string{
		s[0] + "-T", // コード
		s[1],        // 銘柄
		replaceKdbMarket(s[2]), // 市場
		s[9],               // 始値
		s[10],              // 高値
		s[11],              // 安値
		s[5],               // 終値
		s[12],              // 出来高
		thousandYen(s[13]), // 売買代金
	}
}

func replaceKdbMarket(s string) string {
	r := strings.NewReplacer("一部", "1部", "二部", "2部", "JQG", "JQグロース", "JQS", "JQスタンダード", "福証QB", "福証Q-Board", "アンビシャス", "アンビ")
	return r.Replace(s)
}

func thousandYen(s string) string {
	if s == "-" {
		return s
	}
	return s + "000"
}

package providers

import (
	"encoding/json"
	"fmt"
	"strings"
	"github.com/bobesa/go-domain-util/domainutil"
)

type urlscanProvider struct {
	*Config
}

type urlscanResult struct {
	HasNext    bool `json:"has_next"`
	ActualSize int  `json:"actual_size"`
	URLList    []struct {
		Domain   string `json:"domain"`
		URL      string `json:"url"`
		Hostname string `json:"hostname"`
		HTTPCode int    `json:"httpcode"`
		PageNum  int    `json:"page_num"`
		FullSize int    `json:"full_size"`
		Paged    bool   `json:"paged"`
	} `json:"url_list"`
}

const urlscanResultsLimit = 200

func NewurlscanProvider(config *Config) Provider {
	return &urlscanProvider{Config: config}
}

func (o *urlscanProvider) formatURL(domain string, page int) string {
	if !domainutil.HasSubdomain(domain) {
		return fmt.Sprintf("https://urlscan.io.com/api/v1/indicators/domain/%s/url_list?limit=%d&page=%d",
			domain, urlscanResultsLimit, page,
		)
	} else if domainutil.HasSubdomain(domain) && o.IncludeSubdomains {
		return fmt.Sprintf("https://urlscan.io/api/v1/indicators/domain/%s/url_list?limit=%d&page=%d",
			domainutil.Domain(domain), urlscanResultsLimit, page,
		)
	} else {
		return fmt.Sprintf("https://urlscan.io/api/v1/indicators/hostname/%s/url_list?limit=%d&page=%d",
			domain, urlscanResultsLimit, page,
		)
	}
}

func (o *urlscanProvider) Fetch(domain string, results chan<- string) error {
	for page := 0; ; page++ {
		resp, err := o.MakeRequest(o.formatURL(domain, page))
		if err != nil {
			return fmt.Errorf("failed to fetch urlscan results page %d: %s", page, err)
		}

		var result urlscanResult
		if err = json.NewDecoder(resp.Body).Decode(&result); err != nil {
			_ = resp.Body.Close()
			return fmt.Errorf("failed to decode urlscan results for page %d: %s", page, err)
		}

		_ = resp.Body.Close()

		for _, entry := range result.URLList {
			if o.IncludeSubdomains {
				if !domainutil.HasSubdomain(domain) {
					results <- entry.URL
				} else {
					if strings.Contains(strings.ToLower(entry.Hostname), strings.ToLower(domain)) {
						results <- entry.URL
					}
				}
			} else {
				if strings.EqualFold(domain, entry.Hostname) {
					results <- entry.URL
				}
			}
		}

		if !result.HasNext {
			break
		}
	}

	return nil
}

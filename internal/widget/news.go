package widget

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"

	"github.com/karansnarula/personal-dashboard-api/internal/domain"
)

const (
	providerNewsAPI   = "NewsAPI"
	DefaultNewsAPIURL = "https://newsapi.org"
	defaultNewsLimit  = 5
)

type Article struct {
	Headline    string `json:"headline"`
	Source      string `json:"source"`
	URL         string `json:"url"`
	PublishedAt string `json:"published_at"`
}

type NewsData struct {
	Keyword  string    `json:"keyword"`
	Articles []Article `json:"articles"`
}

type NewsClient struct {
	hc      *http.Client
	apiKey  string
	baseURL string
}

func NewNewsClient(hc *http.Client, apiKey, baseURL string) *NewsClient {
	return &NewsClient{hc: hc, apiKey: apiKey, baseURL: baseURL}
}

func (c *NewsClient) Fetch(ctx context.Context, raw json.RawMessage) (any, error) {
	if c.apiKey == "" {
		return nil, notConfigured(providerNewsAPI)
	}
	cfg, err := parseConfig[domain.NewsConfig](raw)
	if err != nil {
		return nil, err
	}
	limit := cfg.Limit
	if limit <= 0 {
		limit = defaultNewsLimit
	}

	q := url.Values{}
	q.Set("q", cfg.Keyword)
	q.Set("pageSize", strconv.Itoa(limit))
	q.Set("sortBy", "publishedAt")
	q.Set("language", "en")

	var resp struct {
		Articles []struct {
			Title  string `json:"title"`
			URL    string `json:"url"`
			Source struct {
				Name string `json:"name"`
			} `json:"source"`
			PublishedAt string `json:"publishedAt"`
		} `json:"articles"`
	}
	headers := map[string]string{"X-Api-Key": c.apiKey}
	if err := getJSON(ctx, c.hc, providerNewsAPI, c.baseURL+"/v2/everything?"+q.Encode(), headers, &resp); err != nil {
		return nil, err
	}

	data := NewsData{Keyword: cfg.Keyword, Articles: make([]Article, 0, len(resp.Articles))}
	for _, a := range resp.Articles {
		data.Articles = append(data.Articles, Article{
			Headline:    a.Title,
			Source:      a.Source.Name,
			URL:         a.URL,
			PublishedAt: a.PublishedAt,
		})
	}
	return data, nil
}

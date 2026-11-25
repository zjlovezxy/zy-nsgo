package cms

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"sync"
	"time"
)

const defaultUserAgent = "Mozilla/5.0 (compatible; zy-nsgo/1.0)"

// Client fetches and parses T1 XML and T2 JSON Apple CMS APIs.
type Client struct {
	httpClient *http.Client
	userAgent  string
}

// NewClient constructs a CMS client with a 15-second request timeout.
func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: 15 * time.Second},
		userAgent:  defaultUserAgent,
	}
}

// Categories fetches the site category list (?ac=list).
func (c *Client) Categories(ctx context.Context, site Site) ([]Category, error) {
	page, _, err := c.fetch(ctx, site.API, url.Values{"ac": {"list"}})
	if err != nil {
		return nil, err
	}
	return page.Categories, nil
}

// List fetches a paginated video list (?ac=videolist&t=...&pg=...).
func (c *Client) List(ctx context.Context, site Site, typeID string, pageNum int) (*PageResult, error) {
	q := url.Values{"ac": {"videolist"}}
	if typeID != "" {
		q.Set("t", typeID)
	}
	if pageNum > 0 {
		q.Set("pg", strconv.Itoa(pageNum))
	}
	page, _, err := c.fetch(ctx, site.API, q)
	return page, err
}

// Detail fetches full metadata and parsed play sources for a single video (?ac=detail&ids=...).
func (c *Client) Detail(ctx context.Context, site Site, vodID string) (*VodDetail, error) {
	q := url.Values{
		"ac":  {"detail"},
		"ids": {vodID},
	}
	_, details, err := c.fetch(ctx, site.API, q)
	if err != nil {
		return nil, err
	}
	if len(details) == 0 {
		return nil, fmt.Errorf("video %q not found on site %q", vodID, site.Key)
	}
	return &details[0], nil
}

// Search queries a single site by keyword (?ac=detail&wd=...&pg=...).
func (c *Client) Search(ctx context.Context, site Site, keyword string, pageNum int) (*PageResult, error) {
	q := url.Values{
		"ac": {"detail"},
		"wd": {keyword},
	}
	if pageNum > 0 {
		q.Set("pg", strconv.Itoa(pageNum))
	}
	page, _, err := c.fetch(ctx, site.API, q)
	return page, err
}

// AggregateSearch concurrently searches all active sites and isolates per-site errors.
func (c *Client) AggregateSearch(ctx context.Context, sites []Site, keyword string) []SearchResult {
	active := make([]Site, 0, len(sites))
	for _, s := range sites {
		if s.Active {
			active = append(active, s)
		}
	}
	results := make([]SearchResult, len(active))
	var wg sync.WaitGroup
	for i, s := range active {
		wg.Add(1)
		go func(idx int, site Site) {
			defer wg.Done()
			siteCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()
			page, err := c.Search(siteCtx, site, keyword, 1)
			res := SearchResult{Site: site, Err: err}
			if page != nil {
				res.Items = page.List
			}
			results[idx] = res
		}(i, s)
	}
	wg.Wait()
	return results
}

func (c *Client) fetch(ctx context.Context, rawAPI string, extra url.Values) (*PageResult, []VodDetail, error) {
	u, err := url.Parse(rawAPI)
	if err != nil {
		return nil, nil, fmt.Errorf("invalid site API URL %q: %w", rawAPI, err)
	}
	q := u.Query()
	for k, vals := range extra {
		for _, v := range vals {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, nil, fmt.Errorf("unexpected HTTP %d from %s", resp.StatusCode, u.String())
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return nil, nil, err
	}
	return ParseResponse(body)
}

package cms

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestParsePlaySources(t *testing.T) {
	sources := ParsePlaySources("bfzym3u8$$$bfzydown", "第1集$https://a.com/1.m3u8#第2集$https://a.com/2.m3u8$$$https://a.com/full.mp4")
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}
	if sources[0].Name != "bfzym3u8" || len(sources[0].Episodes) != 2 {
		t.Fatalf("unexpected source[0]: %+v", sources[0])
	}
	if sources[0].Episodes[1].Name != "第2集" || sources[0].Episodes[1].URL != "https://a.com/2.m3u8" {
		t.Fatalf("unexpected episode[1]: %+v", sources[0].Episodes[1])
	}
	if sources[1].Name != "bfzydown" || sources[1].Episodes[0].Name != "第1集" || sources[1].Episodes[0].URL != "https://a.com/full.mp4" {
		t.Fatalf("unexpected source[1]: %+v", sources[1])
	}
}

func TestParseT1XML(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<rss version="5.1">
  <class>
    <ty id="1">电影</ty>
    <ty id="2">连续剧</ty>
  </class>
  <list page="2" pagecount="5" pagesize="20" recordcount="95">
    <video>
      <last>2026-11-25 12:00:00</last>
      <id>101</id>
      <tid>1</tid>
      <name><![CDATA[测试电影]]></name>
      <type>动作片</type>
      <pic>https://img.example.com/101.jpg</pic>
      <year>2026</year>
      <area>中国大陆</area>
      <note>HD国语</note>
      <actor>张三</actor>
      <director>李四</director>
      <des><![CDATA[简介内容]]></des>
      <dl>
        <dd flag="qzm3u8"><![CDATA[正片$https://cdn.example.com/101.m3u8]]></dd>
      </dl>
    </video>
  </list>
</rss>`

	page, details, err := ParseResponse([]byte(xmlData))
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}
	if page.Page != 2 || page.PageCount != 5 || page.Total != 95 {
		t.Fatalf("unexpected pagination: %+v", page)
	}
	if len(page.Categories) != 2 || page.Categories[1].Name != "连续剧" {
		t.Fatalf("unexpected categories: %+v", page.Categories)
	}
	if len(details) != 1 || details[0].Name != "测试电影" || len(details[0].Sources) != 1 {
		t.Fatalf("unexpected details: %+v", details)
	}
	if details[0].Sources[0].Episodes[0].URL != "https://cdn.example.com/101.m3u8" {
		t.Fatalf("unexpected play url: %+v", details[0].Sources[0].Episodes[0])
	}
}

func TestParseT2JSONMixedTypes(t *testing.T) {
	jsonData := `{
		"code": 1,
		"page": "1",
		"pagecount": 3,
		"total": "42",
		"class": [{"type_id": 6, "type_name": "动作片"}],
		"list": [{
			"vod_id": 88,
			"vod_name": "JSON测试片",
			"type_id": "6",
			"type_name": "动作片",
			"vod_pic": "https://img.example.com/88.jpg",
			"vod_remarks": "更新至02集",
			"vod_year": 2025,
			"vod_play_from": "ffm3u8",
			"vod_play_url": "第01集$https://cdn.example.com/1.m3u8#第02集$https://cdn.example.com/2.m3u8"
		}]
	}`

	page, details, err := ParseResponse([]byte(jsonData))
	if err != nil {
		t.Fatalf("ParseResponse failed: %v", err)
	}
	if page.Page != 1 || page.PageCount != 3 || page.Total != 42 {
		t.Fatalf("unexpected pagination: %+v", page)
	}
	if len(page.Categories) != 1 || page.Categories[0].ID != "6" {
		t.Fatalf("unexpected categories: %+v", page.Categories)
	}
	if len(details) != 1 || details[0].ID != "88" || details[0].Year != "2025" {
		t.Fatalf("unexpected details: %+v", details)
	}
	if len(details[0].Sources) != 1 || len(details[0].Sources[0].Episodes) != 2 {
		t.Fatalf("unexpected play sources: %+v", details[0].Sources)
	}
}

func TestClientAggregateSearch(t *testing.T) {
	srvOK := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("wd") != "流浪" {
			t.Fatalf("unexpected keyword: %q", r.URL.Query().Get("wd"))
		}
		_, _ = w.Write([]byte(`{"code":1,"page":1,"pagecount":1,"total":1,"list":[{"vod_id":9,"vod_name":"流浪地球"}]}`))
	}))
	defer srvOK.Close()

	srvFail := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srvFail.Close()

	sites := []Site{
		{Key: "ok", Name: "OK站", API: srvOK.URL, Active: true},
		{Key: "fail", Name: "故障站", API: srvFail.URL, Active: true},
		{Key: "off", Name: "停用站", API: srvOK.URL, Active: false},
	}

	c := NewClient()
	results := c.AggregateSearch(context.Background(), sites, "流浪")
	if len(results) != 2 {
		t.Fatalf("expected 2 active site results, got %d", len(results))
	}
	if results[0].Err != nil || len(results[0].Items) != 1 || results[0].Items[0].Name != "流浪地球" {
		t.Fatalf("unexpected first result: %+v", results[0])
	}
	if results[1].Err == nil {
		t.Fatalf("expected error from failing site")
	}
}

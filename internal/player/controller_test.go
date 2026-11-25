package player

import (
	"strings"
	"testing"
	"time"

	"zy-nsgo/internal/proxy"
	"zy-nsgo/internal/store"
)

func TestResolvePlayURL(t *testing.T) {
	px := proxy.NewServer(true)
	if err := px.Start(); err != nil {
		t.Fatalf("proxy Start failed: %v", err)
	}
	defer px.Close()

	direct := ResolvePlayURL(PlayRequest{
		URL:      "https://cdn.example.com/movie.m3u8",
		UseProxy: false,
	}, px)
	if direct != "https://cdn.example.com/movie.m3u8" {
		t.Fatalf("expected direct URL, got %q", direct)
	}

	proxiedHLS := ResolvePlayURL(PlayRequest{
		URL:      "https://cdn.example.com/movie.m3u8",
		UseProxy: true,
	}, px)
	if !strings.HasPrefix(proxiedHLS, px.BaseURL()+"/playlist.m3u8?") {
		t.Fatalf("expected proxied m3u8 URL, got %q", proxiedHLS)
	}

	directMP4 := ResolvePlayURL(PlayRequest{
		URL:      "https://cdn.example.com/movie.mp4",
		UseProxy: true,
	}, px)
	if directMP4 != "https://cdn.example.com/movie.mp4" {
		t.Fatalf("expected direct MP4 when no custom headers, got %q", directMP4)
	}

	proxiedMP4WithHeaders := ResolvePlayURL(PlayRequest{
		URL:      "https://cdn.example.com/movie.mp4",
		Headers:  map[string]string{"Referer": "https://ref.example.com"},
		UseProxy: true,
	}, px)
	if !strings.HasPrefix(proxiedMP4WithHeaders, px.BaseURL()+"/segment.ts?") {
		t.Fatalf("expected proxied seg URL for MP4 with headers, got %q", proxiedMP4WithHeaders)
	}
}

func TestComputeResumePosition(t *testing.T) {
	epURL := "https://cdn.example.com/1.m3u8"
	if got := computeResumePosition(&store.HistoryRecord{EpisodeURL: epURL, PositionMS: 2000, DurationMS: 60000}, epURL); got != 0 {
		t.Fatalf("expected 0 for <=3s position, got %v", got)
	}
	if got := computeResumePosition(&store.HistoryRecord{EpisodeURL: epURL, PositionMS: 57000, DurationMS: 60000}, epURL); got != 0 {
		t.Fatalf("expected 0 near end of episode, got %v", got)
	}
	if got := computeResumePosition(&store.HistoryRecord{EpisodeURL: "https://cdn.example.com/2.m3u8", PositionMS: 20000, DurationMS: 60000}, epURL); got != 0 {
		t.Fatalf("expected 0 for different episode URL, got %v", got)
	}
	if got := computeResumePosition(&store.HistoryRecord{EpisodeURL: epURL, PositionMS: 25000, DurationMS: 60000}, epURL); got != 25*time.Second {
		t.Fatalf("expected 25s, got %v", got)
	}
}

func TestClampSeekTarget(t *testing.T) {
	if got := clampSeekTarget(5*time.Second, 60*time.Second, -10*time.Second); got != 0 {
		t.Fatalf("expected clamp to 0, got %v", got)
	}
	if got := clampSeekTarget(55*time.Second, 60*time.Second, 10*time.Second); got != 60*time.Second {
		t.Fatalf("expected clamp to duration, got %v", got)
	}
	if got := clampSeekTarget(20*time.Second, 60*time.Second, 10*time.Second); got != 30*time.Second {
		t.Fatalf("expected 30s, got %v", got)
	}
}

func TestParseSavedVolume(t *testing.T) {
	if got := parseSavedVolume(""); got != 1.0 {
		t.Fatalf("expected 1.0 for empty, got %v", got)
	}
	if got := parseSavedVolume("0.45"); got != 0.45 {
		t.Fatalf("expected 0.45, got %v", got)
	}
	if got := parseSavedVolume("1.5"); got != 1.0 {
		t.Fatalf("expected clamp to 1.0, got %v", got)
	}
	if got := parseSavedVolume("-0.2"); got != 1.0 {
		t.Fatalf("expected fallback 1.0 for negative, got %v", got)
	}
}

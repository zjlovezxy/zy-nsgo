package proxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCleanM3U8AdRemovalAndRewriting(t *testing.T) {
	playlist := `#EXTM3U
#EXT-X-VERSION:3
#EXT-X-TARGETDURATION:10
#EXT-X-KEY:METHOD=AES-128,URI="enc.key"
#EXTINF:10.0,
seg1.ts
#EXTINF:10.0,
seg2.ts
#EXT-X-DISCONTINUITY
#EXTINF:5.0,
https://ad.evil.com/ad1.ts
#EXTINF:5.0,
https://ad.evil.com/ad2.ts
#EXT-X-DISCONTINUITY
#EXTINF:10.0,
seg3.ts
#EXT-X-ENDLIST`

	cleaned := CleanM3U8(playlist, "https://video.example.com/hls/index.m3u8", "http://127.0.0.1:9999", map[string]string{"Referer": "https://site.example.com"}, true)
	if strings.Contains(cleaned, "ad.evil.com") {
		t.Fatalf("expected ad segments to be stripped:\n%s", cleaned)
	}
	if !strings.Contains(cleaned, "http://127.0.0.1:9999/segment.ts?") {
		t.Fatalf("expected rewritten segment URLs:\n%s", cleaned)
	}
	if !strings.Contains(cleaned, `URI="http://127.0.0.1:9999/segment.ts?`) {
		t.Fatalf("expected rewritten EXT-X-KEY URI:\n%s", cleaned)
	}
	if !strings.Contains(cleaned, "seg1.ts") || !strings.Contains(cleaned, "seg3.ts") {
		t.Fatalf("expected main segments to be preserved:\n%s", cleaned)
	}
}

func TestStripFakeImageHeader(t *testing.T) {
	pngHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A, 0x01, 0x02, 0x03}
	tsPayload := make([]byte, 188*3)
	tsPayload[0] = 0x47
	tsPayload[188] = 0x47
	tsPayload[376] = 0x47
	combined := append(pngHeader, tsPayload...)

	stripped := StripFakeImageHeader(combined)
	if !bytes.Equal(stripped, tsPayload) {
		t.Fatalf("expected stripped TS payload of len %d, got len %d", len(tsPayload), len(stripped))
	}
}

func TestServerEndToEnd(t *testing.T) {
	pngFakeTS := append([]byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}, make([]byte, 188*2)...)
	pngFakeTS[8] = 0x47
	pngFakeTS[8+188] = 0x47

	var upstream *httptest.Server
	upstream = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Referer") != "https://ref.example.com" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		switch r.URL.Path {
		case "/index.m3u8":
			w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
			_, _ = io.WriteString(w, "#EXTM3U\n#EXTINF:5.0,\n"+upstream.URL+"/part.png\n#EXT-X-ENDLIST")
		case "/part.png":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write(pngFakeTS)
		default:
			http.NotFound(w, r)
		}
	}))
	defer upstream.Close()

	srv := NewServer(true)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start failed: %v", err)
	}
	defer srv.Close()

	wrapped := srv.WrapURL(upstream.URL+"/index.m3u8", map[string]string{"Referer": "https://ref.example.com"})
	resp, err := http.Get(wrapped)
	if err != nil {
		t.Fatalf("GET wrapped m3u8 failed: %v", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != http.StatusOK || !strings.Contains(string(body), srv.BaseURL()+"/segment.ts?") {
		t.Fatalf("unexpected m3u8 response (%d): %s", resp.StatusCode, string(body))
	}

	// Extract proxied segment URL and fetch it.
	var segURL string
	for _, line := range strings.Split(string(body), "\n") {
		if strings.HasPrefix(line, srv.BaseURL()) {
			segURL = strings.TrimSpace(line)
			break
		}
	}
	segResp, err := http.Get(segURL)
	if err != nil {
		t.Fatalf("GET proxied segment failed: %v", err)
	}
	segData, _ := io.ReadAll(segResp.Body)
	segResp.Body.Close()
	if segResp.Header.Get("Content-Type") != "video/mp2t" || len(segData) != 188*2 || segData[0] != 0x47 {
		t.Fatalf("unexpected proxied segment: ct=%q len=%d first=%x", segResp.Header.Get("Content-Type"), len(segData), segData[0])
	}
}

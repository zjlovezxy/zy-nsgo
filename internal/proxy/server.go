package proxy

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Server is a lightweight local HTTP proxy for m3u8 ad-stripping,
// fake-image TS header removal, and upstream header injection.
type Server struct {
	httpClient *http.Client
	filterAds  bool

	mu       sync.RWMutex
	listener net.Listener
	srv      *http.Server
	baseURL  string
}

// NewServer creates a local stream proxy server.
func NewServer(filterAds bool) *Server {
	return &Server{
		httpClient: &http.Client{Timeout: 20 * time.Second},
		filterAds:  filterAds,
	}
}

// Start binds an ephemeral port on 127.0.0.1 and begins serving requests.
func (s *Server) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		return nil
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("listen local proxy: %w", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/playlist.m3u8", s.handleM3U8)
	mux.HandleFunc("/m3u8", s.handleM3U8)
	mux.HandleFunc("/segment.ts", s.handleSegment)
	mux.HandleFunc("/seg", s.handleSegment)

	srv := &http.Server{Handler: mux}
	s.listener = ln
	s.srv = srv
	s.baseURL = "http://" + ln.Addr().String()

	go func() {
		_ = srv.Serve(ln)
	}()
	return nil
}

// Close shuts down the local proxy server.
func (s *Server) Close() error {
	s.mu.Lock()
	srv := s.srv
	s.srv = nil
	s.listener = nil
	s.baseURL = ""
	s.mu.Unlock()
	if srv != nil {
		return srv.Close()
	}
	return nil
}

// BaseURL returns the listening address (e.g. "http://127.0.0.1:12345").
func (s *Server) BaseURL() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.baseURL
}

// WrapURL rewrites an upstream media URL to flow through this proxy.
func (s *Server) WrapURL(rawURL string, headers map[string]string) string {
	base := s.BaseURL()
	if base == "" {
		return rawURL
	}
	endpoint := "/segment.ts"
	if isM3U8URL(rawURL) {
		endpoint = "/playlist.m3u8"
	}
	return buildProxyURL(base, endpoint, rawURL, encodeHeaders(headers))
}

func (s *Server) handleM3U8(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("url")
	if target == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}
	headers := decodeHeaders(r.URL.Query().Get("h"))
	resp, err := s.doUpstream(r, target, headers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Error(w, fmt.Sprintf("upstream status %d", resp.StatusCode), http.StatusBadGateway)
		return
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}

	cleaned := CleanM3U8(string(body), resp.Request.URL.String(), s.BaseURL(), headers, s.filterAds)
	w.Header().Set("Content-Type", "application/vnd.apple.mpegurl")
	_, _ = io.WriteString(w, cleaned)
}

func (s *Server) handleSegment(w http.ResponseWriter, r *http.Request) {
	target := r.URL.Query().Get("url")
	if target == "" {
		http.Error(w, "missing url", http.StatusBadRequest)
		return
	}
	headers := decodeHeaders(r.URL.Query().Get("h"))
	resp, err := s.doUpstream(r, target, headers)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		http.Error(w, fmt.Sprintf("upstream status %d", resp.StatusCode), resp.StatusCode)
		return
	}

	// Read an initial prefix so fake PNG/JPEG/GIF headers can be stripped from TS segments.
	buf := make([]byte, 4096)
	n, readErr := io.ReadFull(resp.Body, buf)
	if readErr != nil && readErr != io.EOF && readErr != io.ErrUnexpectedEOF {
		http.Error(w, readErr.Error(), http.StatusBadGateway)
		return
	}
	head := StripFakeImageHeader(buf[:n])

	ct := resp.Header.Get("Content-Type")
	if len(head) != n || strings.HasPrefix(strings.ToLower(ct), "image/") {
		w.Header().Set("Content-Type", "video/mp2t")
	} else if ct != "" {
		w.Header().Set("Content-Type", ct)
	}
	if _, err := w.Write(head); err != nil {
		return
	}
	if readErr == nil {
		_, _ = io.Copy(w, resp.Body)
	}
}

func (s *Server) doUpstream(r *http.Request, target string, headers map[string]string) (*http.Response, error) {
	u, err := url.Parse(target)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") {
		return nil, fmt.Errorf("invalid target URL %q", target)
	}
	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	if rng := r.Header.Get("Range"); rng != "" {
		req.Header.Set("Range", rng)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	if req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; zy-nsgo/1.0)")
	}
	return s.httpClient.Do(req)
}

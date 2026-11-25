package player

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/core"

	"zy-nsgo/internal/proxy"
	"zy-nsgo/internal/store"
)

// PlayRequest describes an episode playback request.
type PlayRequest struct {
	SiteKey     string
	VodID       string
	VodName     string
	VodPic      string
	PlaySource  string
	EpisodeName string
	URL         string
	Headers     map[string]string
	UseProxy    bool
}

// Controller manages NSGo core.Video playback state, resume-seeking, and progress persistence.
type Controller struct {
	Ref       *core.Ref
	Source    *core.Signal[string]
	Paused    *core.Signal[bool]
	Muted     *core.Signal[bool]
	Volume    *core.Signal[float32]
	Rate      *core.Signal[float32]
	Position  *core.Signal[time.Duration]
	Duration  *core.Signal[time.Duration]
	Buffering *core.Signal[bool]
	Error     *core.Signal[string]

	store        *store.Store
	proxy        *proxy.Server
	current      PlayRequest
	pendingSeek  time.Duration
	lastSaved    time.Time
	OnEpisodeEnd func()
}

// NewController constructs a reactive playback controller bound to cx, store, and optional proxy.
const (
	settingPlayerVolume = "player_volume"
	settingPlayerMuted  = "player_muted"
)

func parseSavedVolume(raw string) float32 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 1.0
	}
	v, err := strconv.ParseFloat(raw, 32)
	if err != nil || v <= 0 {
		return 1.0
	}
	if v > 1.0 {
		return 1.0
	}
	if v < 0.01 {
		return 0.01
	}
	return float32(v)
}

func NewController(cx *core.Scope, st *store.Store, px *proxy.Server) *Controller {
	vol := float32(1.0)
	muted := false
	if st != nil {
		if rawVol, err := st.GetSetting(settingPlayerVolume); err == nil {
			vol = parseSavedVolume(rawVol)
		}
		if rawMuted, err := st.GetSetting(settingPlayerMuted); err == nil && rawMuted == "true" {
			muted = true
		}
	}
	return &Controller{
		Ref:       &core.Ref{},
		Source:    core.NewSignal(cx, ""),
		Paused:    core.NewSignal(cx, false),
		Muted:     core.NewSignal(cx, muted),
		Volume:    core.NewSignal[float32](cx, vol),
		Rate:      core.NewSignal[float32](cx, 1.0),
		Position:  core.NewSignal(cx, time.Duration(0)),
		Duration:  core.NewSignal(cx, time.Duration(0)),
		Buffering: core.NewSignal(cx, false),
		Error:     core.NewSignal(cx, ""),
		store:     st,
		proxy:     px,
	}
}

// ResolvePlayURL returns either a direct URL or a local proxy URL when UseProxy is enabled
// for HLS playlists or streams requiring custom request headers.
func ResolvePlayURL(req PlayRequest, px *proxy.Server) string {
	raw := strings.TrimSpace(req.URL)
	if raw == "" || !req.UseProxy || px == nil || px.BaseURL() == "" {
		return raw
	}
	if len(req.Headers) > 0 || isHLSURL(raw) {
		return px.WrapURL(raw, req.Headers)
	}
	return raw
}

func isHLSURL(raw string) bool {
	u, err := url.Parse(raw)
	if err != nil {
		return strings.Contains(strings.ToLower(raw), ".m3u8")
	}
	return strings.HasSuffix(strings.ToLower(u.Path), ".m3u8")
}

// computeResumePosition returns the saved resume position if meaningful (>3s and >5s before end).
func computeResumePosition(rec *store.HistoryRecord, episodeURL string) time.Duration {
	if rec == nil || rec.PositionMS <= 3000 {
		return 0
	}
	if rec.EpisodeURL != "" && episodeURL != "" && rec.EpisodeURL != episodeURL {
		return 0
	}
	if rec.DurationMS > 0 && rec.DurationMS-rec.PositionMS <= 5000 {
		return 0
	}
	return time.Duration(rec.PositionMS) * time.Millisecond
}

// Play loads a new episode URL, restores any saved resume position, and starts playback.
func (c *Controller) Play(req PlayRequest) {
	c.current = req
	c.pendingSeek = 0
	c.lastSaved = time.Time{}
	c.Error.Set("")
	c.Position.Set(0)
	c.Duration.Set(0)

	if c.store != nil && req.SiteKey != "" && req.VodID != "" {
		if rec, err := c.store.GetProgress(req.SiteKey, req.VodID); err == nil {
			c.pendingSeek = computeResumePosition(rec, req.URL)
		}
	}

	c.Paused.Set(false)
	c.Source.Set(ResolvePlayURL(req, c.proxy))
}

// TogglePause toggles playback pause state.
func (c *Controller) TogglePause() {
	c.Paused.Update(func(p bool) bool { return !p })
}

// SetRate updates playback speed (e.g. 1.0, 1.5, 2.0).
func (c *Controller) SetRate(rate float32) {
	if rate <= 0 {
		rate = 1.0
	}
	c.Rate.Set(rate)
}

// SetVolume updates playback volume (0..1) and persists it across videos and sessions.
// Since core.Video treats Volume=0 as full volume, 0% is represented via Muted=true.
func (c *Controller) SetVolume(vol float32) {
	if vol <= 0 {
		c.Muted.Set(true)
		if c.store != nil {
			_ = c.store.SetSetting(settingPlayerMuted, "true")
		}
		return
	}
	if vol > 1.0 {
		vol = 1.0
	}
	if vol < 0.01 {
		vol = 0.01
	}
	c.Volume.Set(vol)
	c.Muted.Set(false)
	if c.store != nil {
		_ = c.store.SetSetting(settingPlayerVolume, fmt.Sprintf("%.2f", vol))
		_ = c.store.SetSetting(settingPlayerMuted, "false")
	}
}

// ToggleMute toggles audio mute state and persists it.
func (c *Controller) ToggleMute() {
	next := !c.Muted.GetUntracked()
	c.Muted.Set(next)
	if c.store != nil {
		val := "false"
		if next {
			val = "true"
		}
		_ = c.store.SetSetting(settingPlayerMuted, val)
	}
}

// AdjustVolume changes volume by delta (e.g. +0.1 / -0.1) and persists it.
func (c *Controller) AdjustVolume(delta float32) {
	cur := c.Volume.GetUntracked()
	if c.Muted.GetUntracked() {
		cur = 0
	}
	c.SetVolume(cur + delta)
}

// Seek jumps the underlying native video player to position.
func (c *Controller) Seek(position time.Duration) {
	if position < 0 {
		position = 0
	}
	c.Ref.Seek(position)
	c.Position.Set(position)
}

func clampSeekTarget(cur, dur, delta time.Duration) time.Duration {
	target := cur + delta
	if target < 0 {
		return 0
	}
	if dur > 0 && target > dur {
		return dur
	}
	return target
}

// SeekBy jumps relative to the current playback position (e.g. +10s / -10s).
func (c *Controller) SeekBy(delta time.Duration) {
	target := clampSeekTarget(c.Position.GetUntracked(), c.Duration.GetUntracked(), delta)
	c.Seek(target)
}

// View renders the underlying NSGo core.Video node with native controls and callbacks wired.
func (c *Controller) View(cx *core.Scope) core.Node {
	return core.Video{
		Ref:              c.Ref,
		ResizeMode:       core.Contain,
		Style:            core.Style{Width: core.Pct(100), Height: core.Pct(100), FlexGrow: 1},
		SourceFunc:       c.Source.Get,
		PausedFunc:       c.Paused.Get,
		MutedFunc:        c.Muted.Get,
		VolumeFunc:       c.Volume.Get,
		RateFunc:         c.Rate.Get,
		Controls:         true,
		ProgressInterval: time.Second,
		OnLoad: func(info core.VideoInfo) {
			c.Duration.Set(info.Duration)
			if c.pendingSeek > 0 {
				seekTo := c.pendingSeek
				c.pendingSeek = 0
				c.Ref.Seek(seekTo)
			}
		},
		OnProgress: func(pos, dur time.Duration) {
			c.Position.Set(pos)
			if dur > 0 {
				c.Duration.Set(dur)
			}
			if time.Since(c.lastSaved) >= 5*time.Second {
				c.persistProgress(pos, dur)
			}
		},
		OnPause: func(reason core.VideoPauseReason) {
			c.Paused.Set(true)
			c.persistProgress(c.Position.GetUntracked(), c.Duration.GetUntracked())
		},
		OnPlay: func() {
			c.Paused.Set(false)
		},
		OnBuffer: func(b bool) {
			c.Buffering.Set(b)
		},
		OnEnd: func() {
			c.persistProgress(c.Duration.GetUntracked(), c.Duration.GetUntracked())
			if c.OnEpisodeEnd != nil {
				c.OnEpisodeEnd()
			}
		},
		OnError: func(err error) {
			if err != nil {
				c.Error.Set(err.Error())
			}
		},
	}
}

func (c *Controller) persistProgress(pos, dur time.Duration) {
	if c.store == nil || c.current.SiteKey == "" || c.current.VodID == "" {
		return
	}
	c.lastSaved = time.Now()
	_ = c.store.SaveProgress(store.HistoryRecord{
		SiteKey:     c.current.SiteKey,
		VodID:       c.current.VodID,
		VodName:     c.current.VodName,
		VodPic:      c.current.VodPic,
		PlaySource:  c.current.PlaySource,
		EpisodeName: c.current.EpisodeName,
		EpisodeURL:  c.current.URL,
		PositionMS:  pos.Milliseconds(),
		DurationMS:  dur.Milliseconds(),
		UpdatedAt:   time.Now().UnixMilli(),
	})
}

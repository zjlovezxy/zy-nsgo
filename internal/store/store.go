package store

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	_ "modernc.org/sqlite"

	"zy-nsgo/internal/cms"
)

// HistoryRecord stores playback progress for a specific video on a site.
type HistoryRecord struct {
	SiteKey     string `json:"site_key"`
	VodID       string `json:"vod_id"`
	VodName     string `json:"vod_name"`
	VodPic      string `json:"vod_pic"`
	PlaySource  string `json:"play_source"`
	EpisodeName string `json:"episode_name"`
	EpisodeURL  string `json:"episode_url"`
	PositionMS  int64  `json:"position_ms"`
	DurationMS  int64  `json:"duration_ms"`
	UpdatedAt   int64  `json:"updated_at"`
}

// FavoriteRecord stores a bookmarked video item.
type FavoriteRecord struct {
	SiteKey   string `json:"site_key"`
	VodID     string `json:"vod_id"`
	VodName   string `json:"vod_name"`
	VodPic    string `json:"vod_pic"`
	Remarks   string `json:"remarks"`
	CreatedAt int64  `json:"created_at"`
}

// Store wraps pure-Go SQLite persistence for sites, history, favorites, and settings.
type Store struct {
	db *sql.DB
}

// Open opens the SQLite database at dsn and runs schema migrations.
func Open(dsn string) (*Store, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", dsn, err)
	}
	if dsn != ":memory:" && !strings.Contains(dsn, "mode=memory") {
		if _, err := db.Exec(`PRAGMA journal_mode=WAL;`); err != nil {
			_ = db.Close()
			return nil, fmt.Errorf("enable WAL: %w", err)
		}
	}
	if _, err := db.Exec(`PRAGMA busy_timeout=5000;`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set busy_timeout: %w", err)
	}
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close closes the underlying database connection.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func migrate(db *sql.DB) error {
	const schema = `
CREATE TABLE IF NOT EXISTS sites (
	key TEXT PRIMARY KEY,
	name TEXT NOT NULL,
	api TEXT NOT NULL,
	active INTEGER NOT NULL DEFAULT 1,
	sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS history (
	site_key TEXT NOT NULL,
	vod_id TEXT NOT NULL,
	vod_name TEXT NOT NULL,
	vod_pic TEXT NOT NULL DEFAULT "",
	play_source TEXT NOT NULL DEFAULT "",
	episode_name TEXT NOT NULL DEFAULT "",
	episode_url TEXT NOT NULL DEFAULT "",
	position_ms INTEGER NOT NULL DEFAULT 0,
	duration_ms INTEGER NOT NULL DEFAULT 0,
	updated_at INTEGER NOT NULL,
	PRIMARY KEY (site_key, vod_id)
);

CREATE TABLE IF NOT EXISTS favorites (
	site_key TEXT NOT NULL,
	vod_id TEXT NOT NULL,
	vod_name TEXT NOT NULL,
	vod_pic TEXT NOT NULL DEFAULT "",
	remarks TEXT NOT NULL DEFAULT "",
	created_at INTEGER NOT NULL,
	PRIMARY KEY (site_key, vod_id)
);

CREATE TABLE IF NOT EXISTS settings (
	key TEXT PRIMARY KEY,
	value TEXT NOT NULL
);`
	if _, err := db.Exec(schema); err != nil {
		return fmt.Errorf("migrate schema: %w", err)
	}
	return nil
}

// UpsertSite inserts or updates a CMS site configuration.
func (s *Store) UpsertSite(site cms.Site, sortOrder int) error {
	if strings.TrimSpace(site.Key) == "" || strings.TrimSpace(site.API) == "" {
		return errors.New("site key and api are required")
	}
	active := 0
	if site.Active {
		active = 1
	}
	_, err := s.db.Exec(
		`INSERT INTO sites (key, name, api, active, sort_order) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET name=excluded.name, api=excluded.api, active=excluded.active, sort_order=excluded.sort_order`,
		site.Key, site.Name, site.API, active, sortOrder,
	)
	return err
}

// ListSites returns all configured CMS sites ordered by sort_order then key.
func (s *Store) ListSites() ([]cms.Site, error) {
	rows, err := s.db.Query(`SELECT key, name, api, active FROM sites ORDER BY sort_order ASC, key ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []cms.Site
	for rows.Next() {
		var item cms.Site
		var active int
		if err := rows.Scan(&item.Key, &item.Name, &item.API, &active); err != nil {
			return nil, err
		}
		item.Active = active != 0
		out = append(out, item)
	}
	return out, rows.Err()
}

// DeleteSite deletes a CMS site by key.
func (s *Store) DeleteSite(key string) error {
	_, err := s.db.Exec(`DELETE FROM sites WHERE key = ?`, key)
	return err
}

type importSiteItem struct {
	Key      string `json:"key"`
	Name     string `json:"name"`
	API      string `json:"api"`
	Active   *bool  `json:"active"`
	IsActive *bool  `json:"isActive"`
}

// ImportSitesJSON imports CMS sites from either a JSON array or zyfun-style {"sites": ...} object.
func (s *Store) ImportSitesJSON(data []byte) (int, error) {
	items, err := parseImportSites(data)
	if err != nil {
		return 0, err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.Prepare(
		`INSERT INTO sites (key, name, api, active, sort_order) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(key) DO UPDATE SET name=excluded.name, api=excluded.api, active=excluded.active, sort_order=excluded.sort_order`,
	)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	count := 0
	for i, it := range items {
		key := strings.TrimSpace(it.Key)
		api := strings.TrimSpace(it.API)
		if api == "" {
			continue
		}
		if key == "" {
			key = fmt.Sprintf("site-%d", i+1)
		}
		name := strings.TrimSpace(it.Name)
		if name == "" {
			name = key
		}
		active := 1
		if it.Active != nil && !*it.Active {
			active = 0
		} else if it.IsActive != nil && !*it.IsActive {
			active = 0
		}
		if _, err := stmt.Exec(key, name, api, active, i); err != nil {
			return 0, err
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return count, nil
}

func parseImportSites(data []byte) ([]importSiteItem, error) {
	var arr []importSiteItem
	if err := json.Unmarshal(data, &arr); err == nil {
		return arr, nil
	}

	var obj struct {
		Sites json.RawMessage `json:"sites"`
	}
	if err := json.Unmarshal(data, &obj); err != nil || len(obj.Sites) == 0 {
		return nil, fmt.Errorf("unsupported sites JSON format")
	}
	if err := json.Unmarshal(obj.Sites, &arr); err == nil {
		return arr, nil
	}
	var nested struct {
		Data []importSiteItem `json:"data"`
	}
	if err := json.Unmarshal(obj.Sites, &nested); err != nil {
		return nil, fmt.Errorf("decode nested sites.data: %w", err)
	}
	return nested.Data, nil
}

// SaveProgress inserts or updates playback history for a video.
func (s *Store) SaveProgress(rec HistoryRecord) error {
	if rec.UpdatedAt == 0 {
		rec.UpdatedAt = time.Now().UnixMilli()
	}
	_, err := s.db.Exec(
		`INSERT INTO history (site_key, vod_id, vod_name, vod_pic, play_source, episode_name, episode_url, position_ms, duration_ms, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(site_key, vod_id) DO UPDATE SET
		   vod_name=excluded.vod_name,
		   vod_pic=excluded.vod_pic,
		   play_source=excluded.play_source,
		   episode_name=excluded.episode_name,
		   episode_url=excluded.episode_url,
		   position_ms=excluded.position_ms,
		   duration_ms=excluded.duration_ms,
		   updated_at=excluded.updated_at`,
		rec.SiteKey, rec.VodID, rec.VodName, rec.VodPic, rec.PlaySource, rec.EpisodeName, rec.EpisodeURL, rec.PositionMS, rec.DurationMS, rec.UpdatedAt,
	)
	return err
}

// GetProgress fetches the last saved playback record for a video, or nil if none exists.
func (s *Store) GetProgress(siteKey, vodID string) (*HistoryRecord, error) {
	var rec HistoryRecord
	err := s.db.QueryRow(
		`SELECT site_key, vod_id, vod_name, vod_pic, play_source, episode_name, episode_url, position_ms, duration_ms, updated_at
		 FROM history WHERE site_key = ? AND vod_id = ?`,
		siteKey, vodID,
	).Scan(&rec.SiteKey, &rec.VodID, &rec.VodName, &rec.VodPic, &rec.PlaySource, &rec.EpisodeName, &rec.EpisodeURL, &rec.PositionMS, &rec.DurationMS, &rec.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rec, nil
}

// ListHistory returns playback history sorted from most recent to oldest.
func (s *Store) ListHistory(limit int) ([]HistoryRecord, error) {
	if limit <= 0 {
		limit = 50
	}
	rows, err := s.db.Query(
		`SELECT site_key, vod_id, vod_name, vod_pic, play_source, episode_name, episode_url, position_ms, duration_ms, updated_at
		 FROM history ORDER BY updated_at DESC LIMIT ?`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []HistoryRecord
	for rows.Next() {
		var rec HistoryRecord
		if err := rows.Scan(&rec.SiteKey, &rec.VodID, &rec.VodName, &rec.VodPic, &rec.PlaySource, &rec.EpisodeName, &rec.EpisodeURL, &rec.PositionMS, &rec.DurationMS, &rec.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// DeleteHistory removes a single history record.
func (s *Store) DeleteHistory(siteKey, vodID string) error {
	_, err := s.db.Exec(`DELETE FROM history WHERE site_key = ? AND vod_id = ?`, siteKey, vodID)
	return err
}

// AddFavorite adds or updates a favorite video record.
func (s *Store) AddFavorite(fav FavoriteRecord) error {
	if fav.CreatedAt == 0 {
		fav.CreatedAt = time.Now().UnixMilli()
	}
	_, err := s.db.Exec(
		`INSERT INTO favorites (site_key, vod_id, vod_name, vod_pic, remarks, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)
		 ON CONFLICT(site_key, vod_id) DO UPDATE SET vod_name=excluded.vod_name, vod_pic=excluded.vod_pic, remarks=excluded.remarks`,
		fav.SiteKey, fav.VodID, fav.VodName, fav.VodPic, fav.Remarks, fav.CreatedAt,
	)
	return err
}

// RemoveFavorite removes a favorite video record.
func (s *Store) RemoveFavorite(siteKey, vodID string) error {
	_, err := s.db.Exec(`DELETE FROM favorites WHERE site_key = ? AND vod_id = ?`, siteKey, vodID)
	return err
}

// IsFavorite reports whether a video is currently bookmarked.
func (s *Store) IsFavorite(siteKey, vodID string) (bool, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(1) FROM favorites WHERE site_key = ? AND vod_id = ?`, siteKey, vodID).Scan(&count)
	return count > 0, err
}

// ListFavorites returns all favorite records ordered newest first.
func (s *Store) ListFavorites() ([]FavoriteRecord, error) {
	rows, err := s.db.Query(`SELECT site_key, vod_id, vod_name, vod_pic, remarks, created_at FROM favorites ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []FavoriteRecord
	for rows.Next() {
		var fav FavoriteRecord
		if err := rows.Scan(&fav.SiteKey, &fav.VodID, &fav.VodName, &fav.VodPic, &fav.Remarks, &fav.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, fav)
	}
	return out, rows.Err()
}

// GetSetting retrieves a setting value by key, returning "" if unset.
func (s *Store) GetSetting(key string) (string, error) {
	var val string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key = ?`, key).Scan(&val)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return val, err
}

// SetSetting stores a key/value setting.
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.Exec(
		`INSERT INTO settings (key, value) VALUES (?, ?) ON CONFLICT(key) DO UPDATE SET value=excluded.value`,
		key, value,
	)
	return err
}

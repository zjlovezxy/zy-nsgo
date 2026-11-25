package cms

// Site represents a T1 (XML) or T2 (JSON) Apple CMS video source.
type Site struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	API    string `json:"api"`
	Active bool   `json:"active"`
}

// Category represents a CMS video category.
type Category struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// VodItem represents a video item in list or search responses.
type VodItem struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	TypeID     string `json:"type_id"`
	TypeName   string `json:"type_name"`
	Pic        string `json:"pic"`
	Remarks    string `json:"remarks"`
	Year       string `json:"year"`
	Area       string `json:"area"`
	UpdateTime string `json:"update_time"`
}

// VodDetail represents detailed video metadata and parsed play sources.
type VodDetail struct {
	VodItem
	Actor    string       `json:"actor"`
	Director string       `json:"director"`
	Content  string       `json:"content"`
	PlayFrom string       `json:"play_from"`
	PlayURL  string       `json:"play_url"`
	Sources  []PlaySource `json:"sources"`
}

// PlaySource represents one play line (e.g., "bfzym3u8") and its episodes.
type PlaySource struct {
	Name     string    `json:"name"`
	Episodes []Episode `json:"episodes"`
}

// Episode represents a single playable episode.
type Episode struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// PageResult represents a paginated list or search response from a CMS site.
type PageResult struct {
	Page       int        `json:"page"`
	PageCount  int        `json:"page_count"`
	Total      int        `json:"total"`
	Categories []Category `json:"categories"`
	List       []VodItem  `json:"list"`
}

// SearchResult holds one site result from concurrent aggregate search.
type SearchResult struct {
	Site  Site      `json:"site"`
	Items []VodItem `json:"items"`
	Err   error     `json:"err,omitempty"`
}

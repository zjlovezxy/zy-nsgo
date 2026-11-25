package cms

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
)

// ParsePlaySources splits Apple CMS vod_play_from and vod_play_url strings
// (separated by "$$$" across lines, "#" across episodes, and "$" between title and URL).
func ParsePlaySources(vodPlayFrom, vodPlayURL string) []PlaySource {
	rawURL := strings.TrimSpace(vodPlayURL)
	if rawURL == "" {
		return nil
	}
	fromParts := strings.Split(strings.TrimSpace(vodPlayFrom), "$$$")
	urlParts := strings.Split(rawURL, "$$$")

	sources := make([]PlaySource, 0, len(urlParts))
	for i, group := range urlParts {
		group = strings.TrimSpace(group)
		if group == "" {
			continue
		}
		lineName := fmt.Sprintf("Line %d", i+1)
		if i < len(fromParts) && strings.TrimSpace(fromParts[i]) != "" {
			lineName = strings.TrimSpace(fromParts[i])
		}

		rawEps := strings.Split(group, "#")
		episodes := make([]Episode, 0, len(rawEps))
		for j, rawEp := range rawEps {
			rawEp = strings.TrimSpace(rawEp)
			if rawEp == "" {
				continue
			}
			epName := fmt.Sprintf("第%d集", j+1)
			epURL := rawEp
			if idx := strings.Index(rawEp, "$"); idx >= 0 {
				if name := strings.TrimSpace(rawEp[:idx]); name != "" {
					epName = name
				}
				epURL = strings.TrimSpace(rawEp[idx+1:])
			}
			if epURL == "" {
				continue
			}
			episodes = append(episodes, Episode{Name: epName, URL: epURL})
		}
		if len(episodes) > 0 {
			sources = append(sources, PlaySource{Name: lineName, Episodes: episodes})
		}
	}
	return sources
}

// ParseResponse detects T1 XML vs T2 JSON and normalizes it into PageResult and VodDetail slices.
func ParseResponse(data []byte) (*PageResult, []VodDetail, error) {
	trimmed := bytes.TrimSpace(bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}))
	if len(trimmed) == 0 {
		return nil, nil, fmt.Errorf("empty CMS response")
	}
	if trimmed[0] == '<' {
		return parseXML(trimmed)
	}
	return parseJSON(trimmed)
}

type xmlRSS struct {
	XMLName xml.Name `xml:"rss"`
	Class   struct {
		Types []xmlType `xml:"ty"`
	} `xml:"class"`
	List struct {
		Page        flexInt    `xml:"page,attr"`
		PageCount   flexInt    `xml:"pagecount,attr"`
		RecordCount flexInt    `xml:"recordcount,attr"`
		Types       []xmlType  `xml:"ty"`
		Videos      []xmlVideo `xml:"video"`
	} `xml:"list"`
}

type xmlType struct {
	ID   string `xml:"id,attr"`
	Name string `xml:",chardata"`
}

type xmlVideo struct {
	Last     string `xml:"last"`
	ID       string `xml:"id"`
	TID      string `xml:"tid"`
	Name     string `xml:"name"`
	Type     string `xml:"type"`
	Pic      string `xml:"pic"`
	Year     string `xml:"year"`
	Area     string `xml:"area"`
	Note     string `xml:"note"`
	Actor    string `xml:"actor"`
	Director string `xml:"director"`
	Des      string `xml:"des"`
	DL       struct {
		DD []struct {
			Flag string `xml:"flag,attr"`
			Data string `xml:",chardata"`
		} `xml:"dd"`
	} `xml:"dl"`
}

func parseXML(data []byte) (*PageResult, []VodDetail, error) {
	var doc xmlRSS
	if err := xml.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("decode T1 XML: %w", err)
	}

	rawTypes := doc.Class.Types
	if len(rawTypes) == 0 {
		rawTypes = doc.List.Types
	}
	cats := make([]Category, 0, len(rawTypes))
	for _, t := range rawTypes {
		if id := strings.TrimSpace(t.ID); id != "" {
			cats = append(cats, Category{ID: id, Name: strings.TrimSpace(t.Name)})
		}
	}

	items := make([]VodItem, 0, len(doc.List.Videos))
	details := make([]VodDetail, 0, len(doc.List.Videos))
	for _, v := range doc.List.Videos {
		var fromList, urlList []string
		for _, dd := range v.DL.DD {
			flag := strings.TrimSpace(dd.Flag)
			body := strings.TrimSpace(dd.Data)
			if body == "" {
				continue
			}
			fromList = append(fromList, flag)
			urlList = append(urlList, body)
		}
		playFrom := strings.Join(fromList, "$$$")
		playURL := strings.Join(urlList, "$$$")

		item := VodItem{
			ID:         strings.TrimSpace(v.ID),
			Name:       strings.TrimSpace(v.Name),
			TypeID:     strings.TrimSpace(v.TID),
			TypeName:   strings.TrimSpace(v.Type),
			Pic:        strings.TrimSpace(v.Pic),
			Remarks:    strings.TrimSpace(v.Note),
			Year:       strings.TrimSpace(v.Year),
			Area:       strings.TrimSpace(v.Area),
			UpdateTime: strings.TrimSpace(v.Last),
		}
		items = append(items, item)
		details = append(details, VodDetail{
			VodItem:  item,
			Actor:    strings.TrimSpace(v.Actor),
			Director: strings.TrimSpace(v.Director),
			Content:  strings.TrimSpace(v.Des),
			PlayFrom: playFrom,
			PlayURL:  playURL,
			Sources:  ParsePlaySources(playFrom, playURL),
		})
	}

	res := &PageResult{
		Page:       int(doc.List.Page),
		PageCount:  int(doc.List.PageCount),
		Total:      int(doc.List.RecordCount),
		Categories: cats,
		List:       items,
	}
	return res, details, nil
}

type jsonResponse struct {
	Code      flexInt        `json:"code"`
	Msg       string         `json:"msg"`
	Page      flexInt        `json:"page"`
	PageCount flexInt        `json:"pagecount"`
	Total     flexInt        `json:"total"`
	Class     []jsonCategory `json:"class"`
	List      []jsonVod      `json:"list"`
}

type jsonCategory struct {
	TypeID   flexString `json:"type_id"`
	TypeName flexString `json:"type_name"`
}

type jsonVod struct {
	VodID       flexString `json:"vod_id"`
	VodName     flexString `json:"vod_name"`
	TypeID      flexString `json:"type_id"`
	TypeName    flexString `json:"type_name"`
	VodPic      flexString `json:"vod_pic"`
	VodRemarks  flexString `json:"vod_remarks"`
	VodYear     flexString `json:"vod_year"`
	VodArea     flexString `json:"vod_area"`
	VodTime     flexString `json:"vod_time"`
	VodActor    flexString `json:"vod_actor"`
	VodDirector flexString `json:"vod_director"`
	VodContent  flexString `json:"vod_content"`
	VodPlayFrom flexString `json:"vod_play_from"`
	VodPlayURL  flexString `json:"vod_play_url"`
}

func parseJSON(data []byte) (*PageResult, []VodDetail, error) {
	var doc jsonResponse
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, nil, fmt.Errorf("decode T2 JSON: %w", err)
	}

	cats := make([]Category, 0, len(doc.Class))
	for _, c := range doc.Class {
		if id := strings.TrimSpace(string(c.TypeID)); id != "" {
			cats = append(cats, Category{ID: id, Name: strings.TrimSpace(string(c.TypeName))})
		}
	}

	items := make([]VodItem, 0, len(doc.List))
	details := make([]VodDetail, 0, len(doc.List))
	for _, v := range doc.List {
		playFrom := strings.TrimSpace(string(v.VodPlayFrom))
		playURL := strings.TrimSpace(string(v.VodPlayURL))
		item := VodItem{
			ID:         strings.TrimSpace(string(v.VodID)),
			Name:       strings.TrimSpace(string(v.VodName)),
			TypeID:     strings.TrimSpace(string(v.TypeID)),
			TypeName:   strings.TrimSpace(string(v.TypeName)),
			Pic:        strings.TrimSpace(string(v.VodPic)),
			Remarks:    strings.TrimSpace(string(v.VodRemarks)),
			Year:       strings.TrimSpace(string(v.VodYear)),
			Area:       strings.TrimSpace(string(v.VodArea)),
			UpdateTime: strings.TrimSpace(string(v.VodTime)),
		}
		items = append(items, item)
		details = append(details, VodDetail{
			VodItem:  item,
			Actor:    strings.TrimSpace(string(v.VodActor)),
			Director: strings.TrimSpace(string(v.VodDirector)),
			Content:  strings.TrimSpace(string(v.VodContent)),
			PlayFrom: playFrom,
			PlayURL:  playURL,
			Sources:  ParsePlaySources(playFrom, playURL),
		})
	}

	res := &PageResult{
		Page:       int(doc.Page),
		PageCount:  int(doc.PageCount),
		Total:      int(doc.Total),
		Categories: cats,
		List:       items,
	}
	return res, details, nil
}

// flexString unmarshals JSON strings, numbers, or null into a string.
type flexString string

func (f *flexString) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*f = ""
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexString(s)
		return nil
	}
	*f = flexString(string(b))
	return nil
}

// flexInt unmarshals JSON integers, floats, strings, or XML attributes into an int.
type flexInt int

func (f *flexInt) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" || string(b) == `""` {
		*f = 0
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			*f = 0
			return nil
		}
		*f = flexInt(n)
		return nil
	}
	var n int
	if err := json.Unmarshal(b, &n); err == nil {
		*f = flexInt(n)
		return nil
	}
	var fl float64
	if err := json.Unmarshal(b, &fl); err == nil {
		*f = flexInt(int(fl))
		return nil
	}
	*f = 0
	return nil
}

func (f *flexInt) UnmarshalXMLAttr(attr xml.Attr) error {
	s := strings.TrimSpace(attr.Value)
	if s == "" {
		*f = 0
		return nil
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		*f = 0
		return nil
	}
	*f = flexInt(n)
	return nil
}

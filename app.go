package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"

	"zy-nsgo/internal/cms"
	"zy-nsgo/internal/player"
	"zy-nsgo/internal/proxy"
	"zy-nsgo/internal/store"
)

const (
	tabBrowse    = "browse"
	tabSearch    = "search"
	tabFavorites = "favorites"
	tabHistory   = "history"
	tabSettings  = "settings"
)

// AppState holds reactive UI state and orchestrates CMS, SQLite store, proxy, and player.
type AppState struct {
	cx     *core.Scope
	cms    *cms.Client
	store  *store.Store
	proxy  *proxy.Server
	player *player.Controller

	tab        *core.Signal[string]
	detailOpen *core.Signal[bool]

	sites           *core.Signal[[]cms.Site]
	selectedSiteKey *core.Signal[string]
	useProxy        *core.Signal[bool]

	categories    *core.Signal[[]cms.Category]
	selectedCatID *core.Signal[string]
	page          *core.Signal[int]
	pageCount     *core.Signal[int]
	videos        *core.Signal[[]cms.VodItem]
	loading          *core.Signal[bool]
	loadingMore      *core.Signal[bool]
	statusMsg        *core.Signal[string]
	fullScreenMode   *core.Signal[string]

	searchQuery        *core.Signal[string]
	searching          *core.Signal[bool]
	searchResults      *core.Signal[[]cms.SearchResult]
	searchSelectedKeys *core.Signal[map[string]bool]

	favorites *core.Signal[[]store.FavoriteRecord]
	history   *core.Signal[[]store.HistoryRecord]

	detail        *core.Signal[*cms.VodDetail]
	detailSite    *core.Signal[cms.Site]
	detailLoading *core.Signal[bool]
	isFav         *core.Signal[bool]
	activeLineIdx *core.Signal[int]
	activeEpIdx   *core.Signal[int]

	// Settings form inputs
	newSiteKey  *core.Signal[string]
	newSiteName *core.Signal[string]
	newSiteAPI  *core.Signal[string]
	importJSON  *core.Signal[string]
	settingsMsg *core.Signal[string]
}

func newAppState(cx *core.Scope, client *cms.Client, st *store.Store, px *proxy.Server) *AppState {
	useProxy := true
	if v, err := st.GetSetting("use_proxy"); err == nil && v == "false" {
		useProxy = false
	}

	a := &AppState{
		cx:              cx,
		cms:             client,
		store:           st,
		proxy:           px,
		player:          player.NewController(cx, st, px),
		tab:             core.NewSignal(cx, tabBrowse),
		detailOpen:      core.NewSignal(cx, false),
		sites:           core.NewSignalFunc[[]cms.Site](cx, nil, nil),
		selectedSiteKey: core.NewSignal(cx, ""),
		useProxy:        core.NewSignal(cx, useProxy),
		categories:      core.NewSignalFunc[[]cms.Category](cx, nil, nil),
		selectedCatID:   core.NewSignal(cx, ""),
		page:            core.NewSignal(cx, 1),
		pageCount:       core.NewSignal(cx, 1),
		videos:          core.NewSignalFunc[[]cms.VodItem](cx, nil, nil),
		loading:          core.NewSignal(cx, false),
		loadingMore:      core.NewSignal(cx, false),
		statusMsg:        core.NewSignal(cx, ""),
		fullScreenMode:   core.NewSignal(cx, ""),
		searchQuery:        core.NewSignal(cx, ""),
		searching:          core.NewSignal(cx, false),
		searchResults:      core.NewSignalFunc[[]cms.SearchResult](cx, nil, nil),
		searchSelectedKeys: core.NewSignalFunc[map[string]bool](cx, map[string]bool{}, nil),
		favorites:       core.NewSignalFunc[[]store.FavoriteRecord](cx, nil, nil),
		history:         core.NewSignalFunc[[]store.HistoryRecord](cx, nil, nil),
		detail:          core.NewSignalFunc[*cms.VodDetail](cx, nil, nil),
		detailSite:      core.NewSignalFunc[cms.Site](cx, cms.Site{}, nil),
		detailLoading:   core.NewSignal(cx, false),
		isFav:           core.NewSignal(cx, false),
		activeLineIdx:   core.NewSignal(cx, 0),
		activeEpIdx:     core.NewSignal(cx, 0),
		newSiteKey:      core.NewSignal(cx, ""),
		newSiteName:     core.NewSignal(cx, ""),
		newSiteAPI:      core.NewSignal(cx, ""),
		importJSON:      core.NewSignal(cx, ""),
		settingsMsg:     core.NewSignal(cx, ""),
	}

	a.player.OnEpisodeEnd = func() {
		a.cx.Post(func() {
			det := a.detail.GetUntracked()
			if det == nil {
				return
			}
			lineIdx := a.activeLineIdx.GetUntracked()
			epIdx := a.activeEpIdx.GetUntracked()
			if lineIdx >= 0 && lineIdx < len(det.Sources) {
				if epIdx+1 < len(det.Sources[lineIdx].Episodes) {
					a.playEpisode(lineIdx, epIdx+1)
				}
			}
		})
	}

	a.reloadSites()
	a.reloadFavorites()
	a.reloadHistory()

	sites := a.sites.GetUntracked()
	for _, s := range sites {
		if s.Active {
			a.selectSite(s.Key)
			break
		}
	}
	return a
}

func (a *AppState) switchTab(t string) {
	a.detailOpen.Set(false)
	a.tab.Set(t)
	switch t {
	case tabFavorites:
		a.reloadFavorites()
	case tabHistory:
		a.reloadHistory()
	}
}

func (a *AppState) reloadSites() {
	sites, err := a.store.ListSites()
	if err != nil {
		a.statusMsg.Set("读取站点失败: " + err.Error())
		return
	}
	a.sites.Set(sites)
}

func (a *AppState) findSite(key string) (cms.Site, bool) {
	for _, s := range a.sites.GetUntracked() {
		if s.Key == key {
			return s, true
		}
	}
	return cms.Site{}, false
}

func (a *AppState) activeSites() []cms.Site {
	var out []cms.Site
	for _, s := range a.sites.GetUntracked() {
		if s.Active {
			out = append(out, s)
		}
	}
	return out
}

func (a *AppState) selectSite(key string) {
	site, ok := a.findSite(key)
	if !ok {
		return
	}
	a.selectedSiteKey.Set(site.Key)
	a.selectedCatID.Set("")
	a.page.Set(1)
	a.loadCategories(site)
	a.loadPage(site, "", 1)
}

func (a *AppState) loadCategories(site cms.Site) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
		defer cancel()
		cats, err := a.cms.Categories(ctx, site)
		a.cx.Post(func() {
			if a.selectedSiteKey.GetUntracked() != site.Key {
				return
			}
			if err == nil {
				a.categories.Set(cats)
			}
		})
	}()
}

func (a *AppState) loadPage(site cms.Site, catID string, pageNum int) {
	if pageNum < 1 {
		pageNum = 1
	}
	a.selectedCatID.Set(catID)
	a.page.Set(pageNum)
	a.loading.Set(true)
	a.loadingMore.Set(false)
	a.statusMsg.Set("")

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		res, err := a.cms.List(ctx, site, catID, pageNum)
		a.cx.Post(func() {
			if a.selectedSiteKey.GetUntracked() != site.Key || a.selectedCatID.GetUntracked() != catID {
				return
			}
			a.loading.Set(false)
			if err != nil {
				a.videos.Set(nil)
				a.statusMsg.Set("加载失败: " + err.Error())
				return
			}
			if len(res.Categories) > 0 && len(a.categories.GetUntracked()) == 0 {
				a.categories.Set(res.Categories)
			}
			pc := res.PageCount
			if pc < 1 {
				pc = 1
			}
			a.pageCount.Set(pc)
			a.videos.Set(res.List)
			if len(res.List) == 0 {
				a.statusMsg.Set("该分类暂无数据")
			}
		})
	}()
}

func (a *AppState) loadNextPage() {
	if a.loading.GetUntracked() || a.loadingMore.GetUntracked() {
		return
	}
	curPage := a.page.GetUntracked()
	maxPage := a.pageCount.GetUntracked()
	if curPage >= maxPage {
		return
	}
	site, ok := a.findSite(a.selectedSiteKey.GetUntracked())
	if !ok {
		return
	}
	catID := a.selectedCatID.GetUntracked()
	nextPage := curPage + 1
	a.loadingMore.Set(true)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		res, err := a.cms.List(ctx, site, catID, nextPage)
		a.cx.Post(func() {
			a.loadingMore.Set(false)
			if a.selectedSiteKey.GetUntracked() != site.Key || a.selectedCatID.GetUntracked() != catID {
				return
			}
			if err != nil {
				a.statusMsg.Set("加载更多失败: " + err.Error())
				return
			}
			if res.PageCount > 0 {
				a.pageCount.Set(res.PageCount)
			}
			existing := a.videos.GetUntracked()
			seen := make(map[string]struct{}, len(existing))
			for _, it := range existing {
				seen[it.ID] = struct{}{}
			}
			merged := append([]cms.VodItem{}, existing...)
			for _, it := range res.List {
				if _, dup := seen[it.ID]; !dup {
					seen[it.ID] = struct{}{}
					merged = append(merged, it)
				}
			}
			a.page.Set(nextPage)
			a.videos.Set(merged)
		})
	}()
}

const (
	fsNone   = ""
	fsPage   = "page"
	fsWindow = "window"
)

func (a *AppState) isPlayerExpanded() bool {
	return a.fullScreenMode.Get() != fsNone
}

func (a *AppState) togglePageFullScreen() {
	cur := a.fullScreenMode.GetUntracked()
	if cur == fsPage {
		a.fullScreenMode.Set(fsNone)
		return
	}
	if cur == fsWindow && desktop.FullScreen(a.cx) {
		desktop.ToggleFullScreen(a.cx)
	}
	a.fullScreenMode.Set(fsPage)
}

func (a *AppState) toggleWindowFullScreen() {
	cur := a.fullScreenMode.GetUntracked()
	if cur == fsWindow {
		a.fullScreenMode.Set(fsNone)
		if desktop.FullScreen(a.cx) {
			desktop.ToggleFullScreen(a.cx)
		}
		return
	}
	a.fullScreenMode.Set(fsWindow)
	if !desktop.FullScreen(a.cx) {
		desktop.ToggleFullScreen(a.cx)
	}
}

func (a *AppState) exitPlayerFullScreen() {
	if a.fullScreenMode.GetUntracked() != fsNone {
		a.fullScreenMode.Set(fsNone)
	}
	if desktop.FullScreen(a.cx) {
		desktop.ToggleFullScreen(a.cx)
	}
}

func (a *AppState) selectSearchAllSites() {
	a.searchSelectedKeys.Set(map[string]bool{})
}

func (a *AppState) toggleSearchSite(key string) {
	cur := a.searchSelectedKeys.GetUntracked()
	next := make(map[string]bool, len(cur)+1)
	for k, v := range cur {
		if v {
			next[k] = true
		}
	}
	if next[key] {
		delete(next, key)
	} else {
		next[key] = true
	}
	a.searchSelectedKeys.Set(next)
}

func (a *AppState) runSearch(keyword string) {
	kw := strings.TrimSpace(keyword)
	if kw == "" {
		return
	}
	allSites := a.sites.GetUntracked()
	selected := a.searchSelectedKeys.GetUntracked()
	targetSites := allSites
	if len(selected) > 0 {
		targetSites = make([]cms.Site, 0, len(selected))
		for _, site := range allSites {
			if selected[site.Key] {
				site.Active = true
				targetSites = append(targetSites, site)
			}
		}
	}
	a.searching.Set(true)
	a.searchResults.Set(nil)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		results := a.cms.AggregateSearch(ctx, targetSites, kw)
		a.cx.Post(func() {
			a.searching.Set(false)
			a.searchResults.Set(results)
		})
	}()
}

func (a *AppState) openDetail(site cms.Site, vodID string) {
	a.detailOpen.Set(true)
	a.detailSite.Set(site)
	a.detail.Set(nil)
	a.detailLoading.Set(true)
	a.activeLineIdx.Set(0)
	a.activeEpIdx.Set(0)
	a.player.Paused.Set(true)
	a.player.Source.Set("")

	fav, _ := a.store.IsFavorite(site.Key, vodID)
	a.isFav.Set(fav)
	savedRec, _ := a.store.GetProgress(site.Key, vodID)

	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		det, err := a.cms.Detail(ctx, site, vodID)
		a.cx.Post(func() {
			a.detailLoading.Set(false)
			if err != nil {
				a.player.Error.Set("加载详情失败: " + err.Error())
				return
			}
			a.detail.Set(det)
			lineIdx, epIdx := matchSavedEpisode(det, savedRec)
			if len(det.Sources) > 0 && len(det.Sources[lineIdx].Episodes) > 0 {
				a.playEpisode(lineIdx, epIdx)
			}
		})
	}()
}

func matchSavedEpisode(det *cms.VodDetail, rec *store.HistoryRecord) (int, int) {
	if det == nil {
		return 0, 0
	}
	if rec != nil {
		for li, src := range det.Sources {
			for ei, ep := range src.Episodes {
				if rec.EpisodeURL != "" && ep.URL == rec.EpisodeURL {
					return li, ei
				}
				if rec.PlaySource != "" && src.Name == rec.PlaySource && ep.Name == rec.EpisodeName {
					return li, ei
				}
			}
		}
	}
	// Prefer a directly playable stream line (e.g. jinyingm3u8 over webpage jinyingyun).
	for li, src := range det.Sources {
		if strings.Contains(strings.ToLower(src.Name), "m3u8") && len(src.Episodes) > 0 {
			return li, 0
		}
		if len(src.Episodes) > 0 {
			u := strings.ToLower(src.Episodes[0].URL)
			if strings.Contains(u, ".m3u8") || strings.Contains(u, ".mp4") || strings.Contains(u, ".mov") || strings.Contains(u, ".mkv") {
				return li, 0
			}
		}
	}
	return 0, 0
}

func (a *AppState) playEpisode(lineIdx, epIdx int) {
	det := a.detail.GetUntracked()
	site := a.detailSite.GetUntracked()
	if det == nil || lineIdx < 0 || lineIdx >= len(det.Sources) {
		return
	}
	src := det.Sources[lineIdx]
	if epIdx < 0 || epIdx >= len(src.Episodes) {
		return
	}
	ep := src.Episodes[epIdx]
	a.activeLineIdx.Set(lineIdx)
	a.activeEpIdx.Set(epIdx)

	a.player.Play(player.PlayRequest{
		SiteKey:     site.Key,
		VodID:       det.ID,
		VodName:     det.Name,
		VodPic:      det.Pic,
		PlaySource:  src.Name,
		EpisodeName: ep.Name,
		URL:         ep.URL,
		UseProxy:    a.useProxy.GetUntracked(),
	})
}

func (a *AppState) closeDetail() {
	a.exitPlayerFullScreen()
	a.player.Paused.Set(true)
	a.player.Source.Set("")
	a.detailOpen.Set(false)
	a.reloadHistory()
}

func (a *AppState) toggleFavorite() {
	det := a.detail.GetUntracked()
	site := a.detailSite.GetUntracked()
	if det == nil {
		return
	}
	if a.isFav.GetUntracked() {
		_ = a.store.RemoveFavorite(site.Key, det.ID)
		a.isFav.Set(false)
	} else {
		_ = a.store.AddFavorite(store.FavoriteRecord{
			SiteKey: site.Key,
			VodID:   det.ID,
			VodName: det.Name,
			VodPic:  det.Pic,
			Remarks: det.Remarks,
		})
		a.isFav.Set(true)
	}
	a.reloadFavorites()
}

func (a *AppState) reloadFavorites() {
	favs, err := a.store.ListFavorites()
	if err == nil {
		a.favorites.Set(favs)
	}
}

func (a *AppState) reloadHistory() {
	hist, err := a.store.ListHistory(100)
	if err == nil {
		a.history.Set(hist)
	}
}

func (a *AppState) setUseProxy(enabled bool) {
	a.useProxy.Set(enabled)
	val := "false"
	if enabled {
		val = "true"
	}
	_ = a.store.SetSetting("use_proxy", val)
}

func (a *AppState) addSite() {
	key := strings.TrimSpace(a.newSiteKey.GetUntracked())
	name := strings.TrimSpace(a.newSiteName.GetUntracked())
	api := strings.TrimSpace(a.newSiteAPI.GetUntracked())
	if api == "" {
		a.settingsMsg.Set("请填写 CMS API 地址")
		return
	}
	if key == "" {
		key = fmt.Sprintf("site-%d", time.Now().Unix())
	}
	if name == "" {
		name = key
	}
	if err := a.store.UpsertSite(cms.Site{
		Key:    key,
		Name:   name,
		API:    api,
		Active: true,
	}, len(a.sites.GetUntracked())); err != nil {
		a.settingsMsg.Set("保存失败: " + err.Error())
		return
	}
	a.newSiteKey.Set("")
	a.newSiteName.Set("")
	a.newSiteAPI.Set("")
	a.settingsMsg.Set("已添加站点: " + name)
	a.reloadSites()
	if a.selectedSiteKey.GetUntracked() == "" {
		a.selectSite(key)
	}
}

func (a *AppState) toggleSiteActive(site cms.Site, idx int) {
	site.Active = !site.Active
	_ = a.store.UpsertSite(site, idx)
	a.reloadSites()
}

func (a *AppState) deleteSite(key string) {
	_ = a.store.DeleteSite(key)
	if cur := a.searchSelectedKeys.GetUntracked(); cur[key] {
		next := make(map[string]bool, len(cur))
		for k, v := range cur {
			if v && k != key {
				next[k] = true
			}
		}
		a.searchSelectedKeys.Set(next)
	}
	a.reloadSites()
	if a.selectedSiteKey.GetUntracked() == key {
		a.selectedSiteKey.Set("")
		a.categories.Set(nil)
		a.videos.Set(nil)
		for _, s := range a.sites.GetUntracked() {
			if s.Active {
				a.selectSite(s.Key)
				break
			}
		}
	}
}

func (a *AppState) importSitesFromInput() {
	raw := strings.TrimSpace(a.importJSON.GetUntracked())
	if raw == "" {
		a.settingsMsg.Set("请粘贴站点 JSON 配置")
		return
	}
	n, err := a.store.ImportSitesJSON([]byte(raw))
	if err != nil {
		a.settingsMsg.Set("导入失败: " + err.Error())
		return
	}
	a.importJSON.Set("")
	a.settingsMsg.Set(fmt.Sprintf("成功导入 %d 个站点", n))
	a.reloadSites()
	if a.selectedSiteKey.GetUntracked() == "" {
		for _, s := range a.sites.GetUntracked() {
			if s.Active {
				a.selectSite(s.Key)
				break
			}
		}
	}
}

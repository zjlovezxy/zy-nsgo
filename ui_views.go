package main

import (
	"fmt"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"zy-nsgo/internal/cms"
)

// RootView renders the top-level navigation shell and content pages.
func RootView(cx *core.Scope, app *AppState) core.Node {
	colors := ui.ColorsOf(cx)

	sidebar := core.View{
		Style: core.Style{
			Width:        core.Pt(190),
			FlexShrink:   0,
			Padding:      core.Edges{Top: 14, Left: 8, Right: 8, Bottom: 12},
			Gap:          10,
			Borders:      core.Edges{Right: 1},
		},
		StyleFunc: func() core.Style {
			return core.Style{
				Background:   ui.Chrome(cx),
				BorderColors: core.EdgeColors{Right: colors.Line()},
			}
		},
		Children: []core.Node{
			core.View{
				Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8, Padding: core.Symmetric(4, 8)},
				Children: []core.Node{
					ui.Text("ZyNSGo", 16, ui.Bold, colors.Text),
					core.Text{Content: "CMS", Style: ui.TypeCaption, StyleFunc: ui.Colored(colors.Accent)},
				},
			},
			ui.Sidebar{
				Selected: app.tab.Get,
				OnSelect: app.switchTab,
				Sections: []ui.SidebarSection{
					{
						Items: []ui.SidebarItem{
							{ID: tabBrowse, Text: "影视库", Icon: lucide.Film},
							{ID: tabSearch, Text: "全站搜索", Icon: lucide.Search},
							{ID: tabFavorites, Text: "我的收藏", Icon: lucide.Bookmark},
							{ID: tabHistory, Text: "观看历史", Icon: lucide.Clock},
							{ID: tabSettings, Text: "源与设置", Icon: lucide.Settings},
						},
					},
				},
			},
		},
	}

	keepAlivePane := func(visible func() bool, child core.Node) core.Node {
		return core.View{
			Style: ui.FillRest,
			StyleFunc: func() core.Style {
				if visible() {
					return core.Style{Display: core.DisplayFlex}
				}
				return core.Style{Display: core.DisplayNone}
			},
			Children: []core.Node{child},
		}
	}

	mainArea := core.View{
		Style:     ui.FillRest,
		StyleFunc: ui.Filled(colors.Background),
		Children: []core.Node{
			keepAlivePane(func() bool { return !app.detailOpen.Get() && app.tab.Get() == tabBrowse }, browseView(cx, app)),
			keepAlivePane(func() bool { return !app.detailOpen.Get() && app.tab.Get() == tabSearch }, searchView(cx, app)),
			keepAlivePane(func() bool { return !app.detailOpen.Get() && app.tab.Get() == tabFavorites }, favoritesView(cx, app)),
			keepAlivePane(func() bool { return !app.detailOpen.Get() && app.tab.Get() == tabHistory }, historyView(cx, app)),
			keepAlivePane(func() bool { return !app.detailOpen.Get() && app.tab.Get() == tabSettings }, settingsView(cx, app)),
			core.Show{
				When: app.detailOpen.Get,
				Then: func(cx *core.Scope) core.Node {
					return detailPlayerView(cx, app, false)
				},
			},
		},
	}

	return core.View{
		Style: core.Style{
			Direction:  core.Row,
			AlignItems: core.AlignStretch,
			Width:      core.Pct(100),
			Height:     core.Pct(100),
		},
		Children: []core.Node{
			core.Show{
				When: func() bool { return !(app.detailOpen.Get() && app.isPlayerExpanded()) },
				Then: func(*core.Scope) core.Node { return sidebar },
			},
			mainArea,
		},
	}
}

func browseView(cx *core.Scope, app *AppState) core.Node {
	colors := ui.ColorsOf(cx)

	topBar := core.View{
		Style: core.Style{
			Direction:  core.Row,
			AlignItems: core.AlignCenter,
			Gap:        8,
			Padding:    core.Symmetric(10, 16),
			Borders:    core.Edges{Bottom: 1},
			FlexShrink: 0,
		},
		StyleFunc: func() core.Style {
			return core.Style{BorderColors: core.EdgeColors{Bottom: colors.Line()}}
		},
		Children: []core.Node{
			ui.Text("片源:", 13, ui.Medium, colors.Muted),
			ui.ScrollView{
				Horizontal: true,
				Style:      core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)},
				ContentStyle: core.Style{
					Direction:  core.Row,
					AlignItems: core.AlignCenter,
					Gap:        6,
				},
				Children: []core.Node{
					core.Dynamic(func(cx *core.Scope) core.Node {
						sites := app.activeSites()
						if len(sites) == 0 {
							return core.View{
								Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8},
								Children: []core.Node{
									ui.Text("暂无启用站点，请先添加 CMS 源", 13, ui.Regular, colors.Muted),
									ui.Button{
										Text:    "前往配置",
										Size:    ui.SizeSmall,
										OnPress: func() { app.switchTab(tabSettings) },
									},
								},
							}
						}
						selected := app.selectedSiteKey.Get()
						nodes := make([]core.Node, 0, len(sites))
						for _, s := range sites {
							site := s
							variant := ui.ButtonOutline
							if site.Key == selected {
								variant = ui.ButtonPrimary
							}
							nodes = append(nodes, ui.Button{
								Text:    site.Name,
								Size:    ui.SizeSmall,
								Variant: variant,
								OnPress: func() { app.selectSite(site.Key) },
							})
						}
						return core.Group(nodes)
					}),
				},
			},
			ui.IconButton{
				Icon:  lucide.RefreshCw,
				Label: "刷新",
				Size:  28,
				OnPress: func() {
					if s, ok := app.findSite(app.selectedSiteKey.GetUntracked()); ok {
						app.loadPage(s, app.selectedCatID.GetUntracked(), app.page.GetUntracked())
					}
				},
			},
		},
	}

	catBar := core.View{
		Style: core.Style{
			Padding:    core.Symmetric(8, 16),
			Borders:    core.Edges{Bottom: 1},
			FlexShrink: 0,
		},
		StyleFunc: func() core.Style {
			return core.Style{BorderColors: core.EdgeColors{Bottom: colors.Line()}}
		},
		Children: []core.Node{
			ui.ScrollView{
				Horizontal: true,
				ContentStyle: core.Style{
					Direction:  core.Row,
					AlignItems: core.AlignCenter,
					Gap:        6,
				},
				Children: []core.Node{
					core.Dynamic(func(cx *core.Scope) core.Node {
						cats := app.categories.Get()
						selectedCat := app.selectedCatID.Get()
						siteKey := app.selectedSiteKey.Get()
						if siteKey == "" {
							return core.Group(nil)
						}
						allVariant := ui.ButtonGhost
						if selectedCat == "" {
							allVariant = ui.ButtonSecondary
						}
						nodes := []core.Node{
							ui.Button{
								Text:    "全部",
								Size:    ui.SizeSmall,
								Variant: allVariant,
								OnPress: func() {
									if s, ok := app.findSite(siteKey); ok {
										app.loadPage(s, "", 1)
									}
								},
							},
						}
						for _, c := range cats {
							cat := c
							variant := ui.ButtonGhost
							if cat.ID == selectedCat {
								variant = ui.ButtonSecondary
							}
							nodes = append(nodes, ui.Button{
								Text:    cat.Name,
								Size:    ui.SizeSmall,
								Variant: variant,
								OnPress: func() {
									if s, ok := app.findSite(siteKey); ok {
										app.loadPage(s, cat.ID, 1)
									}
								},
							})
						}
						return core.Group(nodes)
					}),
				},
			},
		},
	}

	grid := ui.ScrollView{
		Style:        ui.FillRest,
		ContentStyle: core.Style{Padding: core.All(16), Gap: 14},
		OnScroll: func(ev core.ScrollEvent) {
			if ev.ContentSize.Height <= 0 || ev.Size.Height <= 0 {
				return
			}
			if ev.Offset.Y > 0 && ev.Offset.Y+ev.Size.Height >= ev.ContentSize.Height-100 {
				app.loadNextPage()
			}
		},
		Children: []core.Node{
			core.Show{
				When: app.loading.Get,
				Then: func(*core.Scope) core.Node {
					return core.View{
						Style:    core.Style{Padding: core.All(32), AlignItems: core.AlignCenter},
						Children: []core.Node{ui.Text("正在加载片库...", 14, ui.Medium, colors.Muted)},
					}
				},
				Else: func(cx *core.Scope) core.Node {
					return core.Group([]core.Node{
						core.Dynamic(func(cx *core.Scope) core.Node {
							if msg := app.statusMsg.Get(); msg != "" && len(app.videos.Get()) == 0 {
								return core.View{
									Style:    core.Style{Padding: core.All(32), AlignItems: core.AlignCenter},
									Children: []core.Node{ui.Text(msg, 14, ui.Regular, colors.Muted)},
								}
							}
							return core.Group(nil)
						}),
						core.View{
							Style: core.Style{Direction: core.Row, Wrap: core.Wrap, Gap: 14},
							Children: []core.Node{
								core.Each(app.videos.Get, func(v cms.VodItem) string { return v.ID }, func(cx *core.Scope, item func() cms.VodItem, _ func() int) core.Node {
									vod := item()
									return vodCard(cx, vod, func() {
										if site, ok := app.findSite(app.selectedSiteKey.GetUntracked()); ok {
											app.openDetail(site, vod.ID)
										}
									})
								}),
							},
						},
						core.Dynamic(func(cx *core.Scope) core.Node {
							items := app.videos.Get()
							if len(items) == 0 {
								return core.Group(nil)
							}
							if app.loadingMore.Get() {
								return core.View{
									Style:    core.Style{Padding: core.Symmetric(12, 0), AlignItems: core.AlignCenter},
									Children: []core.Node{ui.Text("正在加载下一页...", 13, ui.Medium, colors.Muted)},
								}
							}
							curPage := app.page.Get()
							maxPage := app.pageCount.Get()
							if curPage >= maxPage {
								return core.View{
									Style:    core.Style{Padding: core.Symmetric(12, 0), AlignItems: core.AlignCenter},
									Children: []core.Node{ui.Text(fmt.Sprintf("已到底部，共 %d 部", len(items)), 12, ui.Regular, colors.Muted)},
								}
							}
							return core.View{
								Style:    core.Style{Direction: core.Row, Padding: core.Symmetric(8, 0), JustifyContent: core.JustifyCenter, AlignItems: core.AlignCenter, Gap: 10},
								Children: []core.Node{
									ui.Text(fmt.Sprintf("已加载 %d 部 · 第 %d / %d 页（滑到底部自动加载）", len(items), curPage, maxPage), 12, ui.Regular, colors.Muted),
									ui.Button{
										Text:    "加载更多",
										Size:    ui.SizeSmall,
										Variant: ui.ButtonGhost,
										OnPress: app.loadNextPage,
									},
								},
							}
						}),
					})
				},
			},
		},
	}

	return core.View{
		Style:    ui.FillRest,
		Children: []core.Node{topBar, catBar, grid},
	}
}

func vodCard(cx *core.Scope, vod cms.VodItem, onPress func()) core.Node {
	colors := ui.ColorsOf(cx)
	return core.View{
		OnPress: func(core.PressEvent) { onPress() },
		Style: core.Style{
			Width:        core.Pt(148),
			CornerRadius: 8,
			Padding:      core.All(6),
			Gap:          6,
			Borders:      core.All(1),
		},
		StyleFunc: func() core.Style {
			return core.Style{
				Background:   colors.Surface(),
				BorderColors: core.AllColors(colors.Line()),
				Hover:        &core.Style{BorderColor: colors.Accent()},
			}
		},
		Children: []core.Node{
			ui.Image{
				Source:      core.ImageSource{URI: vod.Pic},
				AspectRatio: 0.7,
				Style:       core.Style{Width: core.Pct(100), CornerRadius: 6},
			},
			core.Text{
				Content:  vod.Name,
				MaxLines: 2,
				Style:    core.Style{FontSize: 13, FontWeight: ui.Medium},
				StyleFunc: ui.Colored(colors.Text),
			},
			core.Text{
				Content:  vod.Remarks,
				MaxLines: 1,
				Style:    ui.TypeCaption,
				StyleFunc: ui.Colored(colors.Muted),
			},
		},
	}
}

func searchView(cx *core.Scope, app *AppState) core.Node {
	colors := ui.ColorsOf(cx)

	bar := core.View{
		Style: core.Style{
			Direction:  core.Row,
			AlignItems: core.AlignCenter,
			Gap:        8,
			Padding:    core.Symmetric(12, 16),
			Borders:    core.Edges{Bottom: 1},
			FlexShrink: 0,
		},
		StyleFunc: func() core.Style {
			return core.Style{BorderColors: core.EdgeColors{Bottom: colors.Line()}}
		},
		Children: []core.Node{
			ui.Input{
				Value:       app.searchQuery.Get,
				OnChange:    func(s string) { app.searchQuery.Set(s) },
				OnSubmit:    func(s string) { app.runSearch(s) },
				Placeholder: "输入片名、演员关键字，在所有启用 CMS 源中并发搜索...",
				Style:       core.Style{FlexGrow: 1, FlexShrink: 1},
			},
			ui.Button{
				Text: "搜索",
				Icon: lucide.Search,
				Busy: app.searching.Get,
				OnPress: func() {
					app.runSearch(app.searchQuery.GetUntracked())
				},
			},
		},
	}

	scopeBar := core.View{
		Style: core.Style{
			Direction:  core.Row,
			AlignItems: core.AlignCenter,
			Gap:        8,
			Padding:    core.Symmetric(8, 16),
			Borders:    core.Edges{Bottom: 1},
			FlexShrink: 0,
		},
		StyleFunc: func() core.Style {
			return core.Style{BorderColors: core.EdgeColors{Bottom: colors.Line()}}
		},
		Children: []core.Node{
			ui.Text("搜索范围:", 13, ui.Medium, colors.Muted),
			ui.ScrollView{
				Horizontal: true,
				Style:      core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)},
				ContentStyle: core.Style{
					Direction:  core.Row,
					AlignItems: core.AlignCenter,
					Gap:        6,
				},
				Children: []core.Node{
					core.Dynamic(func(cx *core.Scope) core.Node {
						sites := app.sites.Get()
						selected := app.searchSelectedKeys.Get()
						allVariant := ui.ButtonOutline
						if len(selected) == 0 {
							allVariant = ui.ButtonPrimary
						}
						nodes := []core.Node{
							ui.Button{
								Text:    "全部启用源",
								Size:    ui.SizeSmall,
								Variant: allVariant,
								OnPress: app.selectSearchAllSites,
							},
						}
						for _, st := range sites {
							site := st
							label := site.Name
							variant := ui.ButtonOutline
							if selected[site.Key] {
								label = "✓ " + site.Name
								variant = ui.ButtonPrimary
							}
							nodes = append(nodes, ui.Button{
								Text:    label,
								Size:    ui.SizeSmall,
								Variant: variant,
								OnPress: func() { app.toggleSearchSite(site.Key) },
							})
						}
						return core.Group(nodes)
					}),
				},
			},
		},
	}

	body := ui.ScrollView{
		Style:        ui.FillRest,
		ContentStyle: core.Style{Padding: core.All(16), Gap: 18},
		Children: []core.Node{
			core.Dynamic(func(cx *core.Scope) core.Node {
				if app.searching.Get() {
					selCount := len(app.searchSelectedKeys.Get())
					msg := "正在并发搜索全部启用源..."
					if selCount > 0 {
						msg = fmt.Sprintf("正在并发搜索已选 %d 个源...", selCount)
					}
					return ui.Text(msg, 14, ui.Medium, colors.Muted)
				}
				results := app.searchResults.Get()
				if len(results) == 0 {
					return ui.Text("输入关键字开始搜索。", 13, ui.Regular, colors.Muted)
				}
				sections := make([]core.Node, 0, len(results))
				for _, r := range results {
					res := r
					header := core.View{
						Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8},
						Children: []core.Node{
							ui.Text(res.Site.Name, 15, ui.Semibold, colors.Text),
							core.Text{
								Content: func() string {
									if res.Err != nil {
										return "请求失败: " + res.Err.Error()
									}
									return fmt.Sprintf("%d 条结果", len(res.Items))
								}(),
								Style:     ui.TypeCaption,
								StyleFunc: ui.Colored(colors.Muted),
							},
						},
					}
					cards := make([]core.Node, 0, len(res.Items))
					for _, item := range res.Items {
						vod := item
						cards = append(cards, vodCard(cx, vod, func() {
							app.openDetail(res.Site, vod.ID)
						}))
					}
					sections = append(sections, core.View{
						Style: core.Style{Gap: 10},
						Children: []core.Node{
							header,
							core.View{
								Style:    core.Style{Direction: core.Row, Wrap: core.Wrap, Gap: 12},
								Children: cards,
							},
						},
					})
				}
				return core.Group(sections)
			}),
		},
	}

	return core.View{
		Style:    ui.FillRest,
		Children: []core.Node{bar, scopeBar, body},
	}
}

func favoritesView(cx *core.Scope, app *AppState) core.Node {
	colors := ui.ColorsOf(cx)
	return ui.ScrollView{
		Style:        ui.FillRest,
		ContentStyle: core.Style{Padding: core.All(16), Gap: 14},
		Children: []core.Node{
			ui.Text("我的收藏", 18, ui.Semibold, colors.Text),
			core.Dynamic(func(cx *core.Scope) core.Node {
				favs := app.favorites.Get()
				if len(favs) == 0 {
					return ui.Text("暂无收藏影片。", 13, ui.Regular, colors.Muted)
				}
				cards := make([]core.Node, 0, len(favs))
				for _, f := range favs {
					fav := f
					cards = append(cards, core.View{
						Style: core.Style{
							Width:        core.Pt(148),
							CornerRadius: 8,
							Padding:      core.All(6),
							Gap:          6,
							Borders:      core.All(1),
						},
						StyleFunc: func() core.Style {
							return core.Style{
								Background:   colors.Surface(),
								BorderColors: core.AllColors(colors.Line()),
								Hover:        &core.Style{BorderColor: colors.Accent()},
							}
						},
						Children: []core.Node{
							core.View{
								OnPress: func(core.PressEvent) {
									if site, ok := app.findSite(fav.SiteKey); ok {
										app.openDetail(site, fav.VodID)
									}
								},
								Style: core.Style{Gap: 6},
								Children: []core.Node{
									ui.Image{
										Source:      core.ImageSource{URI: fav.VodPic},
										AspectRatio: 0.7,
										Style:       core.Style{Width: core.Pct(100), CornerRadius: 6},
									},
									core.Text{
										Content:   fav.VodName,
										MaxLines:  2,
										Style:     core.Style{FontSize: 13, FontWeight: ui.Medium},
										StyleFunc: ui.Colored(colors.Text),
									},
									core.Text{
										Content:   fmt.Sprintf("[%s] %s", fav.SiteKey, fav.Remarks),
										MaxLines:  1,
										Style:     ui.TypeCaption,
										StyleFunc: ui.Colored(colors.Muted),
									},
								},
							},
							ui.Button{
								Text:    "取消收藏",
								Icon:    lucide.Trash,
								Size:    ui.SizeSmall,
								Variant: ui.ButtonGhost,
								Style:   core.Style{Width: core.Pct(100)},
								OnPress: func() {
									_ = app.store.RemoveFavorite(fav.SiteKey, fav.VodID)
									app.reloadFavorites()
								},
							},
						},
					})
				}
				return core.View{
					Style:    core.Style{Direction: core.Row, Wrap: core.Wrap, Gap: 14},
					Children: cards,
				}
			}),
		},
	}
}

func historyView(cx *core.Scope, app *AppState) core.Node {
	colors := ui.ColorsOf(cx)
	return ui.ScrollView{
		Style:        ui.FillRest,
		ContentStyle: core.Style{Padding: core.All(16), Gap: 14},
		Children: []core.Node{
			ui.Text("观看历史", 18, ui.Semibold, colors.Text),
			core.Dynamic(func(cx *core.Scope) core.Node {
				hist := app.history.Get()
				if len(hist) == 0 {
					return ui.Text("暂无观看记录。", 13, ui.Regular, colors.Muted)
				}
				cards := make([]core.Node, 0, len(hist))
				for _, h := range hist {
					rec := h
					cards = append(cards, core.View{
						Style: core.Style{
							Width:        core.Pt(148),
							CornerRadius: 8,
							Padding:      core.All(6),
							Gap:          6,
							Borders:      core.All(1),
						},
						StyleFunc: func() core.Style {
							return core.Style{
								Background:   colors.Surface(),
								BorderColors: core.AllColors(colors.Line()),
								Hover:        &core.Style{BorderColor: colors.Accent()},
							}
						},
						Children: []core.Node{
							core.View{
								OnPress: func(core.PressEvent) {
									if site, ok := app.findSite(rec.SiteKey); ok {
										app.openDetail(site, rec.VodID)
									}
								},
								Style: core.Style{Gap: 6},
								Children: []core.Node{
									ui.Image{
										Source:      core.ImageSource{URI: rec.VodPic},
										AspectRatio: 0.7,
										Style:       core.Style{Width: core.Pct(100), CornerRadius: 6},
									},
									core.Text{
										Content:   rec.VodName,
										MaxLines:  2,
										Style:     core.Style{FontSize: 13, FontWeight: ui.Medium},
										StyleFunc: ui.Colored(colors.Text),
									},
									core.Text{
										Content: fmt.Sprintf("%s · %s/%s",
											rec.EpisodeName,
											formatDuration(time.Duration(rec.PositionMS)*time.Millisecond),
											formatDuration(time.Duration(rec.DurationMS)*time.Millisecond),
										),
										MaxLines:  1,
										Style:     ui.TypeCaption,
										StyleFunc: ui.Colored(colors.Muted),
									},
								},
							},
							ui.Button{
								Text:    "删除记录",
								Icon:    lucide.Trash,
								Size:    ui.SizeSmall,
								Variant: ui.ButtonGhost,
								Style:   core.Style{Width: core.Pct(100)},
								OnPress: func() {
									_ = app.store.DeleteHistory(rec.SiteKey, rec.VodID)
									app.reloadHistory()
								},
							},
						},
					})
				}
				return core.View{
					Style:    core.Style{Direction: core.Row, Wrap: core.Wrap, Gap: 14},
					Children: cards,
				}
			}),
		},
	}
}

func settingsView(cx *core.Scope, app *AppState) core.Node {
	colors := ui.ColorsOf(cx)
	cardStyle := core.Style{
		Padding:      core.All(14),
		CornerRadius: 8,
		Borders:      core.All(1),
		Gap:          10,
	}
	cardStyleFn := func() core.Style {
		return core.Style{Background: colors.Surface(), BorderColors: core.AllColors(colors.Line())}
	}

	return ui.ScrollView{
		Style:        ui.FillRest,
		ContentStyle: core.Style{Padding: core.All(16), Gap: 16},
		Children: []core.Node{
			ui.Text("源与播放设置", 18, ui.Semibold, colors.Text),

			// Proxy switch
			core.View{
				Style:     cardStyle,
				StyleFunc: cardStyleFn,
				Children: []core.Node{
					core.View{
						Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, JustifyContent: core.JustifySpaceBetween},
						Children: []core.Node{
							core.View{
								Style: core.Style{Gap: 2, FlexGrow: 1, FlexShrink: 1},
								Children: []core.Node{
									ui.Text("启用本地 m3u8 去广告代理", 14, ui.Medium, colors.Text),
									ui.Text("自动剥离 #EXT-X-DISCONTINUITY 贴片广告并修复伪装图片头 TS 切片", 12, ui.Regular, colors.Muted),
								},
							},
							ui.Switch{
								Checked:  app.useProxy.Get,
								OnChange: app.setUseProxy,
								Label:    "启用本地代理",
							},
						},
					},
				},
			},

			// Add single site
			core.View{
				Style:     cardStyle,
				StyleFunc: cardStyleFn,
				Children: []core.Node{
					ui.Text("添加 T1 (XML) / T2 (JSON) CMS 站点", 14, ui.Semibold, colors.Text),
					core.View{
						Style: core.Style{Direction: core.Row, Gap: 8},
						Children: []core.Node{
							ui.Input{
								Value:       app.newSiteKey.Get,
								OnChange:    func(s string) { app.newSiteKey.Set(s) },
								Placeholder: "标识 Key (可选)",
								Style:       core.Style{Width: core.Pt(140)},
							},
							ui.Input{
								Value:       app.newSiteName.Get,
								OnChange:    func(s string) { app.newSiteName.Set(s) },
								Placeholder: "站点名称 (如: 量子资源)",
								Style:       core.Style{Width: core.Pt(180)},
							},
							ui.Input{
								Value:       app.newSiteAPI.Get,
								OnChange:    func(s string) { app.newSiteAPI.Set(s) },
								Placeholder: "https://.../api.php/provide/vod/",
								Style:       core.Style{FlexGrow: 1, FlexShrink: 1},
							},
							ui.Button{
								Text:    "添加",
								Icon:    lucide.Plus,
								OnPress: app.addSite,
							},
						},
					},
					ui.Textf(app.settingsMsg.Get, 12, ui.Regular, colors.Accent),
				},
			},

			// Bulk JSON import
			core.View{
				Style:     cardStyle,
				StyleFunc: cardStyleFn,
				Children: []core.Node{
					ui.Text("批量导入站点 JSON（兼容 zyfun 导出格式或站点数组）", 14, ui.Semibold, colors.Text),
					ui.Input{
						Value:       app.importJSON.Get,
						OnChange:    func(s string) { app.importJSON.Set(s) },
						Multiline:   true,
						Placeholder: `[{"key":"lz","name":"量子","api":"https://cj.lziapi.com/api.php/provide/vod/","active":true}]`,
						Style:       core.Style{Height: core.Pt(90)},
					},
					core.View{
						Style: core.Style{Direction: core.Row, JustifyContent: core.JustifyEnd},
						Children: []core.Node{
							ui.Button{
								Text:    "导入 JSON",
								Variant: ui.ButtonSecondary,
								OnPress: app.importSitesFromInput,
							},
						},
					},
				},
			},

			// Sites list
			core.View{
				Style:     cardStyle,
				StyleFunc: cardStyleFn,
				Children: []core.Node{
					ui.Text("已配置站点", 14, ui.Semibold, colors.Text),
					core.Dynamic(func(cx *core.Scope) core.Node {
						sites := app.sites.Get()
						if len(sites) == 0 {
							return ui.Text("尚未配置任何 CMS 站点。", 13, ui.Regular, colors.Muted)
						}
						rows := make([]core.Node, 0, len(sites))
						for i, s := range sites {
							idx := i
							site := s
							rows = append(rows, core.View{
								Style: core.Style{
									Direction:      core.Row,
									AlignItems:     core.AlignCenter,
									JustifyContent: core.JustifySpaceBetween,
									Padding:        core.Symmetric(8, 10),
									CornerRadius:   6,
									Borders:        core.All(1),
									Gap:            10,
								},
								StyleFunc: func() core.Style {
									return core.Style{BorderColors: core.AllColors(colors.Line())}
								},
								Children: []core.Node{
									core.View{
										Style: core.Style{Gap: 2, FlexGrow: 1, FlexShrink: 1},
										Children: []core.Node{
											ui.Text(fmt.Sprintf("%s (%s)", site.Name, site.Key), 13, ui.Medium, colors.Text),
											ui.Text(site.API, 12, ui.Regular, colors.Muted),
										},
									},
									ui.Switch{
										Checked:  func() bool { return site.Active },
										OnChange: func(bool) { app.toggleSiteActive(site, idx) },
										Label:    "启用 " + site.Name,
									},
									ui.IconButton{
										Icon:    lucide.Trash,
										Label:   "删除站点",
										Size:    28,
										OnPress: func() { app.deleteSite(site.Key) },
									},
								},
							})
						}
						return core.Group(rows)
					}),
				},
			},
		},
	}
}

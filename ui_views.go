package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"

	"zy-nsgo/internal/cms"
)

// RootView renders the top-level navigation shell and content pages.
func RootView(cx *core.Scope, app *AppState) core.Node {
	colors := ui.ColorsOf(cx)

	brandHeader := core.View{
		Style: core.Style{
			Direction:  core.Row,
			AlignItems: core.AlignCenter,
			Gap:        10,
			Padding:    core.Edges{Top: 6, Left: 8, Right: 8, Bottom: 8},
		},
		Children: []core.Node{
			core.View{
				Style: core.Style{
					Width:          core.Pt(32),
					Height:         core.Pt(32),
					CornerRadius:   9,
					AlignItems:     core.AlignCenter,
					JustifyContent: core.JustifyCenter,
					FlexShrink:     0,
					Shadows:        []core.BoxShadow{ui.Shadow(0, 3, 10, 0, core.RGBA(0x000000, 0.25))},
				},
				StyleFunc: func() core.Style {
					return core.Style{
						Background: colors.AccentFill(),
						Gradients: []core.Gradient{
							core.LinearGradient(135, colors.AccentFill(), ui.Mix(colors.AccentFill(), core.RGB(0x000000), 0.18)),
						},
					}
				},
				Children: []core.Node{
					ui.Icon(lucide.Film, 16, colors.OnAccent),
				},
			},
			core.View{
				Style: core.Style{Gap: 1, FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)},
				Children: []core.Node{
					ui.Text("ZyNSGo", 15, ui.Bold, colors.Text),
					ui.Text("私人影院", 11, ui.Regular, colors.Muted),
				},
			},
		},
	}

	sidebarNav := core.View{
		Style: ui.FillRest,
		Children: []core.Node{
			ui.Sidebar{
				Selected: app.tab.Get,
				OnSelect: app.switchTab,
				Sections: []ui.SidebarSection{
					{
						Title: "发现",
						Items: []ui.SidebarItem{
							{ID: tabBrowse, Text: "影视库", Icon: lucide.Film},
							{ID: tabSearch, Text: "全站搜索", Icon: lucide.Search},
						},
					},
					{
						Title: "片单",
						Items: []ui.SidebarItem{
							{ID: tabFavorites, Text: "我的收藏", Icon: lucide.Bookmark},
							{ID: tabHistory, Text: "观看历史", Icon: lucide.Clock},
						},
					},
					{
						Title: "系统",
						Items: []ui.SidebarItem{
							{ID: tabSettings, Text: "源与设置", Icon: lucide.Settings},
						},
					},
				},
			},
		},
	}

	sidebarFooter := core.View{
		Style: core.Style{
			Gap:        10,
			Padding:    core.Edges{Top: 10, Left: 4, Right: 4, Bottom: 2},
			Borders:    core.Edges{Top: 1},
			FlexShrink: 0,
		},
		StyleFunc: func() core.Style {
			return core.Style{BorderColors: core.EdgeColors{Top: colors.Line()}}
		},
		Children: []core.Node{
			core.View{
				Style: core.Style{
					Direction:  core.Row,
					AlignItems: core.AlignCenter,
					Gap:        7,
					Padding:    core.Symmetric(0, 6),
				},
				Children: []core.Node{
					core.View{
						Style: core.Style{
							Width:        core.Pt(7),
							Height:       core.Pt(7),
							CornerRadius: 999,
							FlexShrink:   0,
						},
						StyleFunc: func() core.Style {
							if app.useProxy.Get() {
								return core.Style{Background: colors.Up()}
							}
							return core.Style{Background: colors.Faint()}
						},
					},
					ui.Textf(func() string {
						if app.useProxy.Get() {
							return "去广告代理运行中"
						}
						return "去广告代理已关闭"
					}, 11.5, ui.Medium, colors.Muted),
				},
			},
			ui.Tabs[string]{
				Variant: ui.TabsSegmented,
				Equal:   true,
				Size:    ui.SizeSmall,
				Label:   "主题外观",
				Value:   app.appearance.Get,
				OnChange: func(v string) {
					app.setAppearance(v)
				},
				Options: []ui.Option[string]{
					{Value: appearanceDark, Label: "深色"},
					{Value: appearanceLight, Label: "浅色"},
					{Value: appearanceSystem, Label: "自动"},
				},
				Style: core.Style{Width: core.Pct(100)},
			},
		},
	}

	sidebar := core.View{
		Style: core.Style{
			Width:      core.Pt(216),
			FlexShrink: 0,
			Padding:    core.Edges{Top: 14, Left: 10, Right: 10, Bottom: 12},
			Gap:        8,
			Borders:    core.Edges{Right: 1},
		},
		StyleFunc: func() core.Style {
			return core.Style{
				Background:   ui.Chrome(cx),
				BorderColors: core.EdgeColors{Right: colors.Line()},
			}
		},
		Children: []core.Node{brandHeader, sidebarNav, sidebarFooter},
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

	header := core.View{
		Style: core.Style{
			Direction:      core.Row,
			AlignItems:     core.AlignCenter,
			JustifyContent: core.JustifySpaceBetween,
			Gap:            12,
			Padding:        core.Edges{Top: 18, Left: 24, Right: 24, Bottom: 10},
			FlexShrink:     0,
		},
		Children: []core.Node{
			core.View{
				Style: core.Style{Gap: 2, FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)},
				Children: []core.Node{
					ui.Textf(func() string {
						if s, ok := app.findSite(app.selectedSiteKey.Get()); ok && s.Name != "" {
							return s.Name
						}
						return "影视库"
					}, 20, ui.Bold, colors.Text),
					ui.Textf(func() string {
						if app.selectedSiteKey.Get() == "" {
							return "配置 CMS 片源后即可浏览最新片库"
						}
						n := len(app.videos.Get())
						if app.loading.Get() {
							return "正在刷新片库..."
						}
						return fmt.Sprintf("已加载 %d 部 · 第 %d / %d 页", n, app.page.Get(), app.pageCount.Get())
					}, 12.5, ui.Regular, colors.Muted),
				},
			},
			core.View{
				Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8},
				Children: []core.Node{
					ui.Button{
						Text:    "搜索影片",
						Icon:    lucide.Search,
						Size:    ui.SizeSmall,
						Variant: ui.ButtonOutline,
						OnPress: func() { app.switchTab(tabSearch) },
					},
					ui.IconButton{
						Icon:  lucide.RefreshCw,
						Label: "刷新当前片库",
						Size:  30,
						OnPress: func() {
							if s, ok := app.findSite(app.selectedSiteKey.GetUntracked()); ok {
								app.loadPage(s, app.selectedCatID.GetUntracked(), app.page.GetUntracked())
							}
						},
					},
				},
			},
		},
	}

	filterArea := core.View{
		Style: core.Style{
			Gap:        10,
			Padding:    core.Edges{Top: 4, Left: 24, Right: 24, Bottom: 14},
			Borders:    core.Edges{Bottom: 1},
			FlexShrink: 0,
		},
		StyleFunc: func() core.Style {
			return core.Style{BorderColors: core.EdgeColors{Bottom: colors.Line()}}
		},
		Children: []core.Node{
			// Site selector strip
			core.View{
				Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10},
				Children: []core.Node{
					ui.Text("片源", 12, ui.Semibold, colors.Muted),
					ui.ScrollView{
						Horizontal: true,
						Style:      core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)},
						ContentStyle: core.Style{
							Direction:  core.Row,
							AlignItems: core.AlignCenter,
							Gap:        8,
						},
						Children: []core.Node{
							core.Dynamic(func(cx *core.Scope) core.Node {
								sites := app.activeSites()
								if len(sites) == 0 {
									return core.View{
										Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10},
										Children: []core.Node{
											ui.Text("暂无启用片源", 12.5, ui.Regular, colors.Muted),
											ui.Button{
												Text:    "前往添加片源",
												Icon:    lucide.Plus,
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
									nodes = append(nodes, filterPill(cx, site.Name, site.Key == selected, true, func() {
										app.selectSite(site.Key)
									}))
								}
								return core.Group(nodes)
							}),
						},
					},
				},
			},
			// Category strip
			core.Dynamic(func(cx *core.Scope) core.Node {
				siteKey := app.selectedSiteKey.Get()
				if siteKey == "" {
					return core.Group(nil)
				}
				cats := app.categories.Get()
				selectedCat := app.selectedCatID.Get()
				nodes := make([]core.Node, 0, len(cats)+1)
				nodes = append(nodes, filterPill(cx, "全部", selectedCat == "", false, func() {
					if s, ok := app.findSite(siteKey); ok {
						app.loadPage(s, "", 1)
					}
				}))
				for _, c := range cats {
					cat := c
					nodes = append(nodes, filterPill(cx, cat.Name, cat.ID == selectedCat, false, func() {
						if s, ok := app.findSite(siteKey); ok {
							app.loadPage(s, cat.ID, 1)
						}
					}))
				}
				return core.View{
					Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10},
					Children: []core.Node{
						ui.Text("分类", 12, ui.Semibold, colors.Muted),
						ui.ScrollView{
							Horizontal: true,
							Style:      core.Style{FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)},
							ContentStyle: core.Style{
								Direction:  core.Row,
								AlignItems: core.AlignCenter,
								Gap:        6,
							},
							Children: nodes,
						},
					},
				}
			}),
		},
	}

	grid := ui.ScrollView{
		Style:        ui.FillRest,
		ContentStyle: core.Style{Padding: core.Symmetric(20, 24), Gap: 20},
		OnScroll: func(ev core.ScrollEvent) {
			if ev.ContentSize.Height <= 0 || ev.Size.Height <= 0 {
				return
			}
			if ev.Offset.Y > 0 && ev.Offset.Y+ev.Size.Height >= ev.ContentSize.Height-120 {
				app.loadNextPage()
			}
		},
		Children: []core.Node{
			core.Show{
				When: app.loading.Get,
				Then: func(cx *core.Scope) core.Node {
					return posterSkeletonGrid(cx, 12)
				},
				Else: func(cx *core.Scope) core.Node {
					return core.Group([]core.Node{
						core.Dynamic(func(cx *core.Scope) core.Node {
							if len(app.activeSites()) == 0 {
								return emptyState(
									cx,
									lucide.Film,
									"尚未配置片源",
									"添加一个 Apple CMS (T1/T2) 站点或批量导入 JSON 配置，即可开始观影。",
									ui.Button{
										Text:    "去配置片源",
										Icon:    lucide.Settings,
										OnPress: func() { app.switchTab(tabSettings) },
									},
								)
							}
							if msg := app.statusMsg.Get(); msg != "" && len(app.videos.Get()) == 0 {
								return emptyState(
									cx,
									lucide.Film,
									"暂无内容",
									msg,
									ui.Button{
										Text:    "重新加载",
										Icon:    lucide.RefreshCw,
										Variant: ui.ButtonOutline,
										OnPress: func() {
											if s, ok := app.findSite(app.selectedSiteKey.GetUntracked()); ok {
												app.loadPage(s, app.selectedCatID.GetUntracked(), 1)
											}
										},
									},
								)
							}
							return core.Group(nil)
						}),
						core.View{
							Style: posterGridStyle(),
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
									Style: core.Style{
										Direction:      core.Row,
										Padding:        core.Symmetric(14, 0),
										JustifyContent: core.JustifyCenter,
										AlignItems:     core.AlignCenter,
										Gap:            10,
									},
									Children: []core.Node{
										ui.Spinner{Size: 18, Label: "正在加载下一页"},
										ui.Text("正在加载更多影片...", 13, ui.Medium, colors.Muted),
									},
								}
							}
							curPage := app.page.Get()
							maxPage := app.pageCount.Get()
							if curPage >= maxPage {
								return core.View{
									Style:    core.Style{Padding: core.Symmetric(14, 0), AlignItems: core.AlignCenter},
									Children: []core.Node{ui.Text(fmt.Sprintf("已展示全部 %d 部影片", len(items)), 12, ui.Regular, colors.Faint)},
								}
							}
							return core.View{
								Style: core.Style{
									Direction:      core.Row,
									Padding:        core.Symmetric(10, 0),
									JustifyContent: core.JustifyCenter,
									AlignItems:     core.AlignCenter,
									Gap:            12,
								},
								Children: []core.Node{
									ui.Text(fmt.Sprintf("第 %d / %d 页 · 下滑自动加载", curPage, maxPage), 12, ui.Regular, colors.Muted),
									ui.Button{
										Text:    "加载下一页",
										Size:    ui.SizeSmall,
										Variant: ui.ButtonOutline,
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
		Children: []core.Node{header, filterArea, grid},
	}
}

func formatVodMeta(vod cms.VodItem) string {
	var parts []string
	if y := strings.TrimSpace(vod.Year); y != "" && y != "0" {
		parts = append(parts, y)
	}
	if a := strings.TrimSpace(vod.Area); a != "" {
		parts = append(parts, a)
	}
	if t := strings.TrimSpace(vod.TypeName); t != "" && len(parts) < 2 {
		parts = append(parts, t)
	}
	return strings.Join(parts, " · ")
}

func vodCard(cx *core.Scope, vod cms.VodItem, onPress func()) core.Node {
	return vodPosterCard(cx, posterCardOpts{
		Title:    vod.Name,
		Pic:      vod.Pic,
		Remarks:  vod.Remarks,
		Subtitle: formatVodMeta(vod),
		OnPress:  onPress,
	})
}

func searchView(cx *core.Scope, app *AppState) core.Node {
	colors := ui.ColorsOf(cx)

	header := pageHeader(
		cx,
		"全站搜索",
		func() string {
			sel := len(app.searchSelectedKeys.Get())
			if sel > 0 {
				return fmt.Sprintf("将在已选的 %d 个片源中并发检索", sel)
			}
			return fmt.Sprintf("将在全部 %d 个启用片源中并发检索", len(app.activeSites()))
		},
		nil,
	)

	searchPanel := core.View{
		Style: core.Style{
			Gap:        12,
			Padding:    core.Edges{Top: 0, Left: 24, Right: 24, Bottom: 16},
			Borders:    core.Edges{Bottom: 1},
			FlexShrink: 0,
		},
		StyleFunc: func() core.Style {
			return core.Style{BorderColors: core.EdgeColors{Bottom: colors.Line()}}
		},
		Children: []core.Node{
			// Elevated search bar
			core.View{
				Style: core.Style{
					Direction:    core.Row,
					AlignItems:   core.AlignCenter,
					Gap:          10,
					Padding:      core.Symmetric(6, 10),
					CornerRadius: 12,
					BorderWidth:  1,
				},
				StyleFunc: func() core.Style {
					return core.Style{
						Background:  colors.Surface(),
						BorderColor: colors.Border(),
						Shadows:     []core.BoxShadow{ui.Shadow(0, 2, 10, 0, core.RGBA(0x000000, 0.08))},
					}
				},
				Children: []core.Node{
					ui.Icon(lucide.Search, 16, colors.Muted),
					ui.Input{
						Value:       app.searchQuery.Get,
						OnChange:    func(s string) { app.searchQuery.Set(s) },
						OnSubmit:    func(s string) { app.runSearch(s) },
						Placeholder: "输入片名、演员或导演关键字，按回车开始搜索...",
						Style: core.Style{
							FlexGrow:    1,
							FlexShrink:  1,
							BorderWidth: 0,
							Background:  core.Transparent,
						},
					},
					core.Show{
						When: func() bool { return app.searchQuery.Get() != "" },
						Then: func(cx *core.Scope) core.Node {
							return ui.IconButton{
								Icon:  lucide.X,
								Label: "清空关键字",
								Size:  26,
								OnPress: func() {
									app.searchQuery.Set("")
									app.searchResults.Set(nil)
								},
							}
						},
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
			},
			// Source filter pills
			core.View{
				Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10},
				Children: []core.Node{
					ui.Text("范围", 12, ui.Semibold, colors.Muted),
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
								nodes := []core.Node{
									filterPill(cx, "全部启用源", len(selected) == 0, true, app.selectSearchAllSites),
								}
								for _, st := range sites {
									site := st
									isSelected := selected[site.Key]
									label := site.Name
									if isSelected {
										label = "✓ " + site.Name
									}
									nodes = append(nodes, filterPill(cx, label, isSelected, true, func() {
										app.toggleSearchSite(site.Key)
									}))
								}
								return core.Group(nodes)
							}),
						},
					},
				},
			},
		},
	}

	body := ui.ScrollView{
		Style:        ui.FillRest,
		ContentStyle: core.Style{Padding: core.Symmetric(20, 24), Gap: 24},
		Children: []core.Node{
			core.Dynamic(func(cx *core.Scope) core.Node {
				if app.searching.Get() {
					selCount := len(app.searchSelectedKeys.Get())
					msg := "正在并发搜索全部启用片源..."
					if selCount > 0 {
						msg = fmt.Sprintf("正在并发搜索已选 %d 个片源...", selCount)
					}
					return core.View{
						Style: core.Style{Gap: 20},
						Children: []core.Node{
							core.View{
								Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10},
								Children: []core.Node{
									ui.Spinner{Size: 18, Label: msg},
									ui.Text(msg, 13.5, ui.Medium, colors.Muted),
								},
							},
							posterSkeletonGrid(cx, 6),
						},
					}
				}
				results := app.searchResults.Get()
				if len(results) == 0 {
					return emptyState(
						cx,
						lucide.Search,
						"聚合多源检索",
						"输入影片名称、演员或导演，即可在多个 CMS 片源中一键并发搜索并直接播放。",
						nil,
					)
				}
				sections := make([]core.Node, 0, len(results))
				for _, r := range results {
					res := r
					badgeVariant := ui.BadgeNeutral
					badgeText := fmt.Sprintf("%d 条结果", len(res.Items))
					if res.Err != nil {
						badgeVariant = ui.BadgeDanger
						badgeText = "请求失败"
					} else if len(res.Items) > 0 {
						badgeVariant = ui.BadgeAccent
					}

					sectionHead := core.View{
						Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10},
						Children: []core.Node{
							ui.Text(res.Site.Name, 16, ui.Bold, colors.Text),
							ui.Badge{Text: badgeText, Variant: badgeVariant},
						},
					}

					var contentNode core.Node
					if res.Err != nil {
						contentNode = ui.Text(res.Err.Error(), 12.5, ui.Regular, colors.Danger)
					} else if len(res.Items) == 0 {
						contentNode = ui.Text("该站点未匹配到相关影片", 12.5, ui.Regular, colors.Faint)
					} else {
						cards := make([]core.Node, 0, len(res.Items))
						for _, item := range res.Items {
							vod := item
							cards = append(cards, vodCard(cx, vod, func() {
								app.openDetail(res.Site, vod.ID)
							}))
						}
						contentNode = core.View{
							Style:    posterGridStyle(),
							Children: cards,
						}
					}

					sections = append(sections, core.View{
						Style:    core.Style{Gap: 12},
						Children: []core.Node{sectionHead, contentNode},
					})
				}
				return core.Group(sections)
			}),
		},
	}

	return core.View{
		Style:    ui.FillRest,
		Children: []core.Node{header, searchPanel, body},
	}
}

func favoritesView(cx *core.Scope, app *AppState) core.Node {
	header := pageHeader(
		cx,
		"我的收藏",
		func() string {
			n := len(app.favorites.Get())
			if n == 0 {
				return "收藏喜欢的剧集与电影，随时一键续看"
			}
			return fmt.Sprintf("共收藏 %d 部影片", n)
		},
		nil,
	)

	body := ui.ScrollView{
		Style:        ui.FillRest,
		ContentStyle: core.Style{Padding: core.Edges{Top: 8, Left: 24, Right: 24, Bottom: 24}, Gap: 16},
		Children: []core.Node{
			core.Dynamic(func(cx *core.Scope) core.Node {
				favs := app.favorites.Get()
				if len(favs) == 0 {
					return emptyState(
						cx,
						lucide.Bookmark,
						"还没有收藏影片",
						"在播放器顶部点击收藏按钮，即可将影片收入你的私人片单。",
						ui.Button{
							Text:    "去影视库逛逛",
							Icon:    lucide.Film,
							OnPress: func() { app.switchTab(tabBrowse) },
						},
					)
				}
				cards := make([]core.Node, 0, len(favs))
				for _, f := range favs {
					fav := f
					siteName := fav.SiteKey
					if s, ok := app.findSite(fav.SiteKey); ok && s.Name != "" {
						siteName = s.Name
					}
					cards = append(cards, vodPosterCard(cx, posterCardOpts{
						Title:        fav.VodName,
						Pic:          fav.VodPic,
						Remarks:      fav.Remarks,
						TopLeftBadge: siteName,
						Subtitle:     siteName,
						OnPress: func() {
							if site, ok := app.findSite(fav.SiteKey); ok {
								app.openDetail(site, fav.VodID)
							}
						},
						ActionIcon:  lucide.Trash,
						ActionLabel: "取消收藏",
						OnAction: func() {
							_ = app.store.RemoveFavorite(fav.SiteKey, fav.VodID)
							app.reloadFavorites()
						},
					}))
				}
				return core.View{
					Style:    posterGridStyle(),
					Children: cards,
				}
			}),
		},
	}

	return core.View{
		Style:    ui.FillRest,
		Children: []core.Node{header, body},
	}
}

func historyView(cx *core.Scope, app *AppState) core.Node {
	header := pageHeader(
		cx,
		"观看历史",
		func() string {
			n := len(app.history.Get())
			if n == 0 {
				return "自动记录播放进度，下次打开精准续播"
			}
			return fmt.Sprintf("最近观看 %d 部影片", n)
		},
		nil,
	)

	body := ui.ScrollView{
		Style:        ui.FillRest,
		ContentStyle: core.Style{Padding: core.Edges{Top: 8, Left: 24, Right: 24, Bottom: 24}, Gap: 16},
		Children: []core.Node{
			core.Dynamic(func(cx *core.Scope) core.Node {
				hist := app.history.Get()
				if len(hist) == 0 {
					return emptyState(
						cx,
						lucide.Clock,
						"暂无观看记录",
						"开始播放任意影片后，你的集数和观看进度会自动保存在这里。",
						ui.Button{
							Text:    "去影视库逛逛",
							Icon:    lucide.Film,
							OnPress: func() { app.switchTab(tabBrowse) },
						},
					)
				}
				cards := make([]core.Node, 0, len(hist))
				for _, h := range hist {
					rec := h
					var pct float32
					if rec.DurationMS > 0 && rec.PositionMS > 0 {
						pct = float32(rec.PositionMS) * 100 / float32(rec.DurationMS)
					}
					posStr := formatDuration(time.Duration(rec.PositionMS) * time.Millisecond)
					durStr := formatDuration(time.Duration(rec.DurationMS) * time.Millisecond)
					sub := fmt.Sprintf("%s / %s", posStr, durStr)
					if rec.EpisodeName != "" {
						sub = rec.EpisodeName + " · " + posStr
					}

					cards = append(cards, vodPosterCard(cx, posterCardOpts{
						Title:       rec.VodName,
						Pic:         rec.VodPic,
						Remarks:     rec.EpisodeName,
						Subtitle:    sub,
						ProgressPct: pct,
						OnPress: func() {
							if site, ok := app.findSite(rec.SiteKey); ok {
								app.openDetail(site, rec.VodID)
							}
						},
						ActionIcon:  lucide.Trash,
						ActionLabel: "删除观看记录",
						OnAction: func() {
							_ = app.store.DeleteHistory(rec.SiteKey, rec.VodID)
							app.reloadHistory()
						},
					}))
				}
				return core.View{
					Style:    posterGridStyle(),
					Children: cards,
				}
			}),
		},
	}

	return core.View{
		Style:    ui.FillRest,
		Children: []core.Node{header, body},
	}
}

func settingsView(cx *core.Scope, app *AppState) core.Node {
	colors := ui.ColorsOf(cx)

	sectionCard := func(icon func() *core.SVG, title, subtitle string, trailing core.Node, children ...core.Node) core.Node {
		headLeft := core.View{
			Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10, FlexGrow: 1, FlexShrink: 1},
			Children: []core.Node{
				core.View{
					Style: core.Style{
						Width:          core.Pt(30),
						Height:         core.Pt(30),
						CornerRadius:   8,
						AlignItems:     core.AlignCenter,
						JustifyContent: core.JustifyCenter,
						FlexShrink:     0,
					},
					StyleFunc: ui.Filled(colors.Inset),
					Children:  []core.Node{ui.Icon(icon, 15, colors.Accent)},
				},
				core.View{
					Style: core.Style{Gap: 1, FlexGrow: 1, FlexShrink: 1},
					Children: func() []core.Node {
						nodes := []core.Node{ui.Text(title, 14.5, ui.Semibold, colors.Text)}
						if subtitle != "" {
							nodes = append(nodes, ui.Text(subtitle, 12, ui.Regular, colors.Muted))
						}
						return nodes
					}(),
				},
			},
		}
		headChildren := []core.Node{headLeft}
		if trailing != nil {
			headChildren = append(headChildren, trailing)
		}
		cardChildren := make([]core.Node, 0, len(children)+1)
		cardChildren = append(cardChildren, core.View{
			Style: core.Style{
				Direction:      core.Row,
				AlignItems:     core.AlignCenter,
				JustifyContent: core.JustifySpaceBetween,
				Gap:            12,
			},
			Children: headChildren,
		})
		cardChildren = append(cardChildren, children...)

		return core.View{
			Style: core.Style{
				Padding:      core.All(18),
				CornerRadius: 12,
				BorderWidth:  1,
				Gap:          14,
				Width:        core.Pct(100),
			},
			StyleFunc: func() core.Style {
				return core.Style{
					Background:  colors.Surface(),
					BorderColor: colors.Line(),
					Shadows:     []core.BoxShadow{ui.Shadow(0, 2, 8, 0, core.RGBA(0x000000, 0.06))},
				}
			},
			Children: cardChildren,
		}
	}

	header := pageHeader(
		cx,
		"源与设置",
		func() string {
			return fmt.Sprintf("已配置 %d 个片源 · %d 个启用中", len(app.sites.Get()), len(app.activeSites()))
		},
		nil,
	)

	content := core.View{
		Style: core.Style{
			Width:      core.Pct(100),
			MaxWidth:   core.Pt(860),
			AutoMargin: core.SidesHorizontal,
			Gap:        16,
		},
		Children: []core.Node{
			// Appearance & Playback preferences
			sectionCard(
				lucide.Settings,
				"外观与播放",
				"自定义界面主题与本地去广告代理行为",
				nil,
				// Appearance row
				core.View{
					Style: core.Style{
						Direction:      core.Row,
						AlignItems:     core.AlignCenter,
						JustifyContent: core.JustifySpaceBetween,
						Gap:            16,
						Padding:        core.Symmetric(10, 12),
						CornerRadius:   9,
					},
					StyleFunc: ui.Filled(colors.Inset),
					Children: []core.Node{
						core.View{
							Style: core.Style{Gap: 2, FlexGrow: 1, FlexShrink: 1},
							Children: []core.Node{
								ui.Text("界面外观", 13.5, ui.Medium, colors.Text),
								ui.Text("默认深色影院模式，支持浅色画廊或跟随系统", 12, ui.Regular, colors.Muted),
							},
						},
						ui.Tabs[string]{
							Variant: ui.TabsSegmented,
							Size:    ui.SizeSmall,
							Label:   "界面外观",
							Value:   app.appearance.Get,
							OnChange: func(v string) {
								app.setAppearance(v)
							},
							Options: []ui.Option[string]{
								{Value: appearanceDark, Label: "深色影院"},
								{Value: appearanceLight, Label: "浅色画廊"},
								{Value: appearanceSystem, Label: "跟随系统"},
							},
						},
					},
				},
				// Proxy row
				core.View{
					Style: core.Style{
						Direction:      core.Row,
						AlignItems:     core.AlignCenter,
						JustifyContent: core.JustifySpaceBetween,
						Gap:            16,
						Padding:        core.Symmetric(10, 12),
						CornerRadius:   9,
					},
					StyleFunc: ui.Filled(colors.Inset),
					Children: []core.Node{
						core.View{
							Style: core.Style{Gap: 2, FlexGrow: 1, FlexShrink: 1},
							Children: []core.Node{
								core.View{
									Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8},
									Children: []core.Node{
										ui.Text("本地 m3u8 去广告代理", 13.5, ui.Medium, colors.Text),
										ui.Badge{
											TextFunc: func() string {
												if app.useProxy.Get() {
													return "运行中"
												}
												return "已停用"
											},
											VariantFunc: func() ui.BadgeVariant {
												if app.useProxy.Get() {
													return ui.BadgeUp
												}
												return ui.BadgeNeutral
											},
										},
									},
								},
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
			),

			// Add single site
			sectionCard(
				lucide.Plus,
				"添加 CMS 片源",
				"支持 T1 (XML) 与 T2 (JSON) 标准 Apple CMS 接口",
				nil,
				core.View{
					Style: core.Style{Gap: 10},
					Children: []core.Node{
						core.View{
							Style: core.Style{Direction: core.Row, Gap: 10},
							Children: []core.Node{
								ui.Input{
									Value:       app.newSiteKey.Get,
									OnChange:    func(s string) { app.newSiteKey.Set(s) },
									Placeholder: "唯一标识 Key（可选，留空自动生成）",
									Style:       core.Style{Width: core.Pt(220)},
								},
								ui.Input{
									Value:       app.newSiteName.Get,
									OnChange:    func(s string) { app.newSiteName.Set(s) },
									Placeholder: "站点名称（如：量子资源）",
									Style:       core.Style{FlexGrow: 1, FlexShrink: 1},
								},
							},
						},
						core.View{
							Style: core.Style{Direction: core.Row, Gap: 10, AlignItems: core.AlignCenter},
							Children: []core.Node{
								ui.Input{
									Value:       app.newSiteAPI.Get,
									OnChange:    func(s string) { app.newSiteAPI.Set(s) },
									OnSubmit:    func(string) { app.addSite() },
									Placeholder: "https://.../api.php/provide/vod/",
									Style:       core.Style{FlexGrow: 1, FlexShrink: 1},
								},
								ui.Button{
									Text:    "添加片源",
									Icon:    lucide.Plus,
									OnPress: app.addSite,
								},
							},
						},
						core.Show{
							When: func() bool { return app.settingsMsg.Get() != "" },
							Then: func(cx *core.Scope) core.Node {
								return core.View{
									Style: core.Style{
										Padding:      core.Symmetric(7, 12),
										CornerRadius: 8,
									},
									StyleFunc: func() core.Style {
										return core.Style{Background: ui.TintsOf(cx).Accent()}
									},
									Children: []core.Node{
										ui.Textf(app.settingsMsg.Get, 12.5, ui.Medium, colors.Accent),
									},
								}
							},
						},
					},
				},
			),

			// Bulk JSON import
			sectionCard(
				lucide.BookmarkPlus,
				"批量导入片源 JSON",
				"兼容 zyfun 导出格式或标准站点对象数组",
				nil,
				ui.Input{
					Value:       app.importJSON.Get,
					OnChange:    func(s string) { app.importJSON.Set(s) },
					Multiline:   true,
					Placeholder: `[{"key":"lz","name":"量子资源","api":"https://cj.lziapi.com/api.php/provide/vod/","active":true}]`,
					Style:       core.Style{Height: core.Pt(96), FontFamily: core.FontMonospace, FontSize: 12},
				},
				core.View{
					Style: core.Style{Direction: core.Row, JustifyContent: core.JustifyEnd},
					Children: []core.Node{
						ui.Button{
							Text:    "解析并导入 JSON",
							Variant: ui.ButtonSecondary,
							OnPress: app.importSitesFromInput,
						},
					},
				},
			),

			// Configured sites list
			sectionCard(
				lucide.Film,
				"已配置片源",
				"开启或关闭单个站点，或移除不再使用的片源",
				ui.Badge{
					TextFunc: func() string {
						return fmt.Sprintf("%d 个站点", len(app.sites.Get()))
					},
					Variant: ui.BadgeNeutral,
				},
				core.Dynamic(func(cx *core.Scope) core.Node {
					sites := app.sites.Get()
					if len(sites) == 0 {
						return core.View{
							Style:    core.Style{Padding: core.Symmetric(16, 0), AlignItems: core.AlignCenter},
							Children: []core.Node{ui.Text("尚未配置任何 CMS 片源，请在上方添加或导入。", 13, ui.Regular, colors.Muted)},
						}
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
								Padding:        core.Symmetric(10, 12),
								CornerRadius:   9,
								BorderWidth:    1,
								Gap:            12,
							},
							StyleFunc: func() core.Style {
								return core.Style{
									Background:  colors.Inset(),
									BorderColor: colors.Line(),
								}
							},
							Children: []core.Node{
								core.View{
									Style: core.Style{
										Width:        core.Pt(8),
										Height:       core.Pt(8),
										CornerRadius: 999,
										FlexShrink:   0,
									},
									StyleFunc: func() core.Style {
										if site.Active {
											return core.Style{Background: colors.Up()}
										}
										return core.Style{Background: colors.Faint()}
									},
								},
								core.View{
									Style: core.Style{Gap: 2, FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)},
									Children: []core.Node{
										core.View{
											Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8},
											Children: []core.Node{
												ui.Text(site.Name, 13.5, ui.Semibold, colors.Text),
												ui.Badge{Text: site.Key, Variant: ui.BadgeOutline},
											},
										},
										core.Text{
											Content:  site.API,
											MaxLines: 1,
											Style: core.Style{
												FontSize:   11.5,
												FontFamily: core.FontMonospace,
											},
											StyleFunc: ui.Colored(colors.Muted),
										},
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
					return core.View{
						Style:    core.Style{Gap: 8},
						Children: rows,
					}
				}),
			),
		},
	}

	return core.View{
		Style: ui.FillRest,
		Children: []core.Node{
			header,
			ui.ScrollView{
				Style:        ui.FillRest,
				ContentStyle: core.Style{Padding: core.Edges{Top: 4, Left: 24, Right: 24, Bottom: 32}},
				Children:     []core.Node{content},
			},
		},
	}
}

package main

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
)

func detailPlayerView(cx *core.Scope, app *AppState, standalone bool) core.Node {
	colors := ui.ColorsOf(cx)
	tints := ui.TintsOf(cx)

	cx.OnKeyWith("left", core.KeyOptions{When: core.KeyNotEditing, Repeat: true}, func() {
		app.player.SeekBy(-10 * time.Second)
	})
	cx.OnKeyWith("right", core.KeyOptions{When: core.KeyNotEditing, Repeat: true}, func() {
		app.player.SeekBy(10 * time.Second)
	})
	cx.OnKeyWith("shift+left", core.KeyOptions{When: core.KeyNotEditing, Repeat: true}, func() {
		app.player.SeekBy(-30 * time.Second)
	})
	cx.OnKeyWith("shift+right", core.KeyOptions{When: core.KeyNotEditing, Repeat: true}, func() {
		app.player.SeekBy(30 * time.Second)
	})
	cx.OnKeyWith("up", core.KeyOptions{When: core.KeyNotEditing, Repeat: true}, func() {
		app.player.AdjustVolume(0.1)
	})
	cx.OnKeyWith("down", core.KeyOptions{When: core.KeyNotEditing, Repeat: true}, func() {
		app.player.AdjustVolume(-0.1)
	})
	cx.OnKeyWith("m", core.KeyOptions{When: core.KeyNotEditing}, func() {
		app.player.ToggleMute()
	})
	cx.OnKeyWith("space", core.KeyOptions{When: core.KeyNotEditing}, func() {
		app.player.TogglePause()
	})
	cx.OnKeyWith("f", core.KeyOptions{When: core.KeyNotEditing}, func() {
		app.toggleWindowFullScreen()
	})
	cx.OnKeyWith("t", core.KeyOptions{When: core.KeyNotEditing}, func() {
		app.togglePageFullScreen()
	})
	cx.OnKeyWith("shift+f", core.KeyOptions{When: core.KeyNotEditing}, func() {
		app.togglePageFullScreen()
	})
	cx.OnKeyWith("escape", core.KeyOptions{When: core.KeyAlways}, func() {
		if app.fullScreenMode.GetUntracked() != fsNone {
			app.exitPlayerFullScreen()
		} else {
			app.closeDetail()
		}
	})

	currentEpisodeName := func() string {
		det := app.detail.Get()
		if det == nil {
			return ""
		}
		li := app.activeLineIdx.Get()
		ei := app.activeEpIdx.Get()
		if li >= 0 && li < len(det.Sources) && ei >= 0 && ei < len(det.Sources[li].Episodes) {
			return det.Sources[li].Episodes[ei].Name
		}
		return ""
	}

	topBar := core.View{
		Style: core.Style{
			Direction:      core.Row,
			AlignItems:     core.AlignCenter,
			JustifyContent: core.JustifySpaceBetween,
			Padding:        core.Symmetric(10, 16),
			Borders:        core.Edges{Bottom: 1},
			FlexShrink:     0,
			Gap:            12,
		},
		StyleFunc: func() core.Style {
			return core.Style{
				Background:   colors.Surface(),
				BorderColors: core.EdgeColors{Bottom: colors.Line()},
			}
		},
		Children: []core.Node{
			core.View{
				Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10, FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)},
				Children: []core.Node{
					ui.Button{
						Text: func() string {
							if standalone {
								return "关闭"
							}
							return "返回"
						}(),
						Icon:    lucide.ArrowLeft,
						Size:    ui.SizeSmall,
						Variant: ui.ButtonGhost,
						OnPress: app.closeDetail,
					},
					core.Text{
						Content:  "",
						MaxLines: 1,
						Style:    core.Style{Display: core.DisplayNone},
					},
					ui.Textf(func() string {
						if d := app.detail.Get(); d != nil && d.Name != "" {
							return d.Name
						}
						return "正在加载影片..."
					}, 15.5, ui.Bold, colors.Text),
					core.Dynamic(func(cx *core.Scope) core.Node {
						ep := currentEpisodeName()
						site := app.detailSite.Get()
						var badges []core.Node
						if ep != "" {
							badges = append(badges, ui.Badge{Text: ep, Variant: ui.BadgeAccent})
						}
						if site.Name != "" {
							badges = append(badges, ui.Badge{Text: site.Name, Variant: ui.BadgeOutline})
						}
						if len(badges) == 0 {
							return core.Group(nil)
						}
						return core.View{
							Style:    core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6, FlexShrink: 0},
							Children: badges,
						}
					}),
				},
			},
			core.View{
				Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 8, FlexShrink: 0},
				Children: []core.Node{
					core.Dynamic(func(cx *core.Scope) core.Node {
						favText := "收藏"
						variant := ui.ButtonOutline
						if app.isFav.Get() {
							favText = "已收藏"
							variant = ui.ButtonPrimary
						}
						return ui.Button{
							Text:    favText,
							Icon:    lucide.Heart,
							Size:    ui.SizeSmall,
							Variant: variant,
							OnPress: app.toggleFavorite,
						}
					}),
					core.Dynamic(func(cx *core.Scope) core.Node {
						variant := ui.ButtonOutline
						if app.playerPanelOpen.Get() {
							variant = ui.ButtonSecondary
						}
						return ui.Button{
							Text:    "选集",
							Icon:    lucide.PanelRight,
							Size:    ui.SizeSmall,
							Variant: variant,
							OnPress: app.togglePlayerPanel,
						}
					}),
				},
			},
		},
	}

	videoStage := core.View{
		Style: core.Style{
			FlexGrow:   1,
			FlexShrink: 1,
			MinHeight:  core.Pt(260),
			Width:      core.Pct(100),
			Background: core.RGB(0x07080a),
		},
		Children: []core.Node{app.player.View(cx)},
	}

	errorBanner := core.Dynamic(func(cx *core.Scope) core.Node {
		errMsg := app.player.Error.Get()
		if errMsg == "" {
			return core.Group(nil)
		}
		return core.View{
			Style: core.Style{
				Direction:  core.Row,
				AlignItems: core.AlignCenter,
				Gap:        8,
				Padding:    core.Symmetric(8, 16),
				FlexShrink: 0,
			},
			StyleFunc: func() core.Style {
				return core.Style{Background: tints.Danger()}
			},
			Children: []core.Node{
				ui.Icon(lucide.Info, 15, colors.Danger),
				ui.Text("播放提示: "+errMsg, 12.5, ui.Medium, colors.Danger),
			},
		}
	})

	rateSelect := ui.Select[float32]{
		Label: "倍速",
		Value: app.player.Rate.Get,
		OnChange: func(v float32) {
			app.player.SetRate(v)
		},
		Options: []ui.Option[float32]{
			{Value: 1.0, Label: "1.0x"},
			{Value: 1.25, Label: "1.25x"},
			{Value: 1.5, Label: "1.5x"},
			{Value: 2.0, Label: "2.0x"},
		},
		Style: core.Style{Width: core.Pt(80)},
	}

	volumeControls := core.View{
		Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6},
		Children: []core.Node{
			core.Dynamic(func(cx *core.Scope) core.Node {
				icon := lucide.Volume2
				label := "静音"
				if app.player.Muted.Get() {
					icon = lucide.VolumeX
					label = "取消静音"
				}
				return ui.IconButton{
					Icon:    icon,
					Label:   label,
					Size:    26,
					OnPress: app.player.ToggleMute,
				}
			}),
			ui.Slider{
				Label: "音量",
				Min:   0,
				Max:   100,
				Step:  5,
				Value: func() float64 {
					if app.player.Muted.Get() {
						return 0
					}
					return math.Round(float64(app.player.Volume.Get() * 100))
				},
				OnChange: func(v float64) {
					app.player.SetVolume(float32(v / 100.0))
				},
				Style: core.Style{Width: core.Pt(82)},
			},
			ui.Tabular(ui.Textf(func() string {
				if app.player.Muted.Get() {
					return "静音"
				}
				return fmt.Sprintf("%d%%", int(math.Round(float64(app.player.Volume.Get()*100))))
			}, 11.5, ui.Medium, colors.Muted)),
		},
	}

	scrubber := core.Dynamic(func(cx *core.Scope) core.Node {
		dur := app.player.Duration.Get()
		maxSec := dur.Seconds()
		if maxSec <= 1 {
			return core.Group(nil)
		}
		return core.View{
			Style: core.Style{Width: core.Pct(100), Padding: core.Edges{Bottom: 2}},
			Children: []core.Node{
				ui.Slider{
					Label: "播放进度",
					Min:   0,
					Max:   maxSec,
					Step:  1,
					Value: func() float64 {
						sec := app.player.Position.Get().Seconds()
						if sec < 0 {
							return 0
						}
						if sec > maxSec {
							return maxSec
						}
						return sec
					},
					OnChange: func(sec float64) {
						app.player.Seek(time.Duration(sec * float64(time.Second)))
					},
					Style: core.Style{Width: core.Pct(100)},
				},
			},
		}
	})

	controlBar := core.View{
		Style: core.Style{
			Padding:    core.Edges{Top: 8, Left: 16, Right: 16, Bottom: 10},
			Borders:    core.Edges{Top: 1},
			FlexShrink: 0,
			Gap:        6,
		},
		StyleFunc: func() core.Style {
			return core.Style{
				Background:   colors.Surface(),
				BorderColors: core.EdgeColors{Top: colors.Line()},
			}
		},
		Children: []core.Node{
			scrubber,
			core.View{
				Style: core.Style{
					Direction:      core.Row,
					AlignItems:     core.AlignCenter,
					JustifyContent: core.JustifySpaceBetween,
					Gap:            10,
				},
				Children: []core.Node{
					// Left playback controls
					core.View{
						Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6},
						Children: []core.Node{
							core.Dynamic(func(cx *core.Scope) core.Node {
								if app.player.Paused.Get() {
									return ui.Button{
										Text:    "播放",
										Icon:    lucide.Play,
										Size:    ui.SizeSmall,
										OnPress: app.player.TogglePause,
									}
								}
								return ui.Button{
									Text:    "暂停",
									Icon:    lucide.Pause,
									Size:    ui.SizeSmall,
									Variant: ui.ButtonSecondary,
									OnPress: app.player.TogglePause,
								}
							}),
							ui.IconButton{
								Icon:    lucide.RotateCcw,
								Label:   "快退 10 秒",
								Size:    28,
								OnPress: func() { app.player.SeekBy(-10 * time.Second) },
							},
							ui.IconButton{
								Icon:    lucide.RotateCw,
								Label:   "快进 10 秒",
								Size:    28,
								OnPress: func() { app.player.SeekBy(10 * time.Second) },
							},
							core.Dynamic(func(cx *core.Scope) core.Node {
								det := app.detail.Get()
								if det == nil {
									return core.Group(nil)
								}
								li := app.activeLineIdx.Get()
								ei := app.activeEpIdx.Get()
								if li < 0 || li >= len(det.Sources) || len(det.Sources[li].Episodes) <= 1 {
									return core.Group(nil)
								}
								epCount := len(det.Sources[li].Episodes)
								return core.View{
									Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 2},
									Children: []core.Node{
										ui.IconButton{
											Icon:     lucide.SkipBack,
											Label:    "上一集",
											Size:     28,
											Disabled: func() bool { return ei <= 0 },
											OnPress:  func() { app.playEpisode(li, ei-1) },
										},
										ui.IconButton{
											Icon:     lucide.SkipForward,
											Label:    "下一集",
											Size:     28,
											Disabled: func() bool { return ei+1 >= epCount },
											OnPress:  func() { app.playEpisode(li, ei+1) },
										},
									},
								}
							}),
							core.View{
								Style: core.Style{Padding: core.Symmetric(0, 6)},
								Children: []core.Node{
									ui.Tabular(ui.Textf(func() string {
										pos := formatDuration(app.player.Position.Get())
										dur := formatDuration(app.player.Duration.Get())
										return pos + " / " + dur
									}, 12, ui.Medium, colors.Muted)),
								},
							},
							core.Show{
								When: app.player.Buffering.Get,
								Then: func(cx *core.Scope) core.Node {
									return core.View{
										Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6},
										Children: []core.Node{
											ui.Spinner{Size: 14, Label: "缓冲中"},
											ui.Text("缓冲中", 12, ui.Medium, colors.Accent),
										},
									}
								},
							},
						},
					},
					// Right audio, speed & fullscreen controls
					core.View{
						Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10},
						Children: []core.Node{
							volumeControls,
							rateSelect,
							core.Dynamic(func(cx *core.Scope) core.Node {
								mode := app.fullScreenMode.Get()
								pageText := "剧场"
								pageVariant := ui.ButtonOutline
								if mode == fsPage {
									pageText = "退出剧场"
									pageVariant = ui.ButtonSecondary
								}
								winText := "全屏"
								winIcon := lucide.Maximize2
								winVariant := ui.ButtonOutline
								if mode == fsWindow {
									winText = "退出全屏"
									winIcon = lucide.Minimize2
									winVariant = ui.ButtonSecondary
								}
								return core.View{
									Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 6},
									Children: []core.Node{
										ui.Button{
											Text:    pageText,
											Icon:    lucide.AppWindow,
											Size:    ui.SizeSmall,
											Variant: pageVariant,
											OnPress: app.togglePageFullScreen,
										},
										ui.Button{
											Text:    winText,
											Icon:    winIcon,
											Size:    ui.SizeSmall,
											Variant: winVariant,
											OnPress: app.toggleWindowFullScreen,
										},
									},
								}
							}),
						},
					},
				},
			},
		},
	}

	episodeTile := func(cx *core.Scope, label string, active bool, onPress func()) core.Node {
		return core.View{
			OnPress: func(core.PressEvent) { onPress() },
			Style: core.Style{
				AlignItems:     core.AlignCenter,
				JustifyContent: core.JustifyCenter,
				Padding:        core.Symmetric(8, 8),
				CornerRadius:   8,
				BorderWidth:    1,
				Cursor:         core.CursorPointer,
				Transition: core.Transition{
					Duration:   120 * time.Millisecond,
					Easing:     core.EaseOut,
					Properties: core.TransitionBackground | core.TransitionBorder,
				},
			},
			StyleFunc: func() core.Style {
				if active {
					return core.Style{
						Background:  colors.AccentFill(),
						BorderColor: colors.AccentFill(),
					}
				}
				return core.Style{
					Background:  colors.Inset(),
					BorderColor: colors.Line(),
					Hover: &core.Style{
						Background:  colors.Hover(),
						BorderColor: ui.Alpha(colors.Accent(), 0.5),
					},
				}
			},
			Children: []core.Node{
				core.Text{
					Content:  label,
					MaxLines: 1,
					Style: core.Style{
						FontSize: 12.5,
						FontWeight: func() core.FontWeight {
							if active {
								return ui.Semibold
							}
							return ui.Medium
						}(),
					},
					StyleFunc: func() core.Style {
						if active {
							return core.Style{Color: colors.OnAccent()}
						}
						return core.Style{Color: colors.Text()}
					},
				},
			},
		}
	}

	sideDrawer := core.View{
		Style: core.Style{
			Width:      core.Pt(328),
			FlexShrink: 0,
			Borders:    core.Edges{Left: 1},
		},
		StyleFunc: func() core.Style {
			return core.Style{
				Background:   colors.Surface(),
				BorderColors: core.EdgeColors{Left: colors.Line()},
			}
		},
		Children: []core.Node{
			// Drawer tab bar
			core.View{
				Style: core.Style{
					Padding:    core.Symmetric(12, 14),
					Borders:    core.Edges{Bottom: 1},
					FlexShrink: 0,
				},
				StyleFunc: func() core.Style {
					return core.Style{BorderColors: core.EdgeColors{Bottom: colors.Line()}}
				},
				Children: []core.Node{
					ui.Tabs[string]{
						Variant: ui.TabsSegmented,
						Equal:   true,
						Size:    ui.SizeSmall,
						Label:   "播放器侧栏",
						Value:   app.playerSideTab.Get,
						OnChange: func(v string) {
							app.playerSideTab.Set(v)
						},
						Options: []ui.Option[string]{
							{Value: "episodes", Label: "选集列表"},
							{Value: "info", Label: "影片简介"},
						},
						Style: core.Style{Width: core.Pct(100)},
					},
				},
			},
			// Drawer scrollable body
			ui.ScrollView{
				Style:        ui.FillRest,
				ContentStyle: core.Style{Padding: core.All(14), Gap: 16},
				Children: []core.Node{
					core.Dynamic(func(cx *core.Scope) core.Node {
						if app.detailLoading.Get() {
							return core.View{
								Style: core.Style{Padding: core.Symmetric(40, 0), AlignItems: core.AlignCenter, Gap: 10},
								Children: []core.Node{
									ui.Spinner{Size: 22, Label: "正在获取剧集"},
									ui.Text("正在获取剧集与播放地址...", 13, ui.Medium, colors.Muted),
								},
							}
						}
						det := app.detail.Get()
						if det == nil {
							return core.Group(nil)
						}

						if app.playerSideTab.Get() == "info" {
							var tags []core.Node
							for _, val := range []string{det.TypeName, det.Year, det.Area, det.Remarks} {
								v := strings.TrimSpace(val)
								if v != "" && v != "0" {
									tags = append(tags, ui.Badge{Text: v, Variant: ui.BadgeNeutral})
								}
							}
							return core.View{
								Style: core.Style{Gap: 14},
								Children: []core.Node{
									core.View{
										Style: core.Style{Direction: core.Row, Gap: 12, AlignItems: core.AlignStart},
										Children: []core.Node{
											ui.Image{
												Source:      core.ImageSource{URI: det.Pic},
												AspectRatio: 2.0 / 3.0,
												Style:       core.Style{Width: core.Pt(86), CornerRadius: 8, FlexShrink: 0},
											},
											core.View{
												Style: core.Style{Gap: 8, FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)},
												Children: []core.Node{
													core.Text{
														Content:   det.Name,
														MaxLines:  3,
														Style:     core.Style{FontSize: 15, FontWeight: ui.Bold, LineHeightMultiple: 1.3},
														StyleFunc: ui.Colored(colors.Text),
													},
													core.View{
														Style:    core.Style{Direction: core.Row, Wrap: core.Wrap, Gap: 6},
														Children: tags,
													},
												},
											},
										},
									},
									core.View{
										Style: core.Style{
											Padding:      core.All(12),
											CornerRadius: 10,
											Gap:          8,
										},
										StyleFunc: ui.Filled(colors.Inset),
										Children: func() []core.Node {
											var rows []core.Node
											if d := strings.TrimSpace(det.Director); d != "" {
												rows = append(rows, core.View{
													Style: core.Style{Gap: 2},
													Children: []core.Node{
														ui.Text("导演", 11, ui.Semibold, colors.Faint),
														ui.Text(d, 12.5, ui.Regular, colors.Text),
													},
												})
											}
											if a := strings.TrimSpace(det.Actor); a != "" {
												rows = append(rows, core.View{
													Style: core.Style{Gap: 2},
													Children: []core.Node{
														ui.Text("主演", 11, ui.Semibold, colors.Faint),
														core.Text{
															Content:   a,
															Style:     core.Style{FontSize: 12.5, LineHeightMultiple: 1.45},
															StyleFunc: ui.Colored(colors.Text),
														},
													},
												})
											}
											return rows
										}(),
									},
									core.View{
										Style: core.Style{Gap: 6},
										Children: []core.Node{
											ui.Text("剧情简介", 13, ui.Semibold, colors.Text),
											core.Text{
												Content: func() string {
													if c := strings.TrimSpace(det.Content); c != "" {
														return c
													}
													return "暂无简介"
												}(),
												Style:     core.Style{FontSize: 12.5, LineHeightMultiple: 1.65},
												StyleFunc: ui.Colored(colors.Muted),
											},
										},
									},
								},
							}
						}

						// Episodes tab
						activeLine := app.activeLineIdx.Get()
						activeEp := app.activeEpIdx.Get()

						linePills := make([]core.Node, 0, len(det.Sources))
						for i, s := range det.Sources {
							li := i
							src := s
							label := fmt.Sprintf("%s (%d)", src.Name, len(src.Episodes))
							linePills = append(linePills, filterPill(cx, label, li == activeLine, false, func() {
								app.playEpisode(li, 0)
							}))
						}

						var epTiles []core.Node
						epTotal := 0
						if activeLine >= 0 && activeLine < len(det.Sources) {
							eps := det.Sources[activeLine].Episodes
							epTotal = len(eps)
							epTiles = make([]core.Node, 0, len(eps))
							for j, e := range eps {
								ei := j
								ep := e
								epTiles = append(epTiles, episodeTile(cx, ep.Name, ei == activeEp, func() {
									app.playEpisode(activeLine, ei)
								}))
							}
						}

						return core.View{
							Style: core.Style{Gap: 16},
							Children: []core.Node{
								core.View{
									Style: core.Style{Gap: 8},
									Children: []core.Node{
										ui.Text("播放线路", 12.5, ui.Semibold, colors.Muted),
										core.View{
											Style:    core.Style{Direction: core.Row, Wrap: core.Wrap, Gap: 6},
											Children: linePills,
										},
									},
								},
								core.View{
									Style: core.Style{Gap: 10},
									Children: []core.Node{
										core.View{
											Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, JustifyContent: core.JustifySpaceBetween},
											Children: []core.Node{
												ui.Text("选集", 12.5, ui.Semibold, colors.Muted),
												ui.Text(fmt.Sprintf("共 %d 集", epTotal), 11.5, ui.Regular, colors.Faint),
											},
										},
										core.View{
											Style: core.Style{
												Display:             core.DisplayGrid,
												GridTemplateColumns: core.GridTracks("repeat(auto-fill, minmax(76px, 1fr))"),
												Gap:                 8,
												Width:               core.Pct(100),
											},
											Children: epTiles,
										},
									},
								},
							},
						}
					}),
				},
			},
		},
	}

	leftStage := core.View{
		Style:    ui.FillRest,
		Children: []core.Node{videoStage, errorBanner, controlBar},
	}

	return core.View{
		Style: ui.FillRest,
		Children: []core.Node{
			core.Show{
				When: func() bool { return !app.isPlayerExpanded() },
				Then: func(*core.Scope) core.Node { return topBar },
			},
			core.View{
				Style: core.Style{
					Direction:  core.Row,
					AlignItems: core.AlignStretch,
					FlexGrow:   1,
					FlexShrink: 1,
					MinHeight:  core.Pt(0),
					Width:      core.Pct(100),
				},
				Children: []core.Node{
					leftStage,
					core.Show{
						When: func() bool { return !app.isPlayerExpanded() && app.playerPanelOpen.Get() },
						Then: func(*core.Scope) core.Node { return sideDrawer },
					},
				},
			},
		},
	}
}

func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	totalSec := int(d.Seconds())
	h := totalSec / 3600
	m := (totalSec % 3600) / 60
	s := totalSec % 60
	if h > 0 {
		return fmt.Sprintf("%02d:%02d:%02d", h, m, s)
	}
	return fmt.Sprintf("%02d:%02d", m, s)
}

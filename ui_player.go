package main

import (
	"fmt"
	"math"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
)

func detailPlayerView(cx *core.Scope, app *AppState, standalone bool) core.Node {
	colors := ui.ColorsOf(cx)

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

	topBar := core.View{
		Style: core.Style{
			Direction:      core.Row,
			AlignItems:     core.AlignCenter,
			JustifyContent: core.JustifySpaceBetween,
			Padding:        core.Symmetric(10, 16),
			Borders:        core.Edges{Bottom: 1},
			FlexShrink:     0,
			Gap:            10,
		},
		StyleFunc: func() core.Style {
			return core.Style{BorderColors: core.EdgeColors{Bottom: colors.Line()}}
		},
		Children: []core.Node{
			core.View{
				Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10, FlexGrow: 1, FlexShrink: 1},
				Children: []core.Node{
					ui.Button{
						Text: func() string { if standalone { return "关闭窗口" }; return "返回" }(),
						Icon:    lucide.ArrowLeft,
						Size:    ui.SizeSmall,
						Variant: ui.ButtonOutline,
						OnPress: app.closeDetail,
					},
					ui.Textf(func() string {
						if d := app.detail.Get(); d != nil {
							return d.Name
						}
						return "正在加载..."
					}, 15, ui.Semibold, colors.Text),
				},
			},
			core.Dynamic(func(cx *core.Scope) core.Node {
				favText := "收藏"
				variant := ui.ButtonOutline
				if app.isFav.Get() {
					favText = "已收藏"
					variant = ui.ButtonSecondary
				}
				return ui.Button{
					Text:    favText,
					Icon:    lucide.Heart,
					Size:    ui.SizeSmall,
					Variant: variant,
					OnPress: app.toggleFavorite,
				}
			}),
		},
	}

	videoStage := core.View{
		StyleFunc: func() core.Style {
			if app.isPlayerExpanded() {
				return core.Style{
					FlexGrow:   1,
					FlexShrink: 1,
					MinHeight:  core.Pt(0),
					Width:      core.Pct(100),
					Background: core.RGB(0x0a0a0c),
				}
			}
			return core.Style{
				Height:     core.Pt(400),
				FlexShrink: 0,
				Width:      core.Pct(100),
				Background: core.RGB(0x0a0a0c),
			}
		},
		Children: []core.Node{app.player.View(cx)},
	}

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
		Style: core.Style{Width: core.Pt(82)},
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
				Style: core.Style{Width: core.Pt(88)},
			},
			ui.Tabular(ui.Textf(func() string {
				if app.player.Muted.Get() {
					return "静音"
				}
				return fmt.Sprintf("%d%%", int(math.Round(float64(app.player.Volume.Get()*100))))
			}, 12, ui.Regular, colors.Muted)),
		},
	}

	controlBar := core.View{
		Style: core.Style{
			Direction:      core.Row,
			AlignItems:     core.AlignCenter,
			JustifyContent: core.JustifySpaceBetween,
			Padding:        core.Symmetric(8, 16),
			Borders:        core.Edges{Bottom: 1},
			FlexShrink:     0,
			Gap:            8,
		},
		StyleFunc: func() core.Style {
			return core.Style{
				Background:   colors.Surface(),
				BorderColors: core.EdgeColors{Bottom: colors.Line()},
			}
		},
		Children: []core.Node{
			core.View{
				Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10},
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
					ui.Tabular(ui.Textf(func() string {
						pos := formatDuration(app.player.Position.Get())
						dur := formatDuration(app.player.Duration.Get())
						return pos + " / " + dur
					}, 12, ui.Medium, colors.Muted)),
					core.Show{
						When: app.player.Buffering.Get,
						Then: func(cx *core.Scope) core.Node {
							return ui.Text("缓冲中...", 12, ui.Regular, colors.Accent)
						},
					},
				},
			},
			core.View{
				Style: core.Style{Direction: core.Row, AlignItems: core.AlignCenter, Gap: 10},
				Children: []core.Node{
					volumeControls,
					rateSelect,
					core.Dynamic(func(cx *core.Scope) core.Node {
						mode := app.fullScreenMode.Get()
						pageText := "页面全屏"
						pageVariant := ui.ButtonOutline
						if mode == fsPage {
							pageText = "退出页面全屏"
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
						return core.Group([]core.Node{
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
						})
					}),
				},
			},
		},
	}

	detailsAndEpisodes := ui.ScrollView{
		Style:        ui.FillRest,
		ContentStyle: core.Style{Padding: core.All(16), Gap: 14},
		Children: []core.Node{
			core.Dynamic(func(cx *core.Scope) core.Node {
				if errMsg := app.player.Error.Get(); errMsg != "" {
					return ui.Text("播放提示: "+errMsg, 13, ui.Medium, colors.Danger)
				}
				return core.Group(nil)
			}),
			core.Dynamic(func(cx *core.Scope) core.Node {
				if app.detailLoading.Get() {
					return ui.Text("正在获取剧集与播放地址...", 14, ui.Medium, colors.Muted)
				}
				det := app.detail.Get()
				if det == nil {
					return core.Group(nil)
				}
				activeLine := app.activeLineIdx.Get()
				activeEp := app.activeEpIdx.Get()

				lineButtons := make([]core.Node, 0, len(det.Sources))
				for i, s := range det.Sources {
					li := i
					src := s
					variant := ui.ButtonOutline
					if li == activeLine {
						variant = ui.ButtonPrimary
					}
					lineButtons = append(lineButtons, ui.Button{
						Text:    fmt.Sprintf("%s (%d)", src.Name, len(src.Episodes)),
						Size:    ui.SizeSmall,
						Variant: variant,
						OnPress: func() { app.playEpisode(li, 0) },
					})
				}

				var epButtons []core.Node
				if activeLine >= 0 && activeLine < len(det.Sources) {
					eps := det.Sources[activeLine].Episodes
					epButtons = make([]core.Node, 0, len(eps))
					for j, e := range eps {
						ei := j
						ep := e
						variant := ui.ButtonOutline
						if ei == activeEp {
							variant = ui.ButtonPrimary
						}
						epButtons = append(epButtons, ui.Button{
							Text:    ep.Name,
							Size:    ui.SizeSmall,
							Variant: variant,
							OnPress: func() { app.playEpisode(activeLine, ei) },
						})
					}
				}

				return core.Group([]core.Node{
					core.View{
						Style: core.Style{Gap: 8},
						Children: []core.Node{
							ui.Text("播放线路", 14, ui.Semibold, colors.Text),
							core.View{
								Style:    core.Style{Direction: core.Row, Wrap: core.Wrap, Gap: 8},
								Children: lineButtons,
							},
						},
					},
					core.View{
						Style: core.Style{Gap: 8},
						Children: []core.Node{
							ui.Text("选集列表", 14, ui.Semibold, colors.Text),
							core.View{
								Style:    core.Style{Direction: core.Row, Wrap: core.Wrap, Gap: 8},
								Children: epButtons,
							},
						},
					},
					core.View{
						Style: core.Style{
							Padding:      core.All(12),
							CornerRadius: 8,
							Borders:      core.All(1),
							Gap:          6,
						},
						StyleFunc: func() core.Style {
							return core.Style{Background: colors.Surface(), BorderColors: core.AllColors(colors.Line())}
						},
						Children: []core.Node{
							ui.Text(fmt.Sprintf("%s · %s · %s · %s", det.TypeName, det.Year, det.Area, det.Remarks), 13, ui.Medium, colors.Text),
							ui.Text("导演: "+det.Director+"    主演: "+det.Actor, 12, ui.Regular, colors.Muted),
							core.Text{
								Content:   det.Content,
								Style:     core.Style{FontSize: 12},
								StyleFunc: ui.Colored(colors.Muted),
							},
						},
					},
				})
			}),
		},
	}

	return core.View{
		Style: ui.FillRest,
		Children: []core.Node{
			core.Show{
				When: func() bool { return !app.isPlayerExpanded() },
				Then: func(*core.Scope) core.Node { return topBar },
			},
			videoStage,
			controlBar,
			core.Show{
				When: func() bool { return !app.isPlayerExpanded() },
				Then: func(*core.Scope) core.Node { return detailsAndEpisodes },
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

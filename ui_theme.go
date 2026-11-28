package main

import (
	"strings"
	"time"

	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/icons/lucide"
	"github.com/nsgo-dev/nsgo/ui"
)

const (
	appearanceDark   = "dark"
	appearanceLight  = "light"
	appearanceSystem = "system"
)

// cinemaTheme builds a curated Obsidian Cinema (dark) & Warm Gallery (light) theme.
func cinemaTheme() *core.Theme {
	t := core.DefaultTheme()
	lightInk := core.RGB(0x1b1d22)
	darkInk := core.RGB(0xf2f1ee)

	t.Radius = 10
	t.Light = core.Palette{
		Background:     core.RGB(0xf6f5f2),
		Surface:        core.RGB(0xffffff),
		SurfaceHover:   lightInk.Faded(0.06),
		SurfacePressed: lightInk.Faded(0.12),
		Border:         lightInk.Faded(0.12),
		Text:           lightInk,
		TextMuted:      lightInk.Faded(0.56),
		Accent:         core.RGB(0xd65122),
		AccentHover:    core.RGB(0xbe451b),
		AccentPressed:  core.RGB(0xa33915),
		AccentText:     core.RGB(0xffffff),
		Danger:         core.RGB(0xc83e46),
		Warning:        core.RGB(0xb46509),
		Success:        core.RGB(0x0d8f66),
		Selection:      core.RGBA(0xd65122, 0.20),
		Focus:          core.RGBA(0xd65122, 0.55),
		Scrollbar:      lightInk.Faded(0.28),
		Colors: map[string]core.Color{
			"inset":       core.RGB(0xedeae4),
			"raised":      core.RGB(0xffffff),
			"field":       core.RGB(0xffffff),
			"fieldBorder": lightInk.Faded(0.14),
			"line":        lightInk.Faded(0.09),
			"faint":       lightInk.Faded(0.36),
			"accentFill":  core.RGB(0xd65122),
			"light":       lightInk,
			"onLight":     core.RGB(0xf6f5f2),
			"track":       lightInk.Faded(0.22),
			"knob":        core.RGB(0xffffff),
			"code":        core.RGB(0x2e2f36),
			"info":        core.RGB(0x2e64d8),
			"down":        core.RGB(0xc83e46),
		},
	}

	t.Dark = core.Palette{
		Background:     core.RGB(0x111216),
		Surface:        core.RGB(0x1a1c22),
		SurfaceHover:   darkInk.Faded(0.08),
		SurfacePressed: darkInk.Faded(0.14),
		Border:         darkInk.Faded(0.12),
		Text:           darkInk,
		TextMuted:      darkInk.Faded(0.58),
		Accent:         core.RGB(0xe86f3a),
		AccentHover:    core.RGB(0xf0804e),
		AccentPressed:  core.RGB(0xcf5d2c),
		AccentText:     core.RGB(0xffffff),
		Danger:         core.RGB(0xe56b6f),
		Warning:        core.RGB(0xe9b44c),
		Success:        core.RGB(0x38b28b),
		Selection:      core.RGBA(0xe86f3a, 0.24),
		Focus:          core.RGBA(0xe86f3a, 0.72),
		Scrollbar:      darkInk.Faded(0.30),
		Colors: map[string]core.Color{
			"inset":       core.RGB(0x16181e),
			"raised":      core.RGB(0x23262f),
			"field":       core.RGB(0x14161b),
			"fieldBorder": darkInk.Faded(0.14),
			"line":        darkInk.Faded(0.08),
			"faint":       darkInk.Faded(0.34),
			"accentFill":  core.RGB(0xe86f3a),
			"light":       darkInk,
			"onLight":     core.RGB(0x111216),
			"track":       darkInk.Faded(0.22),
			"knob":        core.RGB(0xffffff),
			"code":        core.RGB(0xd8d6d0),
			"info":        core.RGB(0x5b9bf8),
			"down":        core.RGB(0xe56b6f),
		},
	}
	return t
}

func applyAppearance(scope *core.Scope, mode string) {
	if scope == nil {
		return
	}
	switch mode {
	case appearanceLight:
		scope.SetAppearance(core.AppearanceLight)
	case appearanceSystem:
		scope.SetAppearance(core.AppearanceSystem)
	default:
		scope.SetAppearance(core.AppearanceDark)
	}
}

func posterGridStyle() core.Style {
	return core.Style{
		Display:             core.DisplayGrid,
		GridTemplateColumns: core.GridTracks("repeat(auto-fill, minmax(154px, 1fr))"),
		ColumnGap:           18,
		RowGap:              22,
		Width:               core.Pct(100),
	}
}

// filterPill renders a sleek pill button for site selectors, categories, and filter chips.
func filterPill(cx *core.Scope, label string, active bool, solid bool, onPress func()) core.Node {
	colors := ui.ColorsOf(cx)
	tints := ui.TintsOf(cx)

	return core.View{
		OnPress: func(core.PressEvent) { onPress() },
		Style: core.Style{
			Direction:      core.Row,
			AlignItems:     core.AlignCenter,
			JustifyContent: core.JustifyCenter,
			Padding:        core.Symmetric(5, 13),
			CornerRadius:   999,
			BorderWidth:    1,
			FlexShrink:     0,
			Cursor:         core.CursorPointer,
			Transition: core.Transition{
				Duration:   140 * time.Millisecond,
				Easing:     core.EaseOut,
				Properties: core.TransitionBackground | core.TransitionBorder,
			},
		},
		StyleFunc: func() core.Style {
			if active {
				if solid {
					return core.Style{
						Background:  colors.AccentFill(),
						BorderColor: colors.AccentFill(),
						Hover:       &core.Style{Background: colors.Now().AccentHover, BorderColor: colors.Now().AccentHover},
					}
				}
				return core.Style{
					Background:  tints.Accent(),
					BorderColor: ui.Alpha(colors.Accent(), 0.35),
					Hover:       &core.Style{Background: ui.Alpha(colors.Accent(), 0.22)},
				}
			}
			return core.Style{
				Background:  colors.Surface(),
				BorderColor: colors.Line(),
				Hover: &core.Style{
					Background:  colors.Hover(),
					BorderColor: colors.Border(),
				},
			}
		},
		Children: []core.Node{
			core.Text{
				Content: label,
				Style: core.Style{
					FontSize: 12.5,
					FontWeight: func() core.FontWeight {
						if active {
							return ui.Semibold
						}
						return ui.Medium
					}(),
					WhiteSpace: core.WhiteSpaceNoWrap,
				},
				StyleFunc: func() core.Style {
					if active {
						if solid {
							return core.Style{Color: colors.OnAccent()}
						}
						return core.Style{Color: colors.Accent()}
					}
					return core.Style{Color: colors.Muted()}
				},
			},
		},
	}
}

// pageHeader renders a clean top title bar for content pages.
func pageHeader(cx *core.Scope, title string, subtitle func() string, trailing core.Node) core.Node {
	colors := ui.ColorsOf(cx)
	leftChildren := []core.Node{
		ui.Text(title, 20, ui.Bold, colors.Text),
	}
	if subtitle != nil {
		leftChildren = append(leftChildren, ui.Textf(subtitle, 12.5, ui.Regular, colors.Muted))
	}
	rowChildren := []core.Node{
		core.View{
			Style:    core.Style{Gap: 2, FlexGrow: 1, FlexShrink: 1, MinWidth: core.Pt(0)},
			Children: leftChildren,
		},
	}
	if trailing != nil {
		rowChildren = append(rowChildren, trailing)
	}
	return core.View{
		Style: core.Style{
			Direction:      core.Row,
			AlignItems:     core.AlignCenter,
			JustifyContent: core.JustifySpaceBetween,
			Gap:            12,
			Padding:        core.Edges{Top: 18, Left: 24, Right: 24, Bottom: 12},
			FlexShrink:     0,
		},
		Children: rowChildren,
	}
}

// emptyState renders a centered, high-craft empty state with an icon, title, subtitle, and optional CTA.
func emptyState(cx *core.Scope, icon func() *core.SVG, title, subtitle string, action core.Node) core.Node {
	colors := ui.ColorsOf(cx)
	tints := ui.TintsOf(cx)

	children := []core.Node{
		core.View{
			Style: core.Style{
				Width:          core.Pt(56),
				Height:         core.Pt(56),
				CornerRadius:   999,
				AlignItems:     core.AlignCenter,
				JustifyContent: core.JustifyCenter,
				Margin:         core.Edges{Bottom: 4},
			},
			StyleFunc: func() core.Style {
				return core.Style{Background: tints.Accent()}
			},
			Children: []core.Node{
				ui.Icon(icon, 24, colors.Accent),
			},
		},
		ui.Text(title, 16, ui.Semibold, colors.Text),
		core.Text{
			Content: subtitle,
			Style: core.Style{
				FontSize:  13,
				TextAlign: core.TextAlignCenter,
				MaxWidth:  core.Pt(360),
			},
			StyleFunc: ui.Colored(colors.Muted),
		},
	}
	if action != nil {
		children = append(children, core.View{
			Style:    core.Style{Margin: core.Edges{Top: 8}},
			Children: []core.Node{action},
		})
	}

	return core.View{
		Style: core.Style{
			Padding:        core.Symmetric(56, 24),
			AlignItems:     core.AlignCenter,
			JustifyContent: core.JustifyCenter,
			Gap:            8,
			Width:          core.Pct(100),
		},
		Children: children,
	}
}

// posterSkeletonGrid renders pulsing poster card placeholders while data loads.
func posterSkeletonGrid(cx *core.Scope, count int) core.Node {
	cards := make([]core.Node, 0, count)
	for i := range count {
		cards = append(cards, core.View{
			Style: core.Style{Gap: 8},
			Children: []core.Node{
				ui.SkeletonBlock{
					Width: core.Pct(100),
					Index: i % 6,
					Style: core.Style{AspectRatio: 2.0 / 3.0, CornerRadius: 12},
				},
				ui.SkeletonBlock{
					Width:  core.Pct(78),
					Height: core.Pt(13),
					Index:  (i % 6) + 1,
					Style:  core.Style{CornerRadius: 4},
				},
				ui.SkeletonBlock{
					Width:  core.Pct(46),
					Height: core.Pt(10),
					Index:  (i % 6) + 2,
					Style:  core.Style{CornerRadius: 4},
				},
			},
		})
	}
	return core.View{
		Style:    posterGridStyle(),
		Children: cards,
	}
}

type posterCardOpts struct {
	Title        string
	Pic          string
	Remarks      string
	TopLeftBadge string
	Subtitle     string
	ProgressPct  float32 // 0..100, >0 shows bottom progress bar
	OnPress      func()
	ActionIcon   func() *core.SVG
	ActionLabel  string
	OnAction     func()
}

// vodPosterCard renders a modern cinema poster card with hover lift, corner badges,
// optional watch progress bar, and clean metadata typography.
func vodPosterCard(cx *core.Scope, opts posterCardOpts) core.Node {
	colors := ui.ColorsOf(cx)
	remarks := strings.TrimSpace(opts.Remarks)
	topLeft := strings.TrimSpace(opts.TopLeftBadge)
	subtitle := strings.TrimSpace(opts.Subtitle)

	posterOverlays := []core.Node{
		ui.Image{
			Source:      core.ImageSource{URI: opts.Pic},
			AspectRatio: 2.0 / 3.0,
			Style:       core.Style{Width: core.Pct(100), CornerRadius: 12},
		},
		// Subtle hover scrim + centered play cue
		core.View{
			Style: core.Style{
				Position:       core.Absolute,
				Top:            core.Pt(0),
				Right:          core.Pt(0),
				Bottom:         core.Pt(0),
				Left:           core.Pt(0),
				AlignItems:     core.AlignCenter,
				JustifyContent: core.JustifyCenter,
				Background:     core.RGBA(0x000000, 0.28),
				Opacity:        core.Alpha(0),
				PointerEvents:  core.PointerEventsNone,
				Transition: core.Transition{
					Duration:   160 * time.Millisecond,
					Easing:     core.EaseOut,
					Properties: core.TransitionOpacity,
				},
				GroupStyle: &core.GroupStyle{
					Name:  "poster",
					Hover: &core.Style{Opacity: core.Alpha(1)},
				},
			},
			Children: []core.Node{
				core.View{
					Style: core.Style{
						Width:          core.Pt(44),
						Height:         core.Pt(44),
						CornerRadius:   999,
						AlignItems:     core.AlignCenter,
						JustifyContent: core.JustifyCenter,
						Shadows:        []core.BoxShadow{ui.Shadow(0, 6, 16, 0, core.RGBA(0x000000, 0.45))},
					},
					StyleFunc: func() core.Style {
						return core.Style{Background: ui.Alpha(colors.AccentFill(), 0.94)}
					},
					Children: []core.Node{
						ui.Icon(lucide.Play, 18, colors.OnAccent),
					},
				},
			},
		},
	}

	if topLeft != "" {
		posterOverlays = append(posterOverlays, core.View{
			Style: core.Style{
				Position:      core.Absolute,
				Top:           core.Pt(8),
				Left:          core.Pt(8),
				Padding:       core.Symmetric(2, 7),
				CornerRadius:  6,
				Background:    core.RGBA(0x0c0d11, 0.76),
				PointerEvents: core.PointerEventsNone,
				MaxWidth:      core.Pct(60),
			},
			Children: []core.Node{
				core.Text{
					Content:  topLeft,
					MaxLines: 1,
					Style: core.Style{
						FontSize:   10.5,
						FontWeight: ui.Medium,
						Color:      core.RGB(0xf2f1ee),
					},
				},
			},
		})
	}

	if remarks != "" {
		posterOverlays = append(posterOverlays, core.View{
			Style: core.Style{
				Position:      core.Absolute,
				Top:           core.Pt(8),
				Right:         core.Pt(8),
				Padding:       core.Symmetric(2, 7),
				CornerRadius:  6,
				Background:    core.RGBA(0x0c0d11, 0.78),
				PointerEvents: core.PointerEventsNone,
				MaxWidth:      core.Pct(72),
			},
			Children: []core.Node{
				core.Text{
					Content:  remarks,
					MaxLines: 1,
					Style: core.Style{
						FontSize:   10.5,
						FontWeight: ui.Medium,
						Color:      core.RGB(0xffffff),
					},
				},
			},
		})
	}

	if opts.ProgressPct > 0 {
		pct := opts.ProgressPct
		if pct > 100 {
			pct = 100
		}
		if pct < 2 {
			pct = 2
		}
		posterOverlays = append(posterOverlays, core.View{
			Style: core.Style{
				Position:       core.Absolute,
				Left:           core.Pt(0),
				Right:          core.Pt(0),
				Bottom:         core.Pt(0),
				Height:         core.Pt(24),
				JustifyContent: core.JustifyEnd,
				PointerEvents:  core.PointerEventsNone,
				Gradients: []core.Gradient{
					core.LinearGradient(180, core.RGBA(0x000000, 0), core.RGBA(0x000000, 0.72)),
				},
			},
			Children: []core.Node{
				core.View{
					Style: core.Style{
						Height:     core.Pt(3.5),
						Width:      core.Pct(100),
						Background: core.RGBA(0xffffff, 0.24),
					},
					Children: []core.Node{
						core.View{
							Style: core.Style{
								Height: core.Pct(100),
								Width:  core.Pct(pct),
							},
							StyleFunc: ui.Filled(colors.AccentFill),
						},
					},
				},
			},
		})
	}

	posterFrame := core.View{
		OnPress: func(core.PressEvent) {
			if opts.OnPress != nil {
				opts.OnPress()
			}
		},
		Style: core.Style{
			Group:        "poster",
			Width:        core.Pct(100),
			CornerRadius: 12,
			Overflow:     core.Hidden,
			BorderWidth:  1,
			Cursor:       core.CursorPointer,
			Transition: core.Transition{
				Duration:   180 * time.Millisecond,
				Easing:     core.EaseOut,
				Properties: core.TransitionTransform | core.TransitionBorder | core.TransitionShadow,
			},
		},
		StyleFunc: func() core.Style {
			return core.Style{
				Background:  colors.Inset(),
				BorderColor: colors.Line(),
				Shadows:     []core.BoxShadow{ui.Shadow(0, 4, 14, 0, core.RGBA(0x000000, 0.16))},
				Hover: &core.Style{
					BorderColor: ui.Alpha(colors.Accent(), 0.75),
					Transforms:  []core.TransformStep{{Kind: core.TransformTranslateY, Value: -3}},
					Shadows:     []core.BoxShadow{ui.Shadow(0, 10, 22, 0, core.RGBA(0x000000, 0.28))},
				},
			}
		},
		Children: posterOverlays,
	}

	titleNode := core.View{
		OnPress: func(core.PressEvent) {
			if opts.OnPress != nil {
				opts.OnPress()
			}
		},
		Style: core.Style{
			FlexGrow:   1,
			FlexShrink: 1,
			MinWidth:   core.Pt(0),
			Gap:        2,
			Cursor:     core.CursorPointer,
		},
		Children: func() []core.Node {
			nodes := []core.Node{
				core.Text{
					Content:  opts.Title,
					MaxLines: 2,
					Style: core.Style{
						FontSize:           13.5,
						FontWeight:         ui.Semibold,
						LineHeightMultiple: 1.28,
					},
					StyleFunc: ui.Colored(colors.Text),
				},
			}
			if subtitle != "" {
				nodes = append(nodes, core.Text{
					Content:   subtitle,
					MaxLines:  1,
					Style:     core.Style{FontSize: 11.5},
					StyleFunc: ui.Colored(colors.Muted),
				})
			}
			return nodes
		}(),
	}

	footerChildren := []core.Node{titleNode}
	if opts.ActionIcon != nil && opts.OnAction != nil {
		footerChildren = append(footerChildren, ui.IconButton{
			Icon:    opts.ActionIcon,
			Label:   opts.ActionLabel,
			Size:    24,
			OnPress: opts.OnAction,
		})
	}

	return core.View{
		Style: core.Style{
			Gap:   8,
			Width: core.Pct(100),
		},
		Children: []core.Node{
			posterFrame,
			core.View{
				Style: core.Style{
					Direction:  core.Row,
					AlignItems: core.AlignStart,
					Gap:        6,
					Padding:    core.Symmetric(0, 2),
				},
				Children: footerChildren,
			},
		},
	}
}

package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/nsgo-dev/nsgo"
	"github.com/nsgo-dev/nsgo/core"
	"github.com/nsgo-dev/nsgo/desktop"

	"zy-nsgo/internal/cms"
	"zy-nsgo/internal/proxy"
	"zy-nsgo/internal/store"
)

func main() {
	px := proxy.NewServer(true)
	if err := px.Start(); err != nil {
		log.Fatalf("start local proxy: %v", err)
	}
	defer px.Close()

	client := cms.NewClient()
	var openedStore *store.Store

	nsgo.Run(
		func(appScope *core.Scope) {
			desktop.SetShutdownMode(appScope, desktop.ShutdownOnMainWindowClose)
			nsgo.SetMainView(appScope, func(cx *core.Scope) core.Node {
				cx.SetTheme(cinemaTheme())
				if openedStore == nil {
					st, err := openAppStore(cx)
					if err != nil {
						log.Fatalf("open store: %v", err)
					}
					openedStore = st
				}
				app := newAppState(appScope, cx, client, openedStore, px)
				return RootView(cx, app)
			})
		},
		nsgo.WindowSize(1280, 820),
		nsgo.WindowConstraints(core.WindowConstraints{MinWidth: 920, MinHeight: 620}),
	)

	if openedStore != nil {
		_ = openedStore.Close()
	}
}

func openAppStore(cx *core.Scope) (*store.Store, error) {
	dbPath := resolveDBPath(cx)
	st, err := store.Open(dbPath)
	if err != nil && dbPath != ":memory:" {
		return store.Open(":memory:")
	}
	return st, err
}

func resolveDBPath(cx *core.Scope) string {
	if cx != nil {
		if dataDir := cx.DataDir(); dataDir != "" {
			if err := os.MkdirAll(dataDir, 0o755); err == nil {
				return filepath.Join(dataDir, "data.db")
			}
		}
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		return "zy-nsgo.db"
	}
	appDir := filepath.Join(dir, "zy-nsgo")
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		return "zy-nsgo.db"
	}
	return filepath.Join(appDir, "data.db")
}

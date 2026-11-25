package store

import (
	"testing"

	"zy-nsgo/internal/cms"
)

func TestStoreCRUD(t *testing.T) {
	st, err := Open(":memory:")
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	defer st.Close()

	if err := st.UpsertSite(cms.Site{Key: "lz", Name: "量子", API: "https://lz.example.com/api.php/provide/vod/", Active: true}, 1); err != nil {
		t.Fatalf("UpsertSite failed: %v", err)
	}
	imported, err := st.ImportSitesJSON([]byte(`{"sites":{"data":[{"key":"ff","name":"非凡","api":"https://ff.example.com/api.php/provide/vod/","isActive":false}]}}`))
	if err != nil || imported != 1 {
		t.Fatalf("ImportSitesJSON failed: imported=%d err=%v", imported, err)
	}

	sites, err := st.ListSites()
	if err != nil || len(sites) != 2 {
		t.Fatalf("ListSites failed: len=%d err=%v", len(sites), err)
	}
	if sites[0].Key != "ff" || sites[0].Active {
		t.Fatalf("unexpected first site: %+v", sites[0])
	}
	if err := st.DeleteSite("ff"); err != nil {
		t.Fatalf("DeleteSite failed: %v", err)
	}

	if err := st.SaveProgress(HistoryRecord{
		SiteKey: "lz", VodID: "101", VodName: "Test", EpisodeName: "第1集",
		EpisodeURL: "https://cdn.example.com/1.m3u8", PositionMS: 12000, DurationMS: 90000,
	}); err != nil {
		t.Fatalf("SaveProgress failed: %v", err)
	}
	rec, err := st.GetProgress("lz", "101")
	if err != nil || rec == nil || rec.PositionMS != 12000 {
		t.Fatalf("GetProgress unexpected: rec=%+v err=%v", rec, err)
	}
	hist, err := st.ListHistory(10)
	if err != nil || len(hist) != 1 {
		t.Fatalf("ListHistory failed: len=%d err=%v", len(hist), err)
	}

	if err := st.AddFavorite(FavoriteRecord{SiteKey: "lz", VodID: "101", VodName: "Test", Remarks: "HD"}); err != nil {
		t.Fatalf("AddFavorite failed: %v", err)
	}
	ok, err := st.IsFavorite("lz", "101")
	if err != nil || !ok {
		t.Fatalf("IsFavorite expected true, got ok=%v err=%v", ok, err)
	}
	if err := st.RemoveFavorite("lz", "101"); err != nil {
		t.Fatalf("RemoveFavorite failed: %v", err)
	}

	if err := st.SetSetting("skip_intro_sec", "90"); err != nil {
		t.Fatalf("SetSetting failed: %v", err)
	}
	val, err := st.GetSetting("skip_intro_sec")
	if err != nil || val != "90" {
		t.Fatalf("GetSetting expected 90, got %q err=%v", val, err)
	}
}

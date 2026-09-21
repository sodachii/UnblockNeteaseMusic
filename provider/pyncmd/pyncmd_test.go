package pyncmd_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/cnsilvan/UnblockNeteaseMusic/common"
	"github.com/cnsilvan/UnblockNeteaseMusic/provider"
	"github.com/cnsilvan/UnblockNeteaseMusic/provider/pyncmd"
)

func TestPyncmdGetSongUrl(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if query.Get("types") != "url" || query.Get("source") != "netease" {
			http.Error(w, "invalid query", http.StatusBadRequest)
			return
		}
		id := query.Get("id")
		if id == "12345" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"url":  "https://example.com/song12345.mp3",
				"br":   320,
				"size": 10000000,
				"from": "music.gdstudio.xyz",
			})
			return
		}
		if id == "flac_song" {
			_ = json.NewEncoder(w).Encode(map[string]any{
				"url":  "https://example.com/song.flac",
				"br":   800,
				"size": 30000000,
				"from": "music.gdstudio.xyz",
			})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"url":  "",
			"br":   0,
			"size": 0,
			"from": "music.gdstudio.xyz",
		})
	}))
	defer server.Close()

	p := &pyncmd.Pyncmd{}

	// Test GetSongUrl with basic song
	song := &common.Song{
		Id: string(common.PyncmdTag) + "12345",
		PlatformUniqueKey: map[string]any{
			"id": "12345",
		},
	}
	res := p.GetSongUrl(common.SearchMusic{Id: "12345", Quality: common.ExHigh}, song)
	if res == nil {
		t.Fatal("expected non-nil song")
	}
}

func TestPyncmdProviderRegistration(t *testing.T) {
	p := provider.NewProvider("pyncmd")
	if _, ok := p.(*pyncmd.Pyncmd); !ok {
		t.Fatalf("expected *pyncmd.Pyncmd, got %T", p)
	}

	common.Source = []string{"pyncmd"}
	provider.Init()
	prov := provider.GetProvider("pyncmd")
	if _, ok := prov.(*pyncmd.Pyncmd); !ok {
		t.Fatalf("expected *pyncmd.Pyncmd from GetProvider, got %T", prov)
	}
}

func TestPyncmdParseSongWithId(t *testing.T) {
	p := &pyncmd.Pyncmd{}
	song := p.ParseSong(common.SearchSong{
		Id:          "33894312",
		Name:        "海阔天空",
		ArtistsName: "Beyond",
		Quality:     common.ExHigh,
	})
	if song == nil {
		t.Fatal("expected non-nil song result")
	}
	if song.Url != "" {
		if !strings.HasPrefix(song.Url, "http") {
			t.Errorf("expected valid http url, got %s", song.Url)
		}
		if song.Br <= 0 {
			t.Errorf("expected positive bitrate, got %d", song.Br)
		}
	}
}

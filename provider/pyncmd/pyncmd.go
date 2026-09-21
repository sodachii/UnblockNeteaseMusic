package pyncmd

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"

	"github.com/buger/jsonparser"
	"github.com/cnsilvan/UnblockNeteaseMusic/common"
	"github.com/cnsilvan/UnblockNeteaseMusic/provider/base"
	"github.com/cnsilvan/UnblockNeteaseMusic/utils"
)

type Pyncmd struct{}

const (
	APIGetSongURL = "https://music-api.gdstudio.xyz/api.php?types=url&source=netease&id=%s&br=%s"
	SearchSongURL = "https://music-api.gdstudio.xyz/api.php?types=search&source=netease&count=10&pages=1&name=%s"
)

func (m *Pyncmd) SearchSong(song common.SearchSong) (songs []*common.Song) {
	song = base.PreSearchSong(song)
	if song.Id != "" && song.Id != "0" && !strings.HasPrefix(song.Id, string(common.StartTag)) {
		songResult := &common.Song{
			Id:         string(common.PyncmdTag) + song.Id,
			Name:       song.Name,
			Artist:     song.ArtistsName,
			Source:     "pyncmd",
			MatchScore: 100,
			PlatformUniqueKey: map[string]any{
				"id":        song.Id,
				"musicId":   song.Id,
				"UnKeyWord": song.Keyword,
			},
		}
		songs = append(songs, songResult)
		return base.AfterSearchSong(song, songs)
	}

	searchKeyword := song.Keyword
	if searchKeyword == "" {
		searchKeyword = song.Name
	}
	searchUrl := fmt.Sprintf(SearchSongURL, url.QueryEscape(searchKeyword))
	result, err := base.FetchV2(searchUrl, nil, nil, true)
	if err != nil {
		slog.Error("pyncmd search fetch error", slog.Any("error", err))
		return songs
	}

	var list [][]byte
	_, _ = jsonparser.ArrayEach(result, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
		if err != nil {
			return
		}
		list = append(list, value)
	})

	listLength := len(list)
	maxIndex := listLength/2 + 1
	if maxIndex > 5 {
		maxIndex = 5
	}

	for index, item := range list {
		if index >= maxIndex {
			break
		}
		songId := utils.StringFromJSON(item, "id")
		if songId == "" {
			continue
		}

		var artists []string
		_, _ = jsonparser.ArrayEach(item, func(value []byte, dataType jsonparser.ValueType, offset int, err error) {
			if err == nil && len(value) > 0 {
				artists = append(artists, string(value))
			}
		}, "artist")

		artistStr := strings.Join(artists, " & ")
		if artistStr == "" {
			artistStr = utils.StringFromJSON(item, "artist")
		}

		songResult := &common.Song{
			Id:        string(common.PyncmdTag) + songId,
			Name:      utils.StringFromJSON(item, "name"),
			Artist:    artistStr,
			AlbumName: utils.StringFromJSON(item, "album"),
			Source:    "pyncmd",
			PlatformUniqueKey: map[string]any{
				"id":        songId,
				"musicId":   songId,
				"UnKeyWord": song.Keyword,
			},
		}

		var ok bool
		songResult.MatchScore, ok = base.CalScore(song, songResult.Name, songResult.Artist, index, maxIndex)
		if !ok {
			continue
		}
		songs = append(songs, songResult)
	}

	return base.AfterSearchSong(song, songs)
}

func (m *Pyncmd) GetSongUrl(searchSong common.SearchMusic, song *common.Song) *common.Song {
	var songId string
	if id, ok := song.PlatformUniqueKey["id"].(string); ok && id != "" {
		songId = id
	} else if id, ok := song.PlatformUniqueKey["musicId"].(string); ok && id != "" {
		songId = id
	} else if strings.HasPrefix(song.Id, string(common.PyncmdTag)) {
		songId = strings.TrimPrefix(song.Id, string(common.PyncmdTag))
	} else if strings.HasPrefix(searchSong.Id, string(common.PyncmdTag)) {
		songId = strings.TrimPrefix(searchSong.Id, string(common.PyncmdTag))
	} else if song.Id != "" && song.Id != "0" && !strings.HasPrefix(song.Id, string(common.StartTag)) {
		songId = song.Id
	} else if searchSong.Id != "" && searchSong.Id != "0" && !strings.HasPrefix(searchSong.Id, string(common.StartTag)) {
		songId = searchSong.Id
	}

	if songId == "" {
		return song
	}

	br := "320"
	switch searchSong.Quality {
	case common.Standard:
		br = "128"
	case common.Higher:
		br = "192"
	case common.ExHigh:
		br = "320"
	case common.Lossless, common.Hires, common.JYEffect, common.Sky, common.JYMaster:
		br = "999"
	default:
		br = "999"
	}

	urlStr := fmt.Sprintf(APIGetSongURL, url.QueryEscape(songId), br)
	result, err := base.FetchV2(urlStr, nil, nil, true)
	if err != nil {
		slog.Error("pyncmd get song url fetch error", slog.Any("error", err))
		return song
	}

	var res struct {
		Url  string      `json:"url"`
		Br   json.Number `json:"br"`
		Size json.Number `json:"size"`
		From string      `json:"from"`
	}
	if err := json.Unmarshal(result, &res); err != nil {
		slog.Error("pyncmd parse json error", slog.Any("error", err))
		return song
	}

	if res.Url == "" {
		return song
	}

	brNum, _ := res.Br.Int64()
	if brNum <= 0 {
		return song
	}

	song.Url = res.Url
	if sizeNum, err := res.Size.Int64(); err == nil && sizeNum > 0 {
		song.Size = sizeNum
	}
	if brNum > 0 && brNum < 10000 {
		song.Br = int(brNum * 1000)
	} else if brNum >= 10000 {
		song.Br = int(brNum)
	}

	return song
}

func (m *Pyncmd) ParseSong(searchSong common.SearchSong) *common.Song {
	song := &common.Song{}
	if searchSong.Id != "" && searchSong.Id != "0" && !strings.HasPrefix(searchSong.Id, string(common.StartTag)) {
		song.Id = string(common.PyncmdTag) + searchSong.Id
		song.Name = searchSong.Name
		song.Artist = searchSong.ArtistsName
		song.PlatformUniqueKey = map[string]any{
			"id":        searchSong.Id,
			"musicId":   searchSong.Id,
			"UnKeyWord": searchSong.Keyword,
		}
		song = m.GetSongUrl(common.SearchMusic{Id: searchSong.Id, Quality: searchSong.Quality}, song)
		if song.Url != "" {
			return song
		}
	}

	songs := m.SearchSong(searchSong)
	if len(songs) > 0 {
		song = m.GetSongUrl(common.SearchMusic{Quality: searchSong.Quality}, songs[0])
	}
	return song
}

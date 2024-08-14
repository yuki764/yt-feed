package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strings"

	"google.golang.org/api/youtube/v3"
)

func main() {
	type video struct {
		Id          string `json:"id"`
		Title       string `json:"title"`
		PublishedAt string `json:"pubAt"`
		Thumbnail   string `json:"thumb"`
	}

	ctx := context.Background()

	service, err := youtube.NewService(ctx)
	if err != nil {
		slog.Error("failed to create new YouTube client", "error", err)
		panic(err)
	}

	channelHandles := strings.Split(os.Getenv("CHANNEL_HANDLES"), ",")

	for _, ch := range channelHandles {
		uploadsPlaylistId := ""
		cResp, err := service.Channels.List([]string{"id", "contentDetails"}).
			ForHandle(ch).
			Do()
		if err != nil {
			panic(err)
		}
		for _, c := range cResp.Items {
			uploadsPlaylistId = c.ContentDetails.RelatedPlaylists.Uploads
			slog.Info("channel @"+ch, "channelId", c.Id, "uploadsPlaylistId", uploadsPlaylistId)
		}

		// calcurate file name
		fn := os.Getenv("FILE_PREFIX") + ch + ".json"

		// load videos info from the file
		existVideos := []video{}

		if _, err := os.Stat(fn); err == nil {
			ef, err := os.Open(fn)
			if err != nil {
				panic(err)
			}
			defer ef.Close()
			if err := json.NewDecoder(ef).Decode(&existVideos); err != nil {
				panic(err)
			}
		}

		lResp, err := service.PlaylistItems.List([]string{"id", "snippet"}).
			PlaylistId(uploadsPlaylistId).
			MaxResults(30).
			Do()
		if err != nil {
			panic(err)
		}

		newVideos := []video{}

		for _, v := range lResp.Items {
			newVideos = append(newVideos, video{
				Id:          v.Snippet.ResourceId.VideoId,
				Title:       v.Snippet.Title,
				PublishedAt: v.Snippet.PublishedAt,
				Thumbnail:   v.Snippet.Thumbnails.High.Url,
			})
		}

		if len(existVideos) > 0 && newVideos[0].Id == existVideos[0].Id {
			slog.Info("videos info is latest", "latestVideoId", newVideos[0].Id, "latestVideoTitle", newVideos[0].Title)
		} else {
			// concatinate new videos and existing videos then save into the file
			newVideoIds := make(map[string]bool)
			for _, v := range newVideos {
				newVideoIds[v.Id] = true
			}
			for i, v := range existVideos {
				if _, ok := newVideoIds[v.Id]; !ok {
					newVideos = append(newVideos, existVideos[i:]...)
					slog.Info("concatinated videos info", "borderVideoId", existVideos[i].Id, "borderVideoTitle", existVideos[i].Title)
					break
				}
			}

			nf, err := os.OpenFile(fn, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
			if err != nil {
				panic(err)
			}
			defer nf.Close()
			if err := json.NewEncoder(nf).Encode(newVideos); err != nil {
				panic(err)
			}
		}

		/* for use search API (consume too many quota!)
		sResp, err := service.Search.List([]string{"id", "snippet"}).
			Type("video").
			ChannelId(cid).
			MaxResults(25).
			Order("date").
			Do()
		if err != nil {
			panic(err)
		}

		// Iterate through each item and add it to the correct list.
		type video struct {
			Id          string `json:"id"`
			Title       string `json:"title"`
			PublishedAt string `json:"pubAt"`
			Thumbnail   string `json:"thumb"`
		}
		videos := []video{}

		for _, v := range sResp.Items {
			videos = append(videos, video{
				Id:          v.Id.VideoId,
				Title:       v.Snippet.Title,
				PublishedAt: v.Snippet.PublishedAt,
				Thumbnail:   v.Snippet.Thumbnails.High.Url,
			})
			slog.Info("info", "video", videos[len(videos)-1])
		}

		f, err := os.OpenFile(ch+".json", os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0644)
		if err != nil {
			panic(err)
		}
		if err := json.NewEncoder(f).Encode(videos); err != nil {
			panic(err)
		}
		*/
	}
}

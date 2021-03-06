package composition

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	netUrl "net/url"
	"os"

	"appengine"
	"appengine/urlfetch"
)

const MercuryURL = "https://mercury.postlight.com/parser?url="

var MercuryToken = os.Getenv("MERCURY_TOKEN")
var YoutubeToken = os.Getenv("YOUTUBE_TOKEN")

// API Response for /article
type ArticleResponse struct {
	Excerpt       string `json:"excerpt"`
	Title         string `json:"title"`
	URL           string `json:"url"`
	DatePublished string `json:"date_published"`
}

// API Response for /youtube
type YoutubeResponse struct {
	Title         string `json:"title"`
	URL           string `json:"url"`
	AuthorName    string `json:"author_name"`
	Thumbnail     string `json:"thumbnail"`
	DatePublished string `json:"date_published"`
	Description   string `json:"description"`
}

// Autogenerated struct based on the response of the youtube video API
type GeneratedYoutubeResponse struct {
	Kind     string `json:"kind"`
	Etag     string `json:"etag"`
	PageInfo struct {
		TotalResults   int `json:"totalResults"`
		ResultsPerPage int `json:"resultsPerPage"`
	} `json:"pageInfo"`
	Items []struct {
		Kind    string `json:"kind"`
		Etag    string `json:"etag"`
		ID      string `json:"id"`
		Snippet struct {
			PublishedAt string `json:"publishedAt"`
			ChannelID   string `json:"channelId"`
			Title       string `json:"title"`
			Description string `json:"description"`
			Thumbnails  struct {
				Default struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"default"`
				Medium struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"medium"`
				High struct {
					URL    string `json:"url"`
					Width  int    `json:"width"`
					Height int    `json:"height"`
				} `json:"high"`
			} `json:"thumbnails"`
			ChannelTitle         string   `json:"channelTitle"`
			Tags                 []string `json:"tags"`
			CategoryID           string   `json:"categoryId"`
			LiveBroadcastContent string   `json:"liveBroadcastContent"`
			DefaultLanguage      string   `json:"defaultLanguage"`
			Localized            struct {
				Title       string `json:"title"`
				Description string `json:"description"`
			} `json:"localized"`
		} `json:"snippet"`
	} `json:"items"`
}

func init() {
	http.HandleFunc("/article", HandleArticle)
	http.HandleFunc("/youtube", HandleYoutube)
}

func HandleArticle(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "Missing URL Parameter", http.StatusBadRequest)
		return
	}

	resp, err := ScrapeArticle(url, urlfetch.Client(appengine.NewContext(r)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

func HandleYoutube(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")
	if url == "" {
		http.Error(w, "Missing URL Parameter", http.StatusBadRequest)
		return
	}

	resp, err := ScrapeYouTube(url, urlfetch.Client(appengine.NewContext(r)))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(resp)
}

// Scrapes article using Mercury API
func ScrapeArticle(url string, client *http.Client) (*ArticleResponse, error) {
	req, err := http.NewRequest(http.MethodGet, MercuryURL+url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("x-api-key", MercuryToken)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	artResp := ArticleResponse{}

	err = json.Unmarshal(bodyBytes, &artResp)
	return &artResp, err
}

// Scrapes YouTube video based on YouTube v3 API
func ScrapeYouTube(rawUrl string, client *http.Client) (*YoutubeResponse, error) {
	parsedUrl, err := netUrl.Parse(rawUrl)
	if err != nil {
		return nil, err
	}
	videoID := parsedUrl.Query().Get("v")
	if videoID == "" {
		return nil, errors.New("Missing video ID")
	}

	resp, err := client.Get("https://www.googleapis.com/youtube/v3/videos?part=snippet&id=" + netUrl.QueryEscape(videoID) + "&key=" + YoutubeToken)
	if err != nil {
		panic(err)
	}

	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	genYTResp := GeneratedYoutubeResponse{}
	err = json.Unmarshal(bodyBytes, &genYTResp)
	if err != nil {
		return nil, err
	}

	if len(genYTResp.Items) < 1 {
		return nil, errors.New("Could Not Find Video")
	}

	ytResp := YoutubeResponse{
		Title:         genYTResp.Items[0].Snippet.Title,
		URL:           rawUrl,
		AuthorName:    genYTResp.Items[0].Snippet.ChannelTitle,
		Thumbnail:     genYTResp.Items[0].Snippet.Thumbnails.Default.URL,
		DatePublished: genYTResp.Items[0].Snippet.PublishedAt,
		Description:   genYTResp.Items[0].Snippet.Description,
	}

	return &ytResp, nil
}

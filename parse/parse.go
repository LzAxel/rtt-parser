package parse

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"reddit-parse/main/logger"
	"reddit-parse/main/settings"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/tidwall/gjson"
	reddit "github.com/turnage/graw/reddit"
)

func GetImagesFromGallery(url string) ([]string, error) {
	var myClient = &http.Client{Timeout: 10 * time.Second}
	var rawJson string
	var galleryImages []string

	url = strings.Replace(url, "gallery", "comments", 1) + ".json"

	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth("", "")
	userAgent := "graw:" + settings.Config.Reddit.Id + ":v1(by u/lzaxel)"
	req.Header.Set("User-agent", userAgent)

	resp, err := myClient.Do(req)
	if err != nil {
		return galleryImages, err
	}
	rawJsonBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return galleryImages, err
	}

	rawJson = string(rawJsonBytes)
	metadata := gjson.Get(rawJson, "0.data.children.0.data.media_metadata|@values.#.p.0.u")
	for _, imgUrl := range metadata.Array() {
		validImgUrl := strings.Split(imgUrl.String(), "?")[0]
		validImgUrl = strings.Replace(validImgUrl, "preview", "i", 1)
		galleryImages = append(galleryImages, validImgUrl)
	}

	defer resp.Body.Close()

	return galleryImages, err
}

func IsGallery(url string) bool {
	splitedUrl := strings.Split(url, "/")
	if splitedUrl[len(splitedUrl)-2] == "gallery" {
		return true
	} else {
		return false
	}

}
func GetImageExtansion(url string) (string, error) {
	splitedUrl := strings.Split(url, ".")
	extansion := splitedUrl[len(splitedUrl)-1]
	if strings.Contains("gif png jpg webm webp", extansion) {
		return extansion, nil
	} else if IsGallery(url) {
		return "gallery", nil
	} else {
		return extansion, errors.New("post doesn't contains image")
	}
}
func GetPosts(client reddit.Bot) ([]*reddit.Post, error) {
	logger.InfoLogger.Println("Getting posts")
	params := map[string]string{
		"limit": settings.Config.Reddit.PostLimit,
		"t":     settings.Config.Reddit.Period,
	}
	path := settings.Config.Reddit.Subreddit + settings.Config.Reddit.Sort
	logger.DebugLogger.Printf("Url path: %s", path)
	harvest, err := client.ListingWithParams(path, params)
	if err != nil {
		fmt.Println(err)
		return harvest.Posts, err
	}

	return harvest.Posts, err
}
func ValidatePosts(posts []*reddit.Post) []*reddit.Post {
	logger.InfoLogger.Printf("Validating posts url | Limit: %s", settings.Config.Reddit.PostLimit)
	validatedPosts := []*reddit.Post{}

	for _, post := range posts {
		_, err := GetImageExtansion(post.URL)
		if err != nil {
			logger.DebugLogger.Printf("[-] %s", post.URL)
		} else {
			logger.DebugLogger.Printf("[+] %s", post.URL)
			validatedPosts = append(validatedPosts, post)
		}

	}

	return validatedPosts
}
func SaveToJson(posts []*reddit.Post) error {
	logger.InfoLogger.Println("Saving new urls to json")
	var savedPosts []string
	postUrls := []string{}

	for _, post := range posts {
		postUrls = append(postUrls, post.URL)
	}

	savedPostsByte, err := ioutil.ReadFile("savedPosts.json")
	if err != nil {
		return err
	}

	err = json.Unmarshal(savedPostsByte, &savedPosts)
	if err != nil {
		return err
	}

	savedPosts = append(savedPosts, postUrls...)

	formattedUrls, err := json.MarshalIndent(savedPosts, "", "    ")
	if err != nil {
		return err
	}
	err = ioutil.WriteFile("savedPosts.json", formattedUrls, 0644)

	return err
}
func CheckIfSaved(posts []*reddit.Post) ([]*reddit.Post, error) {
	logger.InfoLogger.Printf("Cheking for new posts")

	var savedPosts []string
	var NewPosts []*reddit.Post

	data, err := ioutil.ReadFile("savedPosts.json")
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(data, &savedPosts)
	if err != nil {
		return nil, err
	}

	for _, post := range posts {
		isSaved := false
		for _, savedPost := range savedPosts {
			if post.URL == savedPost {
				isSaved = true
				break
			}
		}
		if !isSaved {
			NewPosts = append(NewPosts, post)
			logger.DebugLogger.Printf("%s - Is a new post", post.URL)
		}
	}
	return NewPosts, err
}
func SendImages(images []*reddit.Post) error {
	logger.InfoLogger.Println("Sending images")
	bot, err := tgbotapi.NewBotAPI(settings.Config.Telegram.Token)
	chat := settings.Config.Telegram.ChatId
	if err != nil {
		logger.ErrorLogger.Panicln(err)
	}

	for i, image := range images {

		if i+1%5 == 0 {
			logger.DebugLogger.Println("Sleep for 5 sec...zzz")
			time.Sleep(5 * time.Second)
		}
		logger.DebugLogger.Println("Sending", image.URL)

		extansion, err := GetImageExtansion(image.URL)
		if err != nil {
			break
		}
		switch extansion {
		case "gallery":
			logger.DebugLogger.Println("Gallery found")
			photoList, err := GetImagesFromGallery(image.URL)
			if err != nil {
				logger.ErrorLogger.Println(err)
				continue
			}
			for index, img := range photoList {
				logger.DebugLogger.Println(img)
				photo := tgbotapi.NewPhoto(chat, tgbotapi.FileURL(img))
				msg, _ := bot.Send(photo)
				if index == 0 {
					edit := tgbotapi.NewEditMessageCaption(chat, msg.MessageID, image.Title)
					bot.Send(edit)
				}

			}

		case "gif":
			photo := tgbotapi.NewDocument(chat, tgbotapi.FileURL(image.URL))
			msg, _ := bot.Send(photo)
			tgbotapi.NewEditMessageCaption(chat, msg.MessageID, image.Title)

		default:
			photo := tgbotapi.NewPhoto(chat, tgbotapi.FileURL(image.URL))
			msg, _ := bot.Send(photo)
			edit := tgbotapi.NewEditMessageCaption(chat, msg.MessageID, image.Title)
			bot.Send(edit)
		}

	}
	return err
}
func CheckFirstStart() {
	if _, err := os.Stat("savedPosts.json"); err != nil {
		ioutil.WriteFile("savedPosts.json", []byte("[]"), 0644)
	}
}
func StartParsing(stateChan, exitChan chan int, errChan chan error) {
	logger.InfoLogger.Println("Start parsing")

	cfg := reddit.BotConfig{
		Agent: "graw:6qr2NUgVDlxzv0vyEC8v4w:v1(by u/lzaxel)",

		App: reddit.App{
			ID:       settings.Config.Reddit.Id,
			Secret:   settings.Config.Reddit.Secret,
			Username: settings.Config.Reddit.Username,
			Password: settings.Config.Reddit.Password,
		},
	}
	stateChan <- 0
	bot, _ := reddit.NewBot(cfg)
	stateChan <- 1
	posts, err := GetPosts(bot)
	if err != nil {
		stateChan <- 99
		errChan <- err
		logger.ErrorLogger.Println(err)
	}
	stateChan <- 2
	posts = ValidatePosts(posts)
	stateChan <- 3
	posts, err = CheckIfSaved(posts)
	if err != nil {
		stateChan <- 99
		errChan <- err
		logger.ErrorLogger.Panicln(err)
	}
	if len(posts) != 0 {
		stateChan <- 4
		SaveToJson(posts)

		stateChan <- 5
		err = SendImages(posts)
		if err != nil {
			stateChan <- 99
			errChan <- err
			logger.ErrorLogger.Println(err)
		}
	}
	stateChan <- 6
	exitChan <- 0
	logger.DebugLogger.Println("Parsing finished")
}

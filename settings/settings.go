package settings

import (
	"encoding/json"
	"reddit-parse/main/logger"

	"fyne.io/fyne/v2"
)

var Config *AppSettings

type AppSettings struct {
	Telegram  TelegramSettings `json:"telegram"`
	Reddit    RedditSetting    `json:"reddit"`
	SleepTime int              `json:"sleepTime,string"`
}
type TelegramSettings struct {
	Token  string `json:"token"`
	ChatId int64  `json:"chatId,string"`
}
type RedditSetting struct {
	Id        string `json:"id"`
	Secret    string `json:"secret"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Subreddit string `json:"subreddit"`
	PostLimit string `json:"limit"`
	Period    string `json:"period"`
	Sort      string `json:"sort"`
}

func init() {
	Config = &AppSettings{}
	Config.SleepTime = 120
	Config.Reddit.PostLimit = "70"
}

func ImportSettings(data []byte) error {
	logger.DebugLogger.Println("Importing settings")

	err := json.Unmarshal(data, Config)
	if err != nil {
		return err
	}

	return err
}

func ExportSettings(uc fyne.URIWriteCloser) error {
	logger.DebugLogger.Println("Exporting settings")

	data, err := json.MarshalIndent(Config, "", "    ")
	if err != nil {
		return err
	}
	_, err = uc.Write(data)

	return err
}

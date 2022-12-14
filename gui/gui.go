package gui

import (
	"fmt"
	"io/ioutil"
	"os"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"reddit-parse/main/logger"
	"reddit-parse/main/parse"
	"reddit-parse/main/settings"
)

func init() {
	os.Setenv("FYNE_SCALE", "1")
}

var (
	tgTokenEntry, tgChatEntry, redSecretEntry      *widget.Entry
	redIdEntry, redUsernameEntry, redPasswordEntry *widget.Entry
	subredditEntry, limitEntry, sleepEntry         *widget.Entry
	periodSelect, sortSelect                       *widget.Select
)

func CheckSettings() *settings.AppSettings {
	data, err := ioutil.ReadFile("settings.json")
	if err != nil {
		return settings.Config
	}
	_ = settings.ImportSettings(data)

	return settings.Config
}

func UpdateSettings() error {
	var err error
	settings.Config.Telegram.ChatId, err = strconv.ParseInt(tgChatEntry.Text, 0, 64)
	settings.Config.Telegram.Token = tgTokenEntry.Text
	if err != nil {
		logger.ErrorLogger.Println(err)
		return err
	}
	settings.Config.Reddit.Secret = redSecretEntry.Text
	settings.Config.Reddit.Id = redIdEntry.Text
	settings.Config.Reddit.Username = redUsernameEntry.Text
	settings.Config.Reddit.Password = redPasswordEntry.Text
	settings.Config.Reddit.PostLimit = limitEntry.Text
	settings.Config.Reddit.Sort = sortSelect.Selected
	settings.Config.Reddit.Period = periodSelect.Selected
	settings.Config.Reddit.Subreddit = subredditEntry.Text
	settings.Config.SleepTime, err = strconv.Atoi(sleepEntry.Text)
	if err != nil {
		logger.ErrorLogger.Println(err)
		return err
	}

	return err
}

func UpdateFields() {
	tgTokenEntry.SetText(settings.Config.Telegram.Token)
	tgChatEntry.SetText(fmt.Sprint(settings.Config.Telegram.ChatId))
	redSecretEntry.SetText(settings.Config.Reddit.Secret)
	redIdEntry.SetText(settings.Config.Reddit.Id)
	redUsernameEntry.SetText(settings.Config.Reddit.Username)
	redPasswordEntry.SetText(settings.Config.Reddit.Password)
	subredditEntry.SetText(settings.Config.Reddit.Subreddit)
	limitEntry.SetText(settings.Config.Reddit.PostLimit)
	sleepEntry.SetText(fmt.Sprint(settings.Config.SleepTime))
	sortSelect.SetSelected(settings.Config.Reddit.Sort)
	periodSelect.SetSelected(settings.Config.Reddit.Period)
}

func StartGui() {
	app := app.New()
	window := app.NewWindow("???????????????? ??????")
	window.Resize(fyne.Size{Width: 500, Height: 300})
	// window.SetFixedSize(true)

	r, _ := fyne.LoadResourceFromPath("icon.png")
	window.SetIcon(r)

	parse.CheckFirstStart()

	tgTokenEntry = widget.NewPasswordEntry()
	tgChatEntry = widget.NewEntry()
	redSecretEntry = widget.NewPasswordEntry()
	redIdEntry = widget.NewEntry()
	redUsernameEntry = widget.NewEntry()
	redPasswordEntry = widget.NewPasswordEntry()
	subredditEntry = widget.NewEntry()
	subredditEntry.SetPlaceHolder("/user/aboba32/m/mr_name/")
	limitEntry = widget.NewEntry()
	limitEntry.SetPlaceHolder("70")
	sleepEntry = widget.NewEntry()
	sleepEntry.SetPlaceHolder("120")
	sortSelect = widget.NewSelect([]string{"top", "hot", "new", "rising"}, func(s string) {})
	sortSelect.SetSelected("top")
	periodSelect = widget.NewSelect([]string{"hour", "day", "week", "month", "year"}, func(s string) {})
	periodSelect.SetSelected("day")
	stateLabel := widget.NewLabel("????????????????")

	CheckSettings()
	UpdateFields()

	exitChan := make(chan int)
	stateChan := make(chan int)
	errChan := make(chan error)

	stopBtn := widget.NewButton("????????????????????", nil)
	stopBtn.Disable()
	startBtn := widget.NewButton("??????????????????", nil)

	importBtn := widget.NewButton("????????????", func() {
		fileDialog := dialog.NewFileOpen(
			func(uc fyne.URIReadCloser, err error) {
				if uc == nil {
					return
				}
				data, err := ioutil.ReadAll(uc)
				if err != nil {
					logger.ErrorLogger.Println(err)
					return
				}

				err = settings.ImportSettings(data)

				if err != nil {
					logger.ErrorLogger.Println(err)
					return
				}
				UpdateFields()
			}, window)
		fileDialog.Show()
	})

	exportBtn := widget.NewButton("??????????????", func() {
		fileDialog := dialog.NewFileSave(
			func(uc fyne.URIWriteCloser, err error) {
				if uc == nil {
					return
				}
				err = settings.ExportSettings(uc)
				if err != nil {
					logger.ErrorLogger.Println(err)
					return
				}

			}, window)
		fileDialog.Show()
	})

	stopBtn.OnTapped = func() {
		exitChan <- 1
		stopBtn.Disable()
		go func() {
			<-exitChan
			stateChan <- 10
			startBtn.Enable()
		}()
	}
	startBtn.OnTapped = func() {
		startBtn.Disable()
		stopBtn.Enable()

		UpdateSettings()

		go func() {

			for i := 0; i >= 0; i++ {
				logger.DebugLogger.Println("Searching Loop `i=`", i)
				a := <-exitChan
				logger.DebugLogger.Println("Exit code =", a)
				if a == 1 {
					return
				}
				if i != 0 {
					logger.DebugLogger.Println("Sleep for", settings.Config.SleepTime)
					for b := 0; b <= settings.Config.SleepTime; b++ {
						time.Sleep(time.Second)
					}
				}
				go parse.StartParsing(stateChan, exitChan, errChan)
			}
		}()
		exitChan <- 0
		logger.DebugLogger.Println("End")
	}

	displayUpdates := func() {
		for state := range stateChan {
			switch state {
			case 0:
				stateLabel.SetText("??????????????????????????")
			case 1:
				stateLabel.SetText("?????????????????? ????????????")
			case 2:
				stateLabel.SetText("?????????????????? ????????????")
			case 3:
				stateLabel.SetText("?????????? ?????????? ????????????")
			case 4:
				stateLabel.SetText("???????????????????? ?????????? ????????????")
			case 5:
				stateLabel.SetText("????????????????")
			case 6:
				stateLabel.SetText("???????????????? ?????????? ????????????")
			case 10:
				stateLabel.SetText("????????????????")
			case 99:
				stopBtn.Disable()
				errDialog := dialog.NewError(<-errChan, window)
				errDialog.Show()
				exitChan <- 1
				startBtn.Enable()
			}
		}
	}

	go displayUpdates()

	window.SetContent(

		container.NewVBox(
			container.NewVBox(
				container.NewGridWithColumns(2,
					container.NewVBox(
						container.NewCenter(
							widget.NewLabel("?????????????????? ??????????????????????"),
						),
						container.NewGridWithColumns(2,
							container.NewVBox(widget.NewLabel("???????????????? ??????????"), tgTokenEntry),
							container.NewVBox(widget.NewLabel("???????????? ??????????"), redSecretEntry),
							container.NewVBox(widget.NewLabel("???????????? ID"), redIdEntry),
							container.NewVBox(widget.NewLabel("???????????? ??????????"), redUsernameEntry),
							container.NewVBox(widget.NewLabel("???????????? ????????????"), redPasswordEntry),
							container.NewVBox(widget.NewLabel("ID ????????????"), tgChatEntry),
						),
					),
					container.NewVBox(
						container.NewCenter(
							widget.NewLabel("?????????????????? ????????????????"),
						),
						container.NewGridWithColumns(2,
							container.NewVBox(widget.NewLabel("??????-???? ????????????"), limitEntry),
							container.NewVBox(widget.NewLabel("????????????????????"), sortSelect),
							container.NewVBox(widget.NewLabel("????????????"), periodSelect),
							container.NewVBox(widget.NewLabel("?????????? ????????????????"), sleepEntry),
							container.NewVBox(widget.NewLabel("??????????????????"), subredditEntry),
							layout.NewSpacer(),
						),
					),
				),
				container.NewGridWithColumns(2,
					importBtn,
					exportBtn,
				),
			),
			layout.NewSpacer(),
			container.NewHBox(
				widget.NewLabelWithStyle("??????????????????: ", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}),
				stateLabel,
			),
			layout.NewSpacer(),
			container.NewGridWithColumns(2,
				startBtn, stopBtn,
			),
		))
	window.ShowAndRun()
}

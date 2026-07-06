package main

import (
	"errors"
	"fmt"
	"image/color"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
	dialogx "fyne.io/x/fyne/dialog"
)

var (
	colorLightBlue = color.RGBA{R: 158, G: 209, B: 226, A: 255}
	fyneApp        fyne.App
	fyneWindow     fyne.Window
	userSettings   Settings
)

type Settings struct {
	timeFormat      string
	dateFormat      string
	catTransparency int
	location        string
	isFirstRun      bool
}

func main() {
	fyneApp = app.NewWithID("br.dev.lostdusty.flock")
	fyneWindow = fyneApp.NewWindow("flock")
	fyneWindow.Resize(fyne.Size{Width: 50, Height: 50})
	fyneApp.Driver().SetDisableScreenBlanking(true)

	//populate settings struct
	loadUserPreferences()

	//start logging
	f, err := os.CreateTemp("", "flock-*.log")
	if err != nil {
		log.Fatalln("Unable to create logging file")
	}
	defer os.Remove(f.Name())

	log.SetOutput(f)

	log.Println("Flock logging started ok.")

	//welcome banner
	welcomeText := widget.NewRichTextFromMarkdown("## Welcome!")

	buttonHelp := widget.NewButton("F1 Show/Hide this pop-up", nil)
	buttonHelp.Importance = widget.LowImportance

	buttonSettings := widget.NewButton("F2 Change Settings", nil)
	buttonSettings.Importance = widget.LowImportance
	showSettings := func() {
		selectorDate := widget.NewSelect([]string{"02/01/2006 Monday",
			"02/01/2006",
			"01/02/2006 Monday",
			"01/02/2006",
			"02-01-2006 Monday",
			"02-01-2006",
			"01-02-2006 Monday",
			"01-02-2006",
			"02/01/2006 Mon",
			"02/01/2006",
			"01/02/2006 Mon",
			"01/02/2006",
			"02-01-2006 Mon",
			"02-01-2006",
			"01-02-2006 Mon",
			"01-02-2006",
			"02 Jan 2006",
			"02 Jan 2006 Mon",
			"02 Jan 2006 Monday",
			"02 Jan 06",
			"02 Jan 06 Mon",
			"02 Jan 06 Monday"}, nil)
		selectorDate.Selected = userSettings.dateFormat
		selectorDate.OnChanged = func(s string) {
			fyneApp.Preferences().SetString("date", s)
			userSettings.dateFormat = s
		}

		selectorTime := widget.NewSelect([]string{"03:04:05 PM", "15:04:05"}, nil)
		selectorTime.Selected = userSettings.timeFormat
		selectorTime.OnChanged = func(s string) {
			fyneApp.Preferences().SetString("time", s)
			userSettings.timeFormat = s
		}

		locationWeather := widget.NewEntry()
		locationWeather.PlaceHolder = "Auto"
		locationWeather.OnChanged = func(s string) {
			fyneApp.Preferences().SetString("local", s)
			userSettings.location = s
		}

		catAlpha := widget.NewEntry()
		catAlpha.SetText(fmt.Sprint(userSettings.catTransparency))
		catAlpha.Validator = func(s string) error {
			textToNumber, err := strconv.Atoi(s)
			if err != nil {
				log.Println(err)
				return err
			}
			if textToNumber < 0 || textToNumber > 255 {
				return errors.New("Invalid range. Allowed: 0 to 255")
			}
			return nil
		}
		catAlpha.OnChanged = func(s string) {
			textToNumber, err := strconv.Atoi(s)
			if err != nil {
				log.Println(err)
			}
			fyneApp.Preferences().SetString("transparency", s)
			userSettings.catTransparency = textToNumber
		}

		formSettings := widget.NewForm(&widget.FormItem{
			Text:   "Date Style",
			Widget: selectorDate,
		},
			&widget.FormItem{
				Text:   "Hour Style",
				Widget: selectorTime,
			},
			&widget.FormItem{
				Text:     "Location",
				Widget:   locationWeather,
				HintText: "Location for weather services. Leave blank for automatic location, based on your network.",
			},
			&widget.FormItem{
				Text:     "Overlay Transparency",
				Widget:   catAlpha,
				HintText: "Overlay transparency, from 0 to 255. Default: 156. Restart to take effect.",
			})
		dialog.ShowForm("Flock Settings", "Ok", "Cancel", formSettings.Items, func(b bool) {
			if b {
				fyneWindow.Content().Refresh()
			}
		}, fyneWindow)
	}
	buttonSettings.OnTapped = showSettings

	buttonViewLogs := widget.NewButton("F3 View Logs", nil)
	buttonViewLogs.Importance = widget.LowImportance

	showAbout := func() {
		originalUrl, _ := url.Parse("https://noxyntious.github.io/clock2.html")
		eriUrl, _ := url.Parse("https://nijika.dev/")
		authorUrl, _ := url.Parse("https://lostdusty.dev.br")
		links := []*widget.Hyperlink{
			widget.NewHyperlink("Original Project", originalUrl),
			widget.NewHyperlink("Original Author", eriUrl),
			widget.NewHyperlink("Flock Author", authorUrl),
		}
		dialogx.ShowAboutWindow("Fyne cat clock", links, fyneApp)
	}

	buttonAbout := widget.NewButton("F4 About this program", showAbout)
	buttonAbout.Importance = widget.LowImportance

	openLogs := func() {
		logDir, _ := url.Parse(f.Name())
		fyneApp.OpenURL(logDir)
	}
	buttonViewLogs.OnTapped = openLogs

	helpLayout := container.NewVBox(container.NewCenter(welcomeText), buttonHelp, buttonSettings, buttonViewLogs, buttonAbout)
	modalHelp := widget.NewModalPopUp(helpLayout, fyneWindow.Canvas())
	buttonHelp.OnTapped = func() {
		modalHelp.Hide()
	}
	fyneWindow.Canvas().SetOnTypedKey(func(ke *fyne.KeyEvent) {
		switch ke.Name {
		case fyne.KeyF1:
			if !modalHelp.Hidden {
				modalHelp.Hide()
				return
			}
			modalHelp.Show()
		case fyne.KeyF2:
			showSettings()
		case fyne.KeyF3:
			openLogs()
		case fyne.KeyF4:
			showAbout()
		}

	})

	catArt := widget.NewLabelWithStyle(asciiCat, fyne.TextAlignLeading, fyne.TextStyle{Monospace: true})

	currentTime := time.Now()
	clockText := canvas.NewText(currentTime.Format(userSettings.timeFormat), colorLightBlue)
	dateText := canvas.NewText(currentTime.Format(userSettings.dateFormat), colorLightBlue)
	clockText.TextSize = 38
	dateText.TextSize = 26

	temperatureText := canvas.NewText("Loading...", colorLightBlue)
	weatherDetailsText := canvas.NewText("", colorLightBlue)
	temperatureText.TextSize = 24
	weatherDetailsText.TextSize = 16

	catShadow := canvas.NewRectangle(color.RGBA{R: 0, G: 0, B: 0, A: uint8(userSettings.catTransparency)})

	marginLeft := canvas.NewRectangle(color.Transparent)
	marginLeft.SetMinSize(fyne.Size{Width: 30, Height: 5})

	marginTop := canvas.NewRectangle(color.Transparent)
	marginTop.SetMinSize(fyne.Size{Width: 10, Height: 5})

	clockBox := container.NewVBox(clockText, dateText)
	weatherBox := container.NewVBox(temperatureText, weatherDetailsText)

	topBar := container.NewVBox(
		marginTop,
		container.NewHBox(marginLeft, clockBox, layout.NewSpacer(), catArt),
	)
	bottomBar := container.NewHBox(marginLeft, weatherBox, layout.NewSpacer())

	mainLayout := container.NewBorder(topBar, bottomBar, nil, nil, layout.NewSpacer())

	//go routine to update time
	go func() {
		ticker := time.NewTicker(time.Second)
		for range ticker.C {
			fyne.Do(func() {
				now := time.Now()
				clockText.Text = now.Format(userSettings.timeFormat)
				clockText.Refresh()
				if strconv.Itoa(now.Day()) != dateText.Text[:2] {
					dateText.Text = now.Format(userSettings.dateFormat)
					dateText.Refresh()
				}
			})

		}
	}()

	//go routine to update weather data, every hour
	getWeatherData := func() {
		updateWeather := func() {
			wttrinResponse, err := http.Get(fmt.Sprintf("https://wttr.is/%s?0dAFQT&format=%%t$%%w+|+%%h+|+%%p+|+%%u+UV", userSettings.location))
			if err != nil {
				log.Println(err)
				return
			}

			defer wttrinResponse.Body.Close()

			body, err := io.ReadAll(wttrinResponse.Body)
			if err != nil {
				log.Println(err)
				return
			}

			weatherData := strings.Split(string(body), "$")
			// Check if the split actually returned at least 2 parts to avoid index out of range panics
			if len(weatherData) >= 2 {
				temperatureWithoutExtraPlusSign, _ := strings.CutPrefix(weatherData[0], "+") //cut + from temperature
				fyne.Do(func() {
					temperatureText.Text = temperatureWithoutExtraPlusSign
					temperatureText.Refresh()

					weatherDetailsText.Text = weatherData[1]
					weatherDetailsText.Refresh()
				})

			}
		}

		go updateWeather()

		ticker := time.NewTicker(1 * time.Hour)
		for range ticker.C {
			go updateWeather()
		}
	}

	go getWeatherData()

	if userSettings.isFirstRun {
		fyneApp.Preferences().SetBool("firstrun", false)
		modalHelp.Show()
	}

	fyneWindow.SetContent(container.NewStack(mainLayout, catShadow))
	fyneWindow.ShowAndRun()
}

func loadUserPreferences() {
	s := Settings{}
	s.dateFormat = fyneApp.Preferences().StringWithFallback("date", "02/01/2006 Monday")
	s.timeFormat = fyneApp.Preferences().StringWithFallback("time", "03:04:05 PM")
	s.location = fyneApp.Preferences().StringWithFallback("local", "")
	s.catTransparency, _ = strconv.Atoi(fyneApp.Preferences().StringWithFallback("transparency", "156"))
	s.isFirstRun = fyneApp.Preferences().BoolWithFallback("firstrun", true)

	userSettings = s
}

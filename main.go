package main

import (
	"log"
	"os"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/go-rod/rod"
)

func main() {
	browser := rod.New().ControlURL(getWebSocketURL()).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://web.whatsapp.com")
	time.Sleep(5 * time.Second)
	page.MustSetViewport(912, 1368, 1, true)
	// wait to get the page completed
	page = page.MustWaitStable()

	f, err := os.OpenFile("testlogfile", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			log.Fatalf("error closing file: %v", err)
		}
	}(f)

	log.SetOutput(f)
	openChat(page, mainGroup)

	var mutex = &sync.Mutex{}

	msgCh := make(chan message, 10)
	// send the message
	go sendMessage(page, msgCh, mutex)
	go checkChannel(page, mainGroup, msgCh, mutex, 30*time.Second)
	go checkChannel(page, PortugueseGroup, msgCh, mutex, 1*time.Minute)
	go checkChannel(page, OtherEventsGroup, msgCh, mutex, 1*time.Minute)

	c := cron.New()
	// Every Monday at 12:00 PM
	_, err = c.AddFunc("43 10 * * 1", func() {
		sendScheduledMessage(page, msgCh, mutex)
	})
	if err != nil {
		return
	}

	// Every Friday at 12:00 PM we send the upcoming events
	_, err = c.AddFunc("40 17 * * 5", func() {
		sendUpcomingEvents(page, msgCh, mutex)
	})
	if err != nil {
		return
	}

	c.Start()

	ch := make(chan struct{})
	<-ch
}

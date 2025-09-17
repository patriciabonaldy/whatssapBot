package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
	"github.com/joho/godotenv"
)

func main() {
	browser := rod.New().ControlURL(getWebSocketURL()).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://web.whatsapp.com")
	time.Sleep(5 * time.Second)
	page.MustSetViewport(912, 1368, 1, false)
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

	msgCh := make(chan message)
	go sendWelcomeMessage(page, msgCh, mutex)
	go checkScammers(page, msgCh, mutex, mainGroup)
	go checkScammers(page, msgCh, mutex, OtherEventsGroup)
	//go checkScammers(page, msgCh, mutex, PortugueseGroup)
	go onboardingTrigger(page, msgCh, mutex, mainGroup)

	c := cron.New()
	// Every Monday at 12:00 PM
	_, err = c.AddFunc("00 11 * * 1", func() {
		sendScheduledMessage(page, msgCh, mutex)
	})
	if err != nil {
		return
	}

	// Every Friday at 12:00 PM we send the upcoming events
	_, err = c.AddFunc("40 13 * * 5", func() {
		sendUpcomingEvents(page, msgCh, mutex)
	})
	if err != nil {
		return
	}

	c.Start()

	ch := make(chan struct{})
	<-ch
}

func getWebSocketURL() string {
	resp, err := http.Get("http://127.0.0.1:9222/json/version")
	if err != nil {
		log.Fatalf("failed to get WebSocket URL %s: " + err.Error())
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.Fatalf("error closing file: %v", err)
		}
	}(resp.Body)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatalf("Failed to read response body: %v", err)
	}

	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	return data["webSocketDebuggerUrl"].(string)
}

func openChat(page *rod.Page, chatName string) {
	if chatName == communityGroup {
		openCommunityGroup(page)
		return
	}
	// Click the group
	page.MustElementR("span", chatName).MustClick()
	time.Sleep(5 * time.Second)
}

func openCommunityGroup(page *rod.Page) {
	// Click on the community menu button
	page.MustElement(`button[title="Subgroup switcher"]`).MustClick()
	time.Sleep(1 * time.Second)

	btn := page.MustElement("div._ak8o")
	btn.MustClick()
	time.Sleep(10 * time.Second)
}

func listenToAnnouncements(page *rod.Page) []string {
	// find the chat
	openChat(page, communityGroup)
	log.Println("Listening to messages in the announcements channel...")
	joiners := make(map[string]struct{})

	//get the last message
	text := page.MustElements("div.x6ikm8r.x10wlt62.xlyipyv").Last().MustText()
	firstLine := strings.Split(text, "\n")[0]

	if (strings.Contains(text, "joined") || strings.Contains(text, "Today")) && regexp.MustCompile(`\+\d[\d\s\-\(\)]*\s*joined`).MatchString(text) {
		log.Println("New join detected:", firstLine)
		remittent := strings.TrimSpace(strings.ReplaceAll(firstLine, "~", ""))
		noLineBreaks := strings.ReplaceAll(remittent, "\n", "")
		noLineBreaks = strings.ReplaceAll(noLineBreaks, "\r", "")
		joiners[noLineBreaks] = struct{}{}
	}

	remittent := make([]string, 0)
	for k := range joiners {
		remittent = append(remittent, k)
	}

	return remittent
}

func sendWelcomeMessage(page *rod.Page, msgCh chan message, mutex *sync.Mutex) {
	lastMessages := make(map[string]bool)

	for {
		mutex.Lock()
		// find the new joiners
		joiners := listenToAnnouncements(page)
		openChat(page, mainGroup)
		mutex.Unlock()
		for _, rem := range joiners {
			remittent := strings.TrimSpace(strings.ReplaceAll(rem, "~", ""))
			if remittent == "" {
				continue
			}
			if lastMessages[remittent] {
				continue
			}
			log.Println("rem:", remittent)
			go func() {
				msgCh <- message{
					msgType:   welcomeMsg,
					remittent: remittent,
					message:   welcomeMessage,
					chatName:  mainGroup,
				}
			}()
			// send the message
			sendMessage(page, msgCh, mutex)
			lastMessages[remittent] = true
		}
		time.Sleep(30 * time.Second)
	}
}

func checkScammers(page *rod.Page, msgCh chan message, mutex *sync.Mutex, groupName string) {
	for {
		mutex.Lock()
		// find the chat
		openChat(page, groupName)
		// find the last messages
		messages := page.MustElements(`div.message-in, div.message-out`)
		mutex.Unlock()
		for _, msg := range messages {
			msg.MustElements(`span.selectable-text`)
			text := strings.ToLower(msg.MustText())

			if strings.Contains(text, "stock") ||
				strings.Contains(text, "investment") ||
				strings.Contains(text, "crypto") ||
				strings.Contains(text, "forex") ||
				strings.Contains(text, "income") ||
				strings.Contains(text, "profit") ||
				strings.Contains(text, "trading") {

				// send the message
				log.Println("text:", text)

				//tags admins
				admins := []string{"John Burns", "Inara", "Paula", "Princeso", "Fábio"}

				go func() {
					msgCh <- message{
						msgType:  warningMsg,
						admins:   admins,
						message:  "This kind of message is not allowed on this group",
						chatName: groupName,
					}
				}()
				// send the message
				sendMessage(page, msgCh, mutex)
				goto sleep
			}
		}
	sleep:
		time.Sleep(5 * time.Minute)
	}
}

func sendScheduledMessage(page *rod.Page, msgCh chan message, mutex *sync.Mutex) {
	// find the chat
	openChat(page, mainGroup)
	// send the message
	go func() {
		msg := message{
			msgType:  proposeMsg,
			message:  "Hey Folks 👋\nGot any 💡 ideas for our Saturday walk 🚶? Feel free to share them 🙂 We'll vote later today!",
			chatName: mainGroup,
		}

		msgCh <- msg
	}()
	sendMessage(page, msgCh, mutex)
}

func sendUpcomingEvents(page *rod.Page, msgCh chan message, mutex *sync.Mutex) {
	var wg sync.WaitGroup
	wg.Add(1)
	// send the message
	go func() {
		defer wg.Done()
		msg := message{
			msgType:  proposeMsg,
			message:  "These are our upcoming events for this weekend 🚶",
			chatName: communityGroup,
		}
		msgCh <- msg
	}()

	sendMessage(page, msgCh, mutex)
	events, err := getUpcomingEvents()
	if err != nil {
		log.Fatal(err)
	}
	wg.Add(len(events))

	for _, event := range events {
		go func(e Edge) {
			defer wg.Done()
			msgCh <- message{
				msgType:  proposeMsg,
				message:  e.Node.Title,
				link:     e.Node.EventURL,
				chatName: communityGroup,
			}
		}(event)

		sendMessage(page, msgCh, mutex)
	}
	wg.Wait()
}

func onboardingTrigger(page *rod.Page, msgCh chan message, mutex *sync.Mutex, groupName string) {
	var mapText = make(map[string]bool)
	for {
		mutex.Lock()
		// find the chat
		openChat(page, groupName)
		// find the last messages
		messages := page.MustElements(`div.message-in, div.message-out`)
		mutex.Unlock()
		for _, msg := range messages {
			msg.MustElements(`span.selectable-text`)
			text := strings.ToLower(msg.MustText())

			if strings.Contains(text, "/calendar") && !strings.Contains(text, "tail-in") && !strings.Contains(text, "check upcoming walks sending a message with") && !mapText[text] {
				// send the message
				log.Println("text:", text)
				events, err := getUpcomingEvents()
				if err != nil {
					log.Fatal(err)
				}

				if len(events) == 0 {
					events = append(events, Edge{
						Node: Node{
							Title: "Sorry, No events yet, but stay tuned, we’ll be scheduling some soon!",
						},
					})
				}

				var wg sync.WaitGroup
				wg.Add(len(events))

				for _, event := range events {
					go func(e Edge) {
						defer wg.Done()
						msgCh <- message{
							msgType:  proposeMsg,
							message:  e.Node.Title,
							link:     e.Node.EventURL,
							chatName: mainGroup,
						}
					}(event)

					sendMessage(page, msgCh, mutex)
				}
				wg.Wait()
				mapText[text] = true
			}
		}
		goto sleep
	sleep:
		time.Sleep(30 * time.Second)
	}
}

func sendMessage(page *rod.Page, msgCh chan message, mutex *sync.Mutex) {
	msg := <-msgCh
	config := loadConfig()
	isDryRun, err := strconv.ParseBool(config["isDryRun"])
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}

	if isDryRun {
		return
	}

	mutex.Lock()
	defer mutex.Unlock()

	switch msg.msgType {
	case welcomeMsg:
		// find the chat
		openChat(page, msg.chatName)
		inputBox := page.MustElement(`div[contenteditable="true"][data-tab="10"]`)
		// Simulate a mention
		inputBox.MustInput("Hi ")
		err = page.Keyboard.Type('@')
		if err != nil {
			log.Printf("error typing @: %v", err)
			return
		}
		inputBox.MustInput(msg.remittent)
		err = page.Keyboard.Type(input.Enter)
		if err != nil {
			log.Printf("error typing the remittent: %v", err)
			return
		}

		inputBox.MustInput("\n ")
		inputBox.MustInput(msg.message)
		inputBox.MustInput("\n")
		inputBox.MustInput(welcomeMessage2)
		inputBox.MustInput("\n")
		inputBox.MustInput("Check upcoming walks sending a message with '/calendar' as text")
		inputBox.MustType(input.Enter)
	case proposeMsg:
		// find the chat
		openChat(page, msg.chatName)
		inputBox := page.MustElement(`div[contenteditable="true"][data-tab="10"]`)
		inputBox.MustInput(msg.message)
		if msg.venue != "" {
			inputBox.MustInput("\n")
			inputBox.MustInput(msg.venue)
		}
		if msg.link != "" {
			inputBox.MustInput("\n")
			inputBox.MustInput(msg.link)
			page.MustWaitLoad()
			time.Sleep(3 * time.Second)
		}
		time.Sleep(3 * time.Second)
		inputBox.MustType(input.Enter)
	case warningMsg:
		// find the chat
		openChat(page, msg.chatName)
		inputBox := page.MustElement(`div[contenteditable="true"][data-tab="10"]`)
		inputBox.MustInput(msg.message)
		inputBox.MustType(input.Enter)
		//tags admins
		for _, admin := range msg.admins {
			err = page.Keyboard.Type('@')
			if err != nil {
				return
			}
			inputBox.MustInput(admin)
			err = page.Keyboard.Type(input.Enter)
			if err != nil {
				return
			}
			inputBox.MustInput(" ")
		}

		inputBox.MustInput("\n")
		inputBox.MustInput("🚔⚡ please, delete that message!!")
		inputBox.MustInput("\n")
		inputBox.MustInput("🚨🚨🚔🚔🚔🚔🚔🚨🚨")
		inputBox.MustType(input.Enter)
	}
}

func loadConfig() map[string]string {
	envMap, err := godotenv.Read("config.env")
	if err != nil {
		log.Fatal("Error loading file:", err)
	}

	return envMap
}

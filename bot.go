package main

import (
	"log"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

var (
	lastMessages = make(map[string]bool)
	mapText      = make(map[string]bool)
)

func checkChannel(page *rod.Page, channel string, msgCh chan message, mutex *sync.Mutex, duration time.Duration) {
	for {
		defer func() {
			if err := recover(); err != nil {
				log.Println("recovered in checkChannel:", err)
			}
		}()
		time.Sleep(duration)
		page = page.MustWaitStable()
		getAnnouncementsAndMessages := func(params ...any) any {
			pg, ch := params[0].(*rod.Page), params[1].(string)
			log.Printf("Listening to messages in the %s channel...\n", channel)
			announcements := getAnnouncements(pg, ch)
			messages := getMessages(page)
			return []any{announcements, messages}
		}
		resp := Execute(page, channel, mutex, getAnnouncementsAndMessages, page, channel).([]any)
		if len(resp) == 0 {
			continue
		}

		if resp[0] != nil {
			announcements := resp[0].([]string)
			if len(announcements) > 0 {
				sendWelcomeMessage(msgCh, mutex, announcements, channel)
			}
		}

		if resp[1] != nil {
			messages := resp[1].(rod.Elements)

			if len(messages) > 0 {
				checkScammers(msgCh, messages, channel)
			}

			if channel == mainGroup {

				onboardingTrigger(msgCh, mutex, messages)
			}
		}

		time.Sleep(duration)
	}
}

func sendWelcomeMessage(msgCh chan message, mutex *sync.Mutex, joiners []string, chatName string) {
	// find the new joiners
	mutex.Lock()
	defer mutex.Unlock()

	for _, rem := range joiners {
		remittent := strings.TrimSpace(strings.ReplaceAll(rem, "~", ""))
		if remittent == "" {
			continue
		}
		if lastMessages[remittent] {
			continue
		}
		log.Println("rem:", remittent)
		lastMessages[remittent] = true
		msgCh <- message{
			msgType:   welcomeMsg,
			remittent: remittent,
			message:   getWelcomeMessage(chatName),
			chatName:  chatName,
		}
	}
}

func checkScammers(msgCh chan message, messages rod.Elements, chatName string) {
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
			admins := []string{"Inara", "Paula", "Princeso", "Fábio", "Julie AI thinknn"}
			msgCh <- message{
				msgType:  warningMsg,
				admins:   admins,
				message:  "This kind of message is not allowed on this group",
				chatName: chatName,
			}
		}
	}
}

func sendScheduledMessage(page *rod.Page, msgCh chan message, mutex *sync.Mutex) {
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

func onboardingTrigger(msgCh chan message, mutex *sync.Mutex, messages rod.Elements) {
	mutex.Lock()
	defer mutex.Unlock()
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
			}
			wg.Wait()
			mapText[text] = true
		}
	}
}

func sendMessage(page *rod.Page, msgCh chan message, mutex *sync.Mutex) {
	for {
		select {
		case msg := <-msgCh:
			config := loadConfig()
			isDryRun, err := strconv.ParseBool(config["isDryRun"])
			if err != nil {
				log.Fatalf("error opening file: %v", err)
			}

			if !isDryRun {
				mutex.Lock()
				// find the chat
				openChat(page, msg.chatName)
				switch msg.msgType {
				case welcomeMsg:
					inputBox := page.MustElement(`div[contenteditable="true"][data-tab="10"]`)
					// Simulate a mention
					if msg.chatName == mainGroup {
						inputBox.MustInput("Hi ")
					}
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
					inputBox.MustInput(getWelcomeMessage2(msg.chatName))
					inputBox.MustInput("\n")
					if msg.chatName == mainGroup {
						inputBox.MustInput("Check upcoming walks sending a message with '/calendar' as text")
					}
					inputBox.MustType(input.Enter)
				case proposeMsg:
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
				mutex.Unlock()
			}
		default:
		}
	}
}

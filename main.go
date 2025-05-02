package main

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/input"
)

const groupName = "10k steps"

func main() {
	// Conectarse a Chrome lanzado con remote debugging
	//browser := rod.New().ControlURL("ws://127.0.0.1:9222/devtools/browser/102d9244-b754-4fc1-8b52-18f0ce56f0d8").MustConnect()
	browser := rod.New().ControlURL(getWebSocketURL()).MustConnect()
	defer browser.MustClose()

	page := browser.MustPage("https://web.whatsapp.com")
	// wait to get the page completed
	page.MustWaitStable()
	// find the chat
	page.MustElementR("span", groupName).MustClick()
	time.Sleep(1 * time.Second)

	go sendMessage(page)
	ch := make(chan struct{})
	<-ch
}

func sendMessage(page *rod.Page) {
	const welcomeMessage = ", please introduce yourself 😊"
	lastMessages := make(map[string]bool)

	for {
		// find the chat
		page.MustElementR("span", groupName).MustClick()
		// find the last messages
		messages := page.MustElements(`div.message-in, div.message-out`)
		messages = page.MustElements("div.x6ikm8r.x10wlt62.xlyipyv")
		for _, msg := range messages {
			msg.MustElements(`span.selectable-text`)
			text := msg.MustText()

			if (strings.Contains(text, "joined using") || strings.Contains(text, "joined via")) && !strings.Contains(text, "joined from the community") {
				rem := strings.Split(text, "\n")[0]
				if lastMessages[rem] {
					continue
				}
				remittent := strings.TrimSpace(strings.ReplaceAll(rem, "~", ""))

				// send the message
				inputBox := page.MustElement(`div[contenteditable="true"][data-tab="10"]`)
				// Simulate a mention
				inputBox.MustInput("Hello ")
				err := page.Keyboard.Type('@')
				if err != nil {
					return
				}
				inputBox.MustInput(remittent)
				err = page.Keyboard.Type(input.Enter)
				if err != nil {
					return
				}
				time.Sleep(1 * time.Second)
				inputBox.MustInput(welcomeMessage)
				inputBox.MustType(input.Enter)
				lastMessages[rem] = true
			}
		}

		time.Sleep(20 * time.Second)
	}
}

func getWebSocketURL() string {
	resp, err := http.Get("http://127.0.0.1:9222/json/version")
	if err != nil {
		panic("Failed to get WebSocket URL: " + err.Error())
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var data map[string]interface{}
	json.Unmarshal(body, &data)

	return data["webSocketDebuggerUrl"].(string)
}

func checkScammers(page *rod.Page) {
	lastMessages := make(map[string]bool)
	for {
		// find the last messages
		messages := page.MustElements(`div.message-in, div.message-out`)
		for _, msg := range messages {
			msg.MustElements(`span.selectable-text`)
			text := msg.MustText()

			if strings.Contains(text, "stock") ||
				strings.Contains(text, "option") ||
				strings.Contains(text, "money") ||
				strings.Contains(text, "investment") ||
				strings.Contains(text, "crypto") ||
				strings.Contains(text, "forex") ||
				strings.Contains(text, "income") ||
				strings.Contains(text, "profit") ||
				strings.Contains(text, "trading") {
				if lastMessages[text] {
					continue
				}
				//delete the message
				// Right click (context click) on it
				msg.MustEval(`el => {
								const rect = el.getBoundingClientRect();
								const evt = new MouseEvent('contextmenu', {
									bubbles: true,
									cancelable: true,
									view: window,
									clientX: rect.left + rect.width / 2,
									clientY: rect.top + rect.height / 2,
								});
								el.dispatchEvent(evt);
							}`)

				// Wait for the menu and click "Delete message"
				time.Sleep(1 * time.Second)
				page.MustElementR("div[role='button']", "Delete message").MustClick()

				// Click "Delete for everyone"
				time.Sleep(1 * time.Second)
				page.MustElementR("div[role='button']", "Delete for everyone").MustClick()

				lastMessages[text] = true
				rem := strings.Split(text, "\n")[0]
				remittent := strings.TrimSpace(strings.ReplaceAll(rem, "~", ""))

				// send the message
				inputBox := page.MustElement(`div[contenteditable="true"][data-tab="10"]`)
				// Simula una mención
				err := page.Keyboard.Type('@')
				if err != nil {
					return
				}
				inputBox.MustInput(remittent)
				err = page.Keyboard.Type(input.Enter)
				if err != nil {
					return
				}
				time.Sleep(1 * time.Second)
				inputBox.MustType(input.Enter)
			}
		}

		time.Sleep(20 * time.Second)
	}
}

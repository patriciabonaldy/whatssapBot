package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/go-rod/rod"
)

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

func getAnnouncements(page *rod.Page, channel string) rod.Elements {
	openChat(page, channel)
	//get announcements
	announcements := page.MustElements("div._amk4.false._amkb")
	if len(announcements) == 0 {
		return nil
	}
	if len(announcements) >= 10 {
		announcements = announcements[:10]
	}

	return announcements
}

func getMessages(page *rod.Page, channel string) rod.Elements {
	//get messages
	messages := page.MustElements("div._amk4.false._amkd._amk5")
	if len(messages) == 0 {
		return nil
	}
	if len(messages) >= 10 {
		messages = messages[:10]
	}

	return messages
}

func listenToAnnouncements(elements rod.Elements) []string {
	// find the chat
	log.Println("Listening to messages in the announcements channel...")
	joiners := make(map[string]struct{})

	for _, announcement := range elements {
		text := announcement.MustText()
		firstLine := strings.Split(text, "\n")[0]
		if (strings.Contains(text, "joined") || strings.Contains(text, "Today")) && regexp.MustCompile(`\+\d[\d\s\-\(\)]*\s*joined`).MatchString(text) {
			log.Println("New join detected:", firstLine)
			remittent := strings.TrimSpace(strings.ReplaceAll(firstLine, "~", ""))
			noLineBreaks := strings.ReplaceAll(remittent, "\n", "")
			noLineBreaks = strings.ReplaceAll(noLineBreaks, "\r", "")
			joiners[noLineBreaks] = struct{}{}
		}
	}

	remittent := make([]string, 0)
	for k := range joiners {
		remittent = append(remittent, k)
	}

	return remittent
}

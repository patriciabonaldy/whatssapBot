package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
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
func Execute(page *rod.Page, chatName string, mutex *sync.Mutex, callback func(...any) any, params ...any) any {
	mutex.Lock()
	defer mutex.Unlock()
	openChat(page, chatName)
	result := callback(params...)
	return result
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
	page.MustElement(`button[aria-label="Subgroup switcher"]`).MustClick()
	//page.MustElement(`button[title="Subgroup switcher"]`).MustClick()
	time.Sleep(1 * time.Second)

	btn := page.MustElement("div._ak8o")
	btn.MustClick()
	time.Sleep(10 * time.Second)
}

func getAnnouncements(params ...any) any {
	page := params[0].(*rod.Page)
	//get announcements
	announcements := page.MustElements("div._amk4.false._amkb")
	if len(announcements) == 0 {
		return nil
	}

	return listenToAnnouncements(announcements)
}

func getMessages(params ...any) any {
	page := params[0].(*rod.Page)
	messages := page.MustElements("div._amk4.false._amkd")
	if len(messages) == 0 {
		return nil
	}

	return messages
}

func listenToAnnouncements(params ...any) any {
	remittent := make([]string, 0)
	if params[0] == nil {
		return remittent
	}

	elements := params[0].(rod.Elements)
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

	for k := range joiners {
		remittent = append(remittent, k)
	}

	return remittent
}

func getWelcomeMessage(chatName string) string {
	if chatName == PortugueseGroup {
		return "Bem-vindo ao grupo de Falemos em Portugues! 👋"
	}

	return fmt.Sprintf(welcomeMessage, chatName)
}

func getWelcomeMessage2(chatName string) string {
	if chatName == PortugueseGroup {
		return "Conta um pouco sobre você quando tiver um tempinho!"
	}

	return welcomeMessage2
}

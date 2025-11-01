package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/mdp/qrterminal/v3"
	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"

	_ "github.com/mattn/go-sqlite3"
)

const (
	dbPath = "./people.db"
)

type WhatSSapClient struct {
	client *whatsmeow.Client
}

func NewClient() WhatSSapClient {
	dbLog := waLog.Stdout("Database", "DEBUG", true)
	ctx := context.Background()
	container, err := sqlstore.New(ctx, "sqlite3", "file:examplestore.db?_foreign_keys=on", dbLog)
	if err != nil {
		log.Fatal("failed openning the DB:", err)
	}
	defer container.Close()

	deviceStore, err := container.GetFirstDevice(ctx)
	if err != nil {
		log.Fatal(err)
	}

	clientLog := waLog.Stdout("Client", "DEBUG", true)
	client := whatsmeow.NewClient(deviceStore, clientLog)
	client.AddEventHandler(HandleEvents)

	go func() {
		// Listen to Ctrl+C (you can also do something else that prevents the program from exiting)
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		client.Disconnect()
	}()

	if client.Store.ID == nil {
		// No ID stored, new login
		qrChan, _ := client.GetQRChannel(context.Background())
		err = client.Connect()
		if err != nil {
			panic(err)
		}
		for evt := range qrChan {
			if evt.Event == "code" {
				// Render the QR code here
				// e.g. qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
				// or just manually `echo 2@... | qrencode -t ansiutf8` in a terminal
				qrterminal.GenerateHalfBlock(evt.Code, qrterminal.L, os.Stdout)
			} else {
				fmt.Println("Login event:", evt.Event)
			}
		}
	} else {
		// Already logged in, just connect
		err = client.Connect()
		if err != nil {
			panic(err)
		}
	}

	return WhatSSapClient{client}
}

var allowedGroups = map[string]string{
	"1234567890-123456@g.us": mainGroup,        // grupo 1
	"9876543210-789012@g.us": PortugueseGroup,  // grupo 2
	"9876543210-789013@g.us": OtherEventsGroup, // grupo 2
}

var msgCh chan message
var mutex *sync.Mutex

func HandleEvents(evt interface{}) {
	switch v := evt.(type) {
	case *events.JoinedGroup: // new joiners
		groupName := v.GroupInfo.Name
		if _, ok := allowedGroups[groupName]; !ok {
			return
		}

		var joiners = make([]string, 0)
		for _, p := range v.Participants {
			joiners = append(joiners, p.PhoneNumber.String())
		}
		go sendWelcomeMessage(msgCh, mutex, joiners, groupName)
	case *events.Message:
		if !v.Info.MessageSource.IsGroup {
			return
		}

		if v.Info.MessageSource.IsGroup {
			groupID := v.Info.Chat.String()
			groupName, ok := allowedGroups[groupID]
			if !ok {
				return
			}

			msg := v.Message.GetConversation()
			go checkScammer(msgCh, msg, groupName)
			if groupName == mainGroup {
				go onboardingTrigger2(msgCh, mutex, []string{msg})
			}
		}
	}
}

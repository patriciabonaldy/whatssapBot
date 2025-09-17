package main

import "time"

type message struct {
	msgType   msgType
	remittent string
	message   string
	venue     string
	link      string
	admins    []string
	chatName  string
}
type msgType string

const welcomeMsg msgType = "welcomeMsg"
const proposeMsg msgType = "proposeMsg"
const warningMsg msgType = "warningMsg"

type Node struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	EventURL    string `json:"eventUrl"`
	Description string `json:"description"`
	Group       struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Timezone string `json:"timezone"`
		Typename string `json:"__typename"`
	} `json:"group"`
	FeeSettings interface{} `json:"feeSettings"`
	Venue       struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		Address  string `json:"address"`
		City     string `json:"city"`
		State    string `json:"state"`
		Country  string `json:"country"`
		Typename string `json:"__typename"`
	} `json:"venue"`
	DateTime    time.Time `json:"dateTime"`
	CreatedTime string    `json:"createdTime"`
	EndTime     time.Time `json:"endTime"`
	Going       struct {
		TotalCount int    `json:"totalCount"`
		Typename   string `json:"__typename"`
	} `json:"going"`
	IsAttending  bool          `json:"isAttending"`
	IsOnline     bool          `json:"isOnline"`
	EventType    string        `json:"eventType"`
	Status       string        `json:"status"`
	RsvpState    string        `json:"rsvpState"`
	Actions      []interface{} `json:"actions"`
	RsvpSettings struct {
		RsvpsClosed bool   `json:"rsvpsClosed"`
		Typename    string `json:"__typename"`
	} `json:"rsvpSettings"`
	IsNetworkEvent bool          `json:"isNetworkEvent"`
	NetworkEvent   interface{}   `json:"networkEvent"`
	SocialLabels   []interface{} `json:"socialLabels"`
	Typename       string        `json:"__typename"`
}
type Edge struct {
	Node     Node   `json:"node"`
	Typename string `json:"__typename"`
}
type Events struct {
	TotalCount int `json:"totalCount"`
	PageInfo   struct {
		EndCursor   string `json:"endCursor"`
		HasNextPage bool   `json:"hasNextPage"`
		Typename    string `json:"__typename"`
	} `json:"pageInfo"`
	Edges    []Edge `json:"edges"`
	Typename string `json:"__typename"`
}
type MeetupResponse struct {
	Data struct {
		GroupByUrlname struct {
			ID       string `json:"id"`
			Events   Events `json:"events"`
			Typename string `json:"__typename"`
		} `json:"groupByUrlname"`
	} `json:"data"`
}

const mainGroup = "10k steps"
const OtherEventsGroup = "Other events"
const PortugueseGroup = "Falemos em Portuguese"
const communityGroup = "Announcements"
const welcomeMessage = "Welcome to the 10k steps group! 👋"
const welcomeMessage2 = "Tell us a bit about yourself when you get a chance!"

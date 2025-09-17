package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"time"
)

func getUpcomingEvents() ([]Edge, error) {
	url := "https://www.meetup.com/gql2"
	today := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	config := loadConfig()
	meetupHash := config["meetupHash"]
	//"2025-06-07T10:50:49.094Z"
	payload := strings.NewReader(fmt.Sprintf(`{
    "operationName": "getUpcomingGroupEvents",
    "variables": {
        "urlname": "porto-10000-steps-walk",
        "afterDateTime": "%s"
    },
    "extensions": {
        "persistedQuery": {
            "version": 1,
            "sha256Hash": "%s"
        }
    }}`, today, meetupHash))

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, payload)

	if err != nil {
		return nil, err
	}
	req.Header.Add("content-type", "application/json")
	req.Header.Add("referer", "https://www.meetup.com/porto-10000-steps-walk/events/")
	req.Header.Add("x-meetup-view-id", "0421dd36-fe26-46cf-91f8-69fe8d4465d9")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/125.0.0.0 Safari/537.36")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		return nil, fmt.Errorf("status code error: %d %s", res.StatusCode, res.Status)
	}

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response MeetupResponse
	err = json.Unmarshal(data, &response)
	if err != nil {
		return nil, err
	}
	fmt.Println("Status Code:", res.StatusCode)
	fmt.Println("Response Body:", response)

	edges := response.Data.GroupByUrlname.Events.Edges
	var events []Edge
	for _, event := range edges {
		if event.Node.Status == "CANCELLED" ||
			slices.Contains([]string{"NOT_OPEN_YET", "PAST"}, event.Node.RsvpState) {
			continue
		}

		date := event.Node.DateTime
		if !date.After(time.Now().UTC()) {
			continue
		}

		events = append(events, event)
	}

	return events, nil
}

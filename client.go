package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"sort"
	"strings"
	"time"
)

func getUpcomingEvents(groupName string) ([]Edge, error) {
	url := "https://www.meetup.com/gql2"
	today := time.Now().UTC().Format("2006-01-02T15:04:05.000Z")

	config := loadConfig()
	meetupHash := config["meetupHash"]
	//"2025-06-07T10:50:49.094Z"
	payload := strings.NewReader(fmt.Sprintf(`{
    "operationName": "getUpcomingGroupEvents",
    "variables": {
        "urlname": "%s",
        "afterDateTime": "%s"
    },
    "extensions": {
        "persistedQuery": {
            "version": 1,
            "sha256Hash": "%s"
        }
    }}`, groupName, today, meetupHash))

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, payload)

	if err != nil {
		return nil, err
	}
	req.Header.Add("accept", "*/*")
	req.Header.Add("content-type", "application/json")
	req.Header.Add("referer", "https://www.meetup.com/porto-10000-steps-walk/events/")
	req.Header.Add("Cookie", "MEETUP_BROWSER_ID=id=65a9af42-a7e7-4c26-abd4-c252a261fe60; MEETUP_TRACK=id=146a4ad6-69f2-4620-b113-d3537506613a; SIFT_SESSION_ID=ffd37912-b21c-495f-aea6-1ffda73c8bf0")
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36'")

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
		if !date.After(time.Now().UTC()) || event.Node.EndTime.After(time.Now().AddDate(0, 0, 15).UTC()) {
			continue
		}

		events = append(events, event)
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].Node.EndTime.Before(events[j].Node.EndTime)
	})
	if len(events) > 1 {
		return []Edge{events[0]}, nil
	}

	return events, nil
}

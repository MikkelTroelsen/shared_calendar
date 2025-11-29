package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	ics "github.com/arran4/golang-ical"
)

type CalendarToCopy struct {
	Url  string `json:"url"`
	Name string `json:"name"` // Name is added all event titles. To ignore use ""
}

func main() {
	calendarsToCopy, _ := getCalendarFromJson("calendars.json")
	sharedCal := ics.NewCalendar()
	sharedCal.SetMethod(ics.MethodPublish)
	sharedCal.SetName("Combined Calendar")

	for _, calendar := range calendarsToCopy {
		fmt.Println(calendar.Name)
		for _, event := range getCalendar(calendar.Url).Events() {
			copyEvent(event, sharedCal, calendar.Name)
		}
	}

	// for _, event := range sharedCal.Events() {
	// 	title := event.GetProperty(ics.ComponentPropertySummary).Value
	// 	start, _ := event.GetStartAt()
	// 	end, _ := event.GetEndAt()
	// 	fmt.Println(title, start, end)
	// }

	sharedCalString := sharedCal.Serialize()
	println(sharedCalString)
}

func getCalendarFromJson(path string) ([]CalendarToCopy, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var calendars []CalendarToCopy
	if err := json.NewDecoder(file).Decode(&calendars); err != nil {
		return nil, err
	}
	return calendars, nil
}

func getCalendar(url string) *ics.Calendar {
	url = strings.Replace(url, "webcal://", "https://", 1)
	cal, _ := ics.ParseCalendarFromUrl(url)
	return cal
}

func copyEvent(srcEvent *ics.VEvent, dstCal *ics.Calendar, calendarName string) *ics.VEvent {
	// Create new event in dest cal
	newEvent := dstCal.AddEvent(srcEvent.GetProperty(ics.ComponentPropertyUniqueId).Value)

	if summery := srcEvent.GetProperty(ics.ComponentPropertySummary); summery != nil {
		newEvent.SetSummary(summery.Value + " - " + calendarName)
	}

	if description := srcEvent.GetProperty(ics.ComponentPropertyDescription); description != nil {
		newEvent.SetDescription(description.Value)
	}

	if location := srcEvent.GetProperty(ics.ComponentPropertyLocation); location != nil {
		newEvent.SetLocation(location.Value)
	}

	if rRule := srcEvent.GetProperty(ics.ComponentPropertyRrule); rRule != nil {
		newEvent.SetProperty(ics.ComponentPropertyRrule, rRule.Value)
	}

	for _, ex := range srcEvent.GetProperties(ics.ComponentPropertyExdate) {
		newEvent.SetProperty(ics.ComponentPropertyExdate, ex.Value)
	}

	for _, r := range srcEvent.GetProperties(ics.ComponentPropertyRdate) {
		newEvent.SetProperty(ics.ComponentPropertyRdate, r.Value)
	}

	start, _ := srcEvent.GetStartAt()
	newEvent.SetStartAt(start)

	end, _ := srcEvent.GetEndAt()
	newEvent.SetEndAt(end)

	return newEvent
}

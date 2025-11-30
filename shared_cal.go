package main

import (
	"encoding/json"
	"os"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
)

type CalendarToCopy struct {
	Url  string `json:"url"`
	Name string `json:"name"` // Name is added all event titles. To ignore use ""
}

type TrackedEvent struct {
	Event        *ics.VEvent
	LastModified *time.Time
}

func getIcal() (string, error) {
	calendarsToCopy, err := getCalendarFromJson("calendars.json")
	if err != nil {
		return "", err
	}
	sharedCal := ics.NewCalendar()
	sharedCal.SetMethod(ics.MethodPublish)
	sharedCal.SetName("Combined Calendar")

	uidLastModtifiedMap := make(map[string]TrackedEvent)

	syncCalendars(uidLastModtifiedMap, calendarsToCopy, sharedCal)
	sharedCalString := sharedCal.Serialize()
	return sharedCalString, nil
}

func syncCalendars(cacheMap map[string]TrackedEvent, calendarsToSync []CalendarToCopy, sharedCal *ics.Calendar) {
	toDeleteMap := make(map[string]bool)

	for uid := range cacheMap {
		toDeleteMap[uid] = true
	}

	for _, calendar := range calendarsToSync {
		ical := getCalendar(calendar.Url)
		for _, event := range ical.Events() {
			trackedEvent, exist := cacheMap[event.GetProperty(ics.ComponentPropertyUniqueId).Value]
			eventUid := event.GetProperty(ics.ComponentPropertyUniqueId).Value

			// New Event
			if !exist {
				createEventFromSourceEvent(sharedCal, event, calendar.Name, cacheMap)
				delete(toDeleteMap, eventUid)
				continue
			}

			lastModified := event.GetProperty(ics.ComponentPropertyLastModified)

			// Event have no last modified value. Therefore, an update is required to ensure accuracy
			if lastModified == nil {
				setEventValues(trackedEvent.Event, event, calendar.Name, cacheMap)
				delete(toDeleteMap, eventUid)
				continue
			}

			// If there is no tracked modified time update event
			if trackedEvent.LastModified == nil {
				setEventValues(trackedEvent.Event, event, calendar.Name, cacheMap)
				delete(toDeleteMap, eventUid)
				continue
			}

			lastModifiedTime, err := time.Parse(lastModified.Value, lastModified.Value)
			if err != nil {
				lastModifiedTime, _ = time.Parse("20060102T150405", "20060102T150405")
			}
			trackedTime := *trackedEvent.LastModified
			// If the two tracked times are not the same update event
			if trackedTime.Compare(lastModifiedTime) != 0 {
				setEventValues(trackedEvent.Event, event, calendar.Name, cacheMap)
				delete(toDeleteMap, eventUid)
				continue
			}
		}
	}

	// Remove events that do not exists any more
	for uid := range toDeleteMap {
		sharedCal.RemoveEvent(uid)
	}
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

func createEventFromSourceEvent(dstCal *ics.Calendar, srcEvent *ics.VEvent, calendarName string, cacheMap map[string]TrackedEvent) {
	// Create new event in dest cal
	newEvent := dstCal.AddEvent(srcEvent.GetProperty(ics.ComponentPropertyUniqueId).Value)
	setEventValues(newEvent, srcEvent, calendarName, cacheMap)
}

func setEventValues(event *ics.VEvent, srcEvent *ics.VEvent, calendarName string, cacheMap map[string]TrackedEvent) {
	if summery := srcEvent.GetProperty(ics.ComponentPropertySummary); summery != nil {
		event.SetSummary(summery.Value + " - " + calendarName)
	}

	if description := srcEvent.GetProperty(ics.ComponentPropertyDescription); description != nil {
		event.SetDescription(description.Value)
	}

	if location := srcEvent.GetProperty(ics.ComponentPropertyLocation); location != nil {
		event.SetLocation(location.Value)
	}

	if rRule := srcEvent.GetProperty(ics.ComponentPropertyRrule); rRule != nil {
		event.SetProperty(ics.ComponentPropertyRrule, rRule.Value)
	}

	for _, ex := range srcEvent.GetProperties(ics.ComponentPropertyExdate) {
		event.SetProperty(ics.ComponentPropertyExdate, ex.Value)
	}

	for _, r := range srcEvent.GetProperties(ics.ComponentPropertyRdate) {
		event.SetProperty(ics.ComponentPropertyRdate, r.Value)
	}

	lastModified := srcEvent.GetProperty(ics.ComponentPropertyLastModified)
	var modTime time.Time
	if lastModified != nil {
		time, _ := time.Parse(lastModified.Value, lastModified.Value)
		modTime = time
		event.SetLastModifiedAt(time)
	}

	start, _ := srcEvent.GetStartAt()
	event.SetStartAt(start)

	end, _ := srcEvent.GetEndAt()
	event.SetEndAt(end)

	uid := event.GetProperty(ics.ComponentPropertyUniqueId)
	cacheMap[uid.Value] = TrackedEvent{Event: event, LastModified: &modTime}
}

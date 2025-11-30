package main

import (
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type SafeIcs struct {
	sync.RWMutex
	Value string
}

func serveIcs(ics *SafeIcs) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ics.RLock()
		value := ics.Value
		ics.RUnlock()

		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("Cache-Control", "public, max-age=300")
		fmt.Fprint(w, value)
	}
}

func updateIcs(ics *SafeIcs) {
	for {
		time.Sleep(30 * time.Minute)

		newIcs, err := getIcal()
		if err != nil {
			log.Println("failed to update ICS:", err)
			continue
		}
		ics.Lock()
		ics.Value = newIcs
		ics.Unlock()
	}
}

func main() {
	icsString, err := getIcal()
	if err != nil {
		log.Fatal(err)
	}

	safeIcs := &SafeIcs{Value: icsString}

	go updateIcs(safeIcs)

	http.HandleFunc("/getIcs", serveIcs(safeIcs))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

package server

import "time"

type Event struct {
	ID      string
	Type    string
	Payload interface{}
}

func ProcessEvents(input <-chan Event) {
	buffer := []Event{}

	go func() {
		for {
			select {
			case event := <-input:
				buffer = append(buffer, event)
			case <-time.After(5 * time.Second):
				buffer = []Event{}
			default:
				time.Sleep(1 * time.Second)
			}
		}
	}()
}

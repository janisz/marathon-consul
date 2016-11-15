package web

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/allegro/marathon-consul/metrics"
)

type EventHandler struct {
	eventQueue chan event
}

func newWebHandler(eventQueue chan event) *EventHandler {
	return &EventHandler{eventQueue: eventQueue}
}

const (
	statusUpdateEvent        = "status_update_event"
	healthStatusChangedEvent = "health_status_changed_event"
	unsuportedEventType      = "Unsuported"
)

func (h *EventHandler) Handle(w http.ResponseWriter, r *http.Request) {
	metrics.Time("events.response", func() {
		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			log.WithError(err).Debug("Malformed request")
			handleBadRequest(err, w)
			return
		}
		log.WithField("Body", string(body)).Debug("Received request")

		eventType := eventType(body)
		log.WithField("EventType", eventType).Debug("Received event")
		metrics.Mark("events.requests." + eventType)

		if eventType == unsuportedEventType {
			drop(w)
			return
		}

		h.eventQueue <- event{eventType: eventType, body: body, timestamp: time.Now()}
		accepted(w)
	})
}

func eventType(body []byte) string {
	if bytes.Contains(body, []byte("\""+statusUpdateEvent+"\"")) {
		return statusUpdateEvent
	} else if bytes.Contains(body, []byte("\""+healthStatusChangedEvent+"\"")) {
		return healthStatusChangedEvent
	}
	return unsuportedEventType
}

func handleBadRequest(err error, w http.ResponseWriter) {
	metrics.Mark("events.response.error.400")
	w.WriteHeader(http.StatusBadRequest)
	log.WithError(err).Debug("Returning 400 due to malformed request")
	fmt.Fprintln(w, err.Error())
}

func accepted(w http.ResponseWriter) {
	metrics.Mark("events.response.202")
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintln(w, "OK")
}

func drop(w http.ResponseWriter) {
	metrics.Mark("events.response.200")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "DROP")
}

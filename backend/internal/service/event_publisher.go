package service

import "task-manager/internal/events"

type EventPublisher interface {
	Publish(event events.Event)
}

func publishEvent(publisher EventPublisher, event events.Event) {
	if publisher != nil {
		publisher.Publish(event)
	}
}

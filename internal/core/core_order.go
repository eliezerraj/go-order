package core

import (
	"time"
)

type Order struct {
	ID				int			`json:"id,omitempty"`
	PK				string		`dynamodbav:"pk"`
	SK				string		`dynamodbav:"sk"`
	OrderID			string  	`json:"order_id,omitempty"`
	PersonID		string  	`json:"person_id,omitempty"`
	ProductID		[]string  	`json:"products_id,omitempty"`
	Status			string  	`json:"status,omitempty"`
	Currency		string  	`json:"currency,omitempty"`
	Amount			float64 	`json:"amount,omitempty"`
	CreateAt		time.Time 	`json:"create_at,omitempty"`
	UpdateAt		*time.Time 	`json:"update_at,omitempty"`
	TenantID		string  	`json:"tenant_id,omitempty"`
}

type Event struct {
	Key			string      `json:"key"`
    EventDate   time.Time   `json:"event_date"`
    EventType   string      `json:"event_type"`
    EventData   *EventData   `json:"event_data"`
}

type EventData struct {
    Order   *Order    `json:"order"`
}
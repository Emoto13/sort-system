package service

import (
	"github.com/Emoto13/sort-system/fulfillment-service/state"
	"github.com/Emoto13/sort-system/gen"
)

type FulfillmentServiceParameters struct {
	SortingRobot gen.SortingRobotClient
	State        state.State
	Orders       chan []*gen.Order
}

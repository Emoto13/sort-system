package state

import (
	"sync"

	"github.com/Emoto13/sort-system/gen"
	"github.com/preslavmihaylov/ordertocubby"
)

type State interface {
	
}

type StateManager struct {
	ItemCodeToCubbyIds map[string][]*gen.Cubby
	CubbyIdToOrderId   map[string]string
	NumberOfItems      int
	mu                 sync.Mutex
}

func NewStateManager() *StateManager {
	return &StateManager{
		ItemCodeToCubbyIds: make(map[string][]*gen.Cubby),
		CubbyIdToOrderId:   make(map[string]string),
		NumberOfItems:      0,
		mu:                 sync.Mutex{},
	}
}

func (sm *StateManager) mapItemCodesToCubby(items []*gen.Item, cubby *gen.Cubby) {
	for _, item := range items {
		sm.ItemCodeToCubbyIds[item.Code] = append(sm.ItemCodeToCubbyIds[item.Code], cubby)
	}
}

func (sm *StateManager) getCubbyIdByOrderId(orderId string, times int) string {
	cubbyId := ordertocubby.Map(orderId, uint32(times), 10)
	attemptsToAvoidCollision := 1
	for true {
		if _, present := sm.CubbyIdToOrderId[cubbyId]; !present {
			break
		}
		cubbyId = ordertocubby.Map(orderId, uint32(times+attemptsToAvoidCollision), 10)
		attemptsToAvoidCollision++
	}
	return cubbyId
}

func (sm *StateManager) GetPreparedOrders(orders []*gen.Order) []*gen.PreparedOrder {
	preparedOrders := []*gen.PreparedOrder{}

	for i, order := range orders {
		cubbyId := sm.getCubbyIdByOrderId(order.Id, i)
		sm.CubbyIdToOrderId[cubbyId] = order.Id

		cubby := &gen.Cubby{Id: cubbyId}
		preparedOrder := &gen.PreparedOrder{Order: order, Cubby: cubby}
		preparedOrders = append(preparedOrders, preparedOrder)

		sm.mapItemCodesToCubby(order.Items, cubby)
		sm.NumberOfItems += len(order.Items)
	}

	return preparedOrders
}

func (sm *StateManager) GetCubbyByItemCode(itemCode string) *gen.Cubby {
	cubby := sm.ItemCodeToCubbyIds[itemCode][0]
	sm.ItemCodeToCubbyIds[itemCode] = sm.ItemCodeToCubbyIds[itemCode][1:]
	return cubby
}

package state

import (
	"sync"

	"github.com/Emoto13/sort_system/gen"
	"github.com/preslavmihaylov/ordertocubby"
)

type StateManager struct {
	itemCodeToCubbyIds map[string][]*gen.Cubby
	cubbyIdToOrderId   map[string]string
	numberOfItems      int
	mu                 sync.Mutex
}

func NewStateManager() *StateManager {
	return &StateManager{
		itemCodeToCubbyIds: make(map[string][]*gen.Cubby),
		cubbyIdToOrderId:   make(map[string]string),
		numberOfItems:      0,
		mu:                 sync.Mutex{},
	}
}

func (sm *StateManager) mapItemCodesToCubby(items []*gen.Item, cubby *gen.Cubby) {
	for _, item := range items {
		sm.itemCodeToCubbyIds[item.Code] = append(sm.itemCodeToCubbyIds[item.Code], cubby)
	}
}

func (sm *StateManager) getCubbyIdByOrderId(orderId string, times int) string {
	cubbyId := ordertocubby.Map(orderId, uint32(times), 10)
	attemptsToAvoidCollision := 1
	for true {
		if _, present := sm.cubbyIdToOrderId[cubbyId]; !present {
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
		sm.cubbyIdToOrderId[cubbyId] = order.Id

		cubby := &gen.Cubby{Id: cubbyId}
		preparedOrder := &gen.PreparedOrder{Order: order, Cubby: cubby}
		preparedOrders = append(preparedOrders, preparedOrder)

		sm.mapItemCodesToCubby(order.Items, cubby)
		sm.numberOfItems += len(order.Items)
	}

	return preparedOrders
}

func (sm *StateManager) GetCubbyByItemCode(itemCode string) *gen.Cubby {
	cubby := sm.itemCodeToCubbyIds[itemCode][0]
	sm.itemCodeToCubbyIds[itemCode] = sm.itemCodeToCubbyIds[itemCode][1:]
	return cubby
}

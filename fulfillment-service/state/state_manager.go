package state

import (
	"fmt"
	"sync"

	"github.com/Emoto13/sort-system/gen"
	"github.com/preslavmihaylov/ordertocubby"
)

type State interface {
	GetCubbyByItemCode(itemCode string) (*gen.Cubby, error)
	GetPreparedOrders(orders []*gen.Order) []*gen.PreparedOrder
	GetOrderStateById(orderId string) (gen.OrderState, error)
	SetOrdersState(orders []*gen.Order, status gen.OrderState) error
	SetOrderStateById(orderId string, status gen.OrderState) error
	GetFulfillmentStatusByOrderId(orderId string) ([]*gen.FulfillmentStatus, error)
	GetFulfillmentStatusOfAllOrders() ([]*gen.FulfillmentStatus, error)
	Clear()
}

type stateManager struct {
	ItemCodeToCubby  map[string][]*gen.Cubby
	CubbyIdToOrderId map[string]string
	OrderIdToCubby   map[string]*gen.Cubby
	OrderIdToItems   map[string][]*gen.Item
	OrderIdToState   map[string]gen.OrderState
	mu               sync.RWMutex
}

func New() State {
	return &stateManager{
		ItemCodeToCubby:  make(map[string][]*gen.Cubby),
		CubbyIdToOrderId: make(map[string]string),
		OrderIdToCubby:   make(map[string]*gen.Cubby),
		OrderIdToItems:   make(map[string][]*gen.Item),
		OrderIdToState:   make(map[string]gen.OrderState),
		mu:               sync.RWMutex{},
	}
}

func (sm *stateManager) mapItemCodesToCubby(items []*gen.Item, cubby *gen.Cubby) {
	for _, item := range items {
		sm.ItemCodeToCubby[item.Code] = append(sm.ItemCodeToCubby[item.Code], cubby)
	}
}

func (sm *stateManager) getCubbyIdByOrderId(orderId string, times int) string {
	cubbyId := ordertocubby.Map(orderId, uint32(times), 10)
	attemptsToAvoidCollision := 1
	for true {
		if _, ok := sm.CubbyIdToOrderId[cubbyId]; !ok {
			break
		}

		cubbyId = ordertocubby.Map(orderId, uint32(times+attemptsToAvoidCollision), 10)
		attemptsToAvoidCollision++
	}
	return cubbyId
}

func (sm *stateManager) GetPreparedOrders(orders []*gen.Order) []*gen.PreparedOrder {
	preparedOrders := []*gen.PreparedOrder{}

	for i, order := range orders {
		sm.OrderIdToItems[order.Id] = order.Items

		cubbyId := sm.getCubbyIdByOrderId(order.Id, i)
		sm.CubbyIdToOrderId[cubbyId] = order.Id

		cubby := &gen.Cubby{Id: cubbyId}
		sm.OrderIdToCubby[order.Id] = cubby

		preparedOrder := &gen.PreparedOrder{Order: order, Cubby: cubby}
		preparedOrders = append(preparedOrders, preparedOrder)

		sm.mapItemCodesToCubby(order.Items, cubby)
	}

	return preparedOrders
}

func (sm *stateManager) GetCubbyByItemCode(itemCode string) (*gen.Cubby, error) {
	if len(sm.ItemCodeToCubby[itemCode]) == 0 {
		return nil, fmt.Errorf("This item was distributed to all necessary cubbies")
	}

	cubby := sm.ItemCodeToCubby[itemCode][0]
	sm.ItemCodeToCubby[itemCode] = sm.ItemCodeToCubby[itemCode][1:]
	return cubby, nil
}

func (sm *stateManager) Clear() {
	sm.ItemCodeToCubby = map[string][]*gen.Cubby{}
	sm.CubbyIdToOrderId = map[string]string{}
	sm.OrderIdToCubby = map[string]*gen.Cubby{}
	sm.OrderIdToItems = map[string][]*gen.Item{}
	sm.OrderIdToState = map[string]gen.OrderState{}
}

func (sm *stateManager) GetOrderStateById(orderId string) (gen.OrderState, error) {
	orderState, ok := sm.OrderIdToState[orderId]
	if !ok {
		return gen.OrderState_FAILED, fmt.Errorf("No order with such ID")
	}

	return orderState, nil
}

func (sm *stateManager) SetOrdersState(orders []*gen.Order, state gen.OrderState) error {
	for _, order := range orders {
		sm.OrderIdToState[order.Id] = state
	}
	return nil
}

func (sm *stateManager) SetOrderStateById(orderId string, state gen.OrderState) error {
	sm.OrderIdToState[orderId] = state
	return nil
}

func (sm *stateManager) GetFulfillmentStatusByOrderId(orderId string) ([]*gen.FulfillmentStatus, error) {
	order := &gen.Order{Id: orderId, Items: sm.OrderIdToItems[orderId]}
	orderState, err := sm.GetOrderStateById(order.Id)
	if err != nil {
		return nil, err
	}

	fulfillmentStatus := &gen.FulfillmentStatus{Order: order, Cubby: sm.OrderIdToCubby[orderId], State: orderState}
	return []*gen.FulfillmentStatus{fulfillmentStatus}, nil
}

func (sm *stateManager) GetFulfillmentStatusOfAllOrders() ([]*gen.FulfillmentStatus, error) {
	fulfillmentStatusSlice := []*gen.FulfillmentStatus{}

	for orderId, _ := range sm.OrderIdToItems {
		fulfillmentStatus, err := sm.GetFulfillmentStatusByOrderId(orderId)
		if err != nil {
			return nil, err
		}

		fulfillmentStatusSlice = append(fulfillmentStatusSlice, fulfillmentStatus[0])
	}
	return fulfillmentStatusSlice, nil
}

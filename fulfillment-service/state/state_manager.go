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
	OrderIdToData    map[string]*orderData
	mu               sync.RWMutex
}

func New() State {
	return &stateManager{
		ItemCodeToCubby:  make(map[string][]*gen.Cubby),
		CubbyIdToOrderId: make(map[string]string),
		OrderIdToData:    make(map[string]*orderData),
		mu:               sync.RWMutex{},
	}
}

func (sm *stateManager) mapItemCodesToCubby(items []*gen.Item, cubby *gen.Cubby) {
	for _, item := range items {
		sm.ItemCodeToCubby[item.Code] = append(sm.ItemCodeToCubby[item.Code], cubby)
	}
}

func (sm *stateManager) doesOrderWithIdExist(orderId string) bool {
	_, ok := sm.OrderIdToData[orderId]
	return ok
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
		cubbyId := sm.getCubbyIdByOrderId(order.Id, i)
		sm.CubbyIdToOrderId[cubbyId] = order.Id

		cubby := &gen.Cubby{Id: cubbyId}
		sm.OrderIdToData[order.Id] = &orderData{id: order.Id, cubby: cubby, items: order.Items}

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
	sm.OrderIdToData = map[string]*orderData{}
}

func (sm *stateManager) GetOrderStateById(orderId string) (gen.OrderState, error) {
	if !sm.doesOrderWithIdExist(orderId) {
		return gen.OrderState_FAILED, fmt.Errorf("No order with such ID")
	}
	data := sm.OrderIdToData[orderId]
	return data.state, nil
}

func (sm *stateManager) SetOrdersState(orders []*gen.Order, state gen.OrderState) error {
	for _, order := range orders {
		err := sm.SetOrderStateById(order.Id, state)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sm *stateManager) SetOrderStateById(orderId string, state gen.OrderState) error {
	if !sm.doesOrderWithIdExist(orderId) {
		return fmt.Errorf("No order with such ID")
	}

	data := sm.OrderIdToData[orderId]
	data.state = state
	return nil
}

func (sm *stateManager) GetFulfillmentStatusByOrderId(orderId string) ([]*gen.FulfillmentStatus, error) {
	if !sm.doesOrderWithIdExist(orderId) {
		return nil, fmt.Errorf("No order with such ID")
	}

	data := sm.OrderIdToData[orderId]
	order := &gen.Order{Id: orderId, Items: data.items}
	fulfillmentStatus := &gen.FulfillmentStatus{Order: order, Cubby: data.cubby, State: data.state}
	return []*gen.FulfillmentStatus{fulfillmentStatus}, nil
}

func (sm *stateManager) GetFulfillmentStatusOfAllOrders() ([]*gen.FulfillmentStatus, error) {
	fulfillmentStatusSlice := []*gen.FulfillmentStatus{}
	for orderId, _ := range sm.OrderIdToData {
		fulfillmentStatus, err := sm.GetFulfillmentStatusByOrderId(orderId)
		if err != nil {
			return nil, err
		}

		fulfillmentStatusSlice = append(fulfillmentStatusSlice, fulfillmentStatus[0])
	}
	return fulfillmentStatusSlice, nil
}

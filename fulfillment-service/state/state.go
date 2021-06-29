package state

import (
	"fmt"
	"sync"

	"github.com/Emoto13/sort-system/gen"
	"github.com/preslavmihaylov/ordertocubby"
)

type State interface {
	AddOrders(orders []*gen.Order)

	GetCubbyByItemCode(itemCode string) (*gen.Cubby, error)
	GetOrderCubby(orderId string) (*gen.Cubby, error)
	GetOrderItems(orderId string) ([]*gen.Item, error)
	GetOrderStatus(orderId string) (gen.OrderStatus, error)
	SetOrderStatus(orderId string, status gen.OrderStatus) error

	GetFulfillmentStatusOfAllOrders() ([]*gen.FulfillmentStatus, error)

	Clear()
}

type state struct {
	itemCodeToCubby  map[string][]*gen.Cubby
	cubbyIdToOrderId map[string]string
	orderIdToData    map[string]*orderData
	mu               *sync.RWMutex
}

func New() State {
	return &state{
		itemCodeToCubby:  make(map[string][]*gen.Cubby),
		cubbyIdToOrderId: make(map[string]string),
		orderIdToData:    make(map[string]*orderData),
		mu:               &sync.RWMutex{},
	}
}

func (sm *state) mapItemCodesToCubby(items []*gen.Item, cubby *gen.Cubby) {
	for _, item := range items {
		sm.itemCodeToCubby[item.Code] = append(sm.itemCodeToCubby[item.Code], cubby)
	}
}

func (sm *state) doesOrderWithIdExist(orderId string) bool {
	_, ok := sm.orderIdToData[orderId]
	return ok
}

func (sm *state) getCubbyIdByOrderId(orderId string, times int) string {
	cubbyId := ordertocubby.Map(orderId, uint32(times), 10)
	attemptsToAvoidCollision := 1
	for true {
		if _, ok := sm.cubbyIdToOrderId[cubbyId]; !ok {
			break
		}

		cubbyId = ordertocubby.Map(orderId, uint32(times+attemptsToAvoidCollision), 10)
		attemptsToAvoidCollision++
	}
	return cubbyId
}

func (sm *state) AddOrders(orders []*gen.Order) {
	for i, order := range orders {
		cubbyId := sm.getCubbyIdByOrderId(order.Id, i)
		sm.cubbyIdToOrderId[cubbyId] = order.Id

		cubby := &gen.Cubby{Id: cubbyId}
		sm.orderIdToData[order.Id] = &orderData{id: order.Id, items: order.Items, cubby: cubby, status: gen.OrderStatus_PENDING}
		sm.mapItemCodesToCubby(order.Items, cubby)
	}
	fmt.Println(sm.orderIdToData)
}

func (sm *state) GetCubbyByItemCode(itemCode string) (*gen.Cubby, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if len(sm.itemCodeToCubby[itemCode]) == 0 {
		return nil, fmt.Errorf("Item: " + itemCode + " was distributed to all necessary cubbies")
	}

	cubby := sm.itemCodeToCubby[itemCode][0]
	sm.itemCodeToCubby[itemCode] = sm.itemCodeToCubby[itemCode][1:]
	return cubby, nil
}

func (sm *state) GetOrderCubby(orderId string) (*gen.Cubby, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.doesOrderWithIdExist(orderId) {
		fmt.Println("Cubby", orderId)

		return nil, fmt.Errorf("No order with such ID" + "Cubby " + orderId)

	}
	data := sm.orderIdToData[orderId]
	return data.cubby, nil

}

func (sm *state) GetOrderItems(orderId string) ([]*gen.Item, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.doesOrderWithIdExist(orderId) {
		return nil, fmt.Errorf("No order with such ID")
	}

	data := sm.orderIdToData[orderId]
	return data.items, nil
}

func (sm *state) GetOrderStatus(orderId string) (gen.OrderStatus, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.doesOrderWithIdExist(orderId) {
		return gen.OrderStatus_FAILED, fmt.Errorf("No order with such ID")
	}
	data := sm.orderIdToData[orderId]
	return data.status, nil
}

func (sm *state) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.itemCodeToCubby = map[string][]*gen.Cubby{}
	sm.cubbyIdToOrderId = map[string]string{}
	sm.orderIdToData = map[string]*orderData{}
}

func (sm *state) SetOrderStatus(orderId string, status gen.OrderStatus) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.doesOrderWithIdExist(orderId) {
		return fmt.Errorf("No order with such ID")
	}

	data := sm.orderIdToData[orderId]
	data.status = status
	return nil
}

func (sm *state) GetFulfillmentStatusByOrderId(orderId string) ([]*gen.FulfillmentStatus, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.doesOrderWithIdExist(orderId) {
		return nil, fmt.Errorf("No order with such ID")
	}

	data := sm.orderIdToData[orderId]
	order := &gen.Order{Id: orderId, Items: data.items}
	fulfillmentStatus := &gen.FulfillmentStatus{Order: order, Cubby: data.cubby, Status: data.status}
	return []*gen.FulfillmentStatus{fulfillmentStatus}, nil
}

func (sm *state) GetFulfillmentStatusOfAllOrders() ([]*gen.FulfillmentStatus, error) {
	fulfillmentStatusSlice := []*gen.FulfillmentStatus{}
	for _, orderData := range sm.orderIdToData {
		items, err := sm.GetOrderItems(orderData.id)
		if err != nil {
			return nil, err
		}

		cubby, err := sm.GetOrderCubby(orderData.id)
		if err != nil {
			return nil, err
		}

		orderStatus, err := sm.GetOrderStatus(orderData.id)
		if err != nil {
			return nil, err
		}

		order := &gen.Order{Id: orderData.id, Items: items}
		fulfillmentStatus := &gen.FulfillmentStatus{Order: order, Cubby: cubby, Status: orderStatus}
		fulfillmentStatusSlice = append(fulfillmentStatusSlice, fulfillmentStatus)
	}
	return fulfillmentStatusSlice, nil
}

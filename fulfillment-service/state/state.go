package state

import (
	"fmt"
	"sync"

	"github.com/Emoto13/sort-system/gen"
	"github.com/preslavmihaylov/ordertocubby"
)

type State interface {
	AddOrders(orders []*gen.Order)

	GetOrderCubbyByItemCode(itemCode string) (*OrderCubby, error)
	GetOrderDataById(orderId string) (OrderData, error)
	GetAllOrdersData() ([]OrderData, error)

	AddItemStatusForOrder(orderId string, itemStatus ItemStatus) error
	SetOrderStatus(orderId string, status gen.OrderStatus) error
	Clear()
}

type state struct {
	itemCodeToOrderCubby map[string][]*OrderCubby
	cubbyIdToOrderId     map[string]string
	orderIdToData        map[string]*OrderData
	mu                   sync.RWMutex
}

func New() State {
	return &state{
		itemCodeToOrderCubby: make(map[string][]*OrderCubby),
		cubbyIdToOrderId:     make(map[string]string),
		orderIdToData:        make(map[string]*OrderData),
		mu:                   sync.RWMutex{},
	}
}

func (sm *state) mapItemCodesToOrderCubby(items []*gen.Item, order *gen.Order, cubby *gen.Cubby) {
	for _, item := range items {
		sm.itemCodeToOrderCubby[item.Code] = append(sm.itemCodeToOrderCubby[item.Code], &OrderCubby{Order: order, Cubby: cubby})
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

func (sm *state) getOrderStatus(orderId string) (gen.OrderStatus, error) {
	if !sm.doesOrderWithIdExist(orderId) {
		return gen.OrderStatus_FAILED, fmt.Errorf("No order with such id: " + orderId)
	}

	data := sm.orderIdToData[orderId]
	if len(data.itemsFulfillmentStatus) == 0 {
		return gen.OrderStatus_PENDING, nil
	}

	for _, itemStatus := range data.itemsFulfillmentStatus {
		if itemStatus == Failed {
			return gen.OrderStatus_FAILED, nil
		} else if itemStatus == Pending {
			return gen.OrderStatus_PENDING, nil
		}
	}

	for _, itemStatus := range data.itemsFulfillmentStatus {
		if itemStatus == Pending {
			return gen.OrderStatus_PENDING, nil
		}
	}
	return gen.OrderStatus_READY, nil
}

func (sm *state) AddOrders(orders []*gen.Order) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for i, order := range orders {
		cubbyId := sm.getCubbyIdByOrderId(order.Id, i)
		sm.cubbyIdToOrderId[cubbyId] = order.Id

		cubby := &gen.Cubby{Id: cubbyId}
		sm.orderIdToData[order.Id] = &OrderData{Id: order.Id, Items: order.Items, Cubby: cubby, Status: gen.OrderStatus_PENDING}
		sm.mapItemCodesToOrderCubby(order.Items, order, cubby)
	}
}

func (sm *state) GetOrderCubbyByItemCode(itemCode string) (*OrderCubby, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if len(sm.itemCodeToOrderCubby[itemCode]) == 0 {
		return nil, fmt.Errorf("item: " + itemCode + " was distributed to all necessary cubbies")
	}

	orderCubby := sm.itemCodeToOrderCubby[itemCode][0]
	sm.itemCodeToOrderCubby[itemCode] = sm.itemCodeToOrderCubby[itemCode][1:]
	return orderCubby, nil
}

func (sm *state) GetOrderDataById(orderId string) (OrderData, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	if !sm.doesOrderWithIdExist(orderId) {
		return OrderData{}, fmt.Errorf("no order with such id: " + orderId)
	}

	status, err := sm.getOrderStatus(orderId)
	if err != nil {
		return OrderData{}, err
	}

	data := sm.orderIdToData[orderId]
	data.Status = status
	return *data, nil
}

func (sm *state) GetAllOrdersData() ([]OrderData, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	orderDataSlice := []OrderData{}
	for _, orderData := range sm.orderIdToData {
		data, err := sm.GetOrderDataById(orderData.Id)
		if err != nil {
			return nil, err
		}

		orderDataSlice = append(orderDataSlice, data)
	}

	return orderDataSlice, nil
}

func (sm *state) Clear() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.itemCodeToOrderCubby = map[string][]*OrderCubby{}
	sm.cubbyIdToOrderId = map[string]string{}
	sm.orderIdToData = map[string]*OrderData{}
}

func (sm *state) SetOrderStatus(orderId string, status gen.OrderStatus) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.doesOrderWithIdExist(orderId) {
		return fmt.Errorf("no order with such ID")
	}

	data := sm.orderIdToData[orderId]
	data.Status = status
	return nil
}

func (sm *state) AddItemStatusForOrder(orderId string, itemStatus ItemStatus) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if !sm.doesOrderWithIdExist(orderId) {
		return fmt.Errorf("no order with such ID")
	}

	data := sm.orderIdToData[orderId]
	data.itemsFulfillmentStatus = append(data.itemsFulfillmentStatus, itemStatus)
	return nil
}

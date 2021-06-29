package service

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/Emoto13/sort-system/fulfillment-service/state"
	"github.com/Emoto13/sort-system/gen"
)

type fulfillmentService struct {
	sortingRobot     gen.SortingRobotClient
	state            state.State
	orders           chan []*gen.Order
	processingOrders bool
	mu               sync.Mutex
}

func New(sortingRobot gen.SortingRobotClient) gen.FulfillmentServer {
	return &fulfillmentService{
		sortingRobot:     sortingRobot,
		state:            state.New(),
		orders:           make(chan []*gen.Order),
		processingOrders: false,
		mu:               sync.Mutex{},
	}
}

func (fs *fulfillmentService) areOrdersBeingProcessed() bool {
	return fs.processingOrders
}

func (fs *fulfillmentService) setAreOrdersBeingProcessed(value bool) {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fs.processingOrders = value
}

func (fs *fulfillmentService) LoadOrders(ctx context.Context, in *gen.LoadOrdersRequest) (*gen.CompleteResponse, error) {
	fs.orders <- in.Orders
	if fs.areOrdersBeingProcessed() {
		return &gen.CompleteResponse{Status: "Will start to process the request shortly", Orders: []*gen.PreparedOrder{}}, nil
	}

	return &gen.CompleteResponse{Status: "The request will be handled immediately", Orders: []*gen.PreparedOrder{}}, nil
}

func (fs *fulfillmentService) ProcessOrders(ctx context.Context, in *gen.Empty) (*gen.Empty, error) {
	for {
		orders := <-fs.orders

		if !fs.areOrdersBeingProcessed() {
			fs.setAreOrdersBeingProcessed(true)
			fs.StartProcessingOrder(ctx, orders)
		}

		fs.setAreOrdersBeingProcessed(false)
	}
	return &gen.Empty{}, nil
}

func (fs *fulfillmentService) StartProcessingOrder(ctx context.Context, orders []*gen.Order) ([]*gen.PreparedOrder, error) {
	fmt.Println("Start Processing Order")
	fs.state.Clear()
	fs.state.AddOrders(orders)

	preparedOrders, err := fs.GetPreparedOrders(orders)
	if err != nil {
		return nil, err
	}

	err = fs.fulfillOrders(ctx, orders)
	if err != nil {
		return nil, err
	}

	return preparedOrders, nil
}

func (fs *fulfillmentService) GetPreparedOrders(orders []*gen.Order) ([]*gen.PreparedOrder, error) {
	preparedOrders := []*gen.PreparedOrder{}
	for _, order := range orders {
		cubby, err := fs.state.GetOrderCubby(order.Id)
		if err != nil {
			return nil, err
		}

		preparedOrder := &gen.PreparedOrder{Order: order, Cubby: cubby}
		preparedOrders = append(preparedOrders, preparedOrder)
	}

	return preparedOrders, nil
}

func (fs *fulfillmentService) fulfillOrders(ctx context.Context, orders []*gen.Order) error {
	for _, order := range orders {
		for _, _ = range order.Items {
			resp, err := fs.sortingRobot.SelectItem(ctx, &gen.Empty{})
			if err != nil {
				return err
			}

			cubby, err := fs.state.GetCubbyByItemCode(resp.Item.Code)
			if err != nil {
				log.Println(err)
				continue
			}

			_, err = fs.sortingRobot.MoveItem(ctx, &gen.MoveItemRequest{Cubby: cubby})
			if err != nil {
				return err
			}

			fmt.Println("Item with code ", resp.Item.Code, " is moved to: ", cubby.Id)
		}
		fs.state.SetOrderStatus(order.Id, gen.OrderStatus_READY)
	}

	return nil
}

func (fs *fulfillmentService) GetOrderFulfillmentStatusById(ctx context.Context, in *gen.OrderIdRequest) (*gen.OrdersStatusResponse, error) {
	items, err := fs.state.GetOrderItems(in.OrderId)
	if err != nil {
		return nil, err
	}

	cubby, err := fs.state.GetOrderCubby(in.OrderId)
	if err != nil {
		return nil, err
	}

	orderStatus, err := fs.state.GetOrderStatus(in.OrderId)
	if err != nil {
		return nil, err
	}

	order := &gen.Order{Id: in.OrderId, Items: items}
	fulfillmentStatus := &gen.FulfillmentStatus{Order: order, Cubby: cubby, Status: orderStatus}
	return &gen.OrdersStatusResponse{FulfillmentStatus: []*gen.FulfillmentStatus{fulfillmentStatus}}, nil
}

func (fs *fulfillmentService) GetAllOrdersFulfillmentStatus(ctx context.Context, in *gen.Empty) (*gen.OrdersStatusResponse, error) {
	allOrdersFulfillmentStatus, err := fs.state.GetFulfillmentStatusOfAllOrders()
	if err != nil {
		return nil, err
	}

	return &gen.OrdersStatusResponse{FulfillmentStatus: allOrdersFulfillmentStatus}, nil
}

func (fs *fulfillmentService) MarkFulfilled(ctx context.Context, in *gen.OrderIdRequest) (*gen.Empty, error) {
	err := fs.state.SetOrderStatus(in.OrderId, gen.OrderStatus_READY)
	if err != nil {
		return nil, err
	}

	return &gen.Empty{}, nil
}

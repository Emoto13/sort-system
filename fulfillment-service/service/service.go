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

func New(params FulfillmentServiceParameters) gen.FulfillmentServer {
	return &fulfillmentService{
		sortingRobot:     params.SortingRobot,
		state:            params.State,
		orders:           params.Orders,
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
	go func() {
		fs.orders <- in.Orders
	}()

	if fs.areOrdersBeingProcessed() {
		return &gen.CompleteResponse{Status: "Will start to process the request shortly", Orders: []*gen.PreparedOrder{}}, nil
	}

	return &gen.CompleteResponse{Status: "The request will be handled immediately", Orders: []*gen.PreparedOrder{}}, nil
}

func (fs *fulfillmentService) ProcessOrders(ctx context.Context, in *gen.Empty) (*gen.Empty, error) {
	for {
		orders := <-fs.orders
		err := fs.processOrders(ctx, orders)
		if err != nil {
			return nil, err
		}
	}
	return &gen.Empty{}, nil
}

func (fs *fulfillmentService) processOrders(ctx context.Context, orders []*gen.Order) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	err := fs.StartProcessingOrder(ctx, orders)
	if err != nil {
		return err
	}

	return nil
}

func (fs *fulfillmentService) StartProcessingOrder(ctx context.Context, orders []*gen.Order) error {
	fmt.Println("Start Processing Order")
	fs.state.AddOrders(orders)

	err := fs.fulfillOrders(ctx, orders)
	if err != nil {
		return err
	}

	return nil
}

func (fs *fulfillmentService) GetPreparedOrders(orders []*gen.Order) ([]*gen.PreparedOrder, error) {
	preparedOrders := []*gen.PreparedOrder{}
	for _, order := range orders {
		orderData, err := fs.state.GetOrderDataById(order.Id)
		if err != nil {
			return nil, err
		}
		cubby := orderData.Cubby
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

			orderCubby, err := fs.state.GetOrderCubbyByItemCode(resp.Item.Code)
			if err != nil {
				log.Println(err)
				fs.state.AddItemStatusForOrder(order.Id, state.Failed)
				continue
			}

			_, err = fs.sortingRobot.MoveItem(ctx, &gen.MoveItemRequest{Cubby: orderCubby.Cubby})
			if err != nil {
				fs.state.AddItemStatusForOrder(order.Id, state.Failed)

				return err
			}
			fs.state.AddItemStatusForOrder(order.Id, state.Ready)
			fmt.Println("Item with code ", resp.Item.Code, " is moved to: ", orderCubby.Cubby.Id)
		}
	}
	fmt.Println(fs.state.GetAllOrdersData())
	return nil
}

func (fs *fulfillmentService) GetOrderFulfillmentStatusById(ctx context.Context, in *gen.OrderIdRequest) (*gen.OrdersStatusResponse, error) {
	fmt.Println("GetOrderFulfillmentStatusById")
	orderData, err := fs.state.GetOrderDataById(in.OrderId)
	if err != nil {
		return nil, err
	}

	order := &gen.Order{Id: in.OrderId, Items: orderData.Items}
	fulfillmentStatus := &gen.FulfillmentStatus{Order: order, Cubby: orderData.Cubby, Status: orderData.Status}
	return &gen.OrdersStatusResponse{FulfillmentStatus: []*gen.FulfillmentStatus{fulfillmentStatus}}, nil
}

func (fs *fulfillmentService) GetAllOrdersFulfillmentStatus(ctx context.Context, in *gen.Empty) (*gen.OrdersStatusResponse, error) {
	fmt.Println("GetAllOrdersFulfillmentStatus")
	orderDataSlice, err := fs.state.GetAllOrdersData()
	if err != nil {
		return nil, err
	}

	fulfillmentStatusSlice := []*gen.FulfillmentStatus{}
	for _, orderData := range orderDataSlice {
		order := &gen.Order{Id: orderData.Id, Items: orderData.Items}
		fulfillmentStatus := &gen.FulfillmentStatus{Order: order, Cubby: orderData.Cubby, Status: orderData.Status}
		fulfillmentStatusSlice = append(fulfillmentStatusSlice, fulfillmentStatus)
	}

	return &gen.OrdersStatusResponse{FulfillmentStatus: fulfillmentStatusSlice}, nil
}

func (fs *fulfillmentService) MarkFulfilled(ctx context.Context, in *gen.OrderIdRequest) (*gen.Empty, error) {
	err := fs.state.SetOrderStatus(in.OrderId, gen.OrderStatus_READY)
	if err != nil {
		return nil, err
	}

	return &gen.Empty{}, nil
}

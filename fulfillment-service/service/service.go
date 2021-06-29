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
	//orderProcessor     orderproc.OrderProcessor
	sortingRobot            gen.SortingRobotClient
	state                   state.State
	orders                  chan []*gen.Order
	areOrdersBeingProcessed bool
	mu                      *sync.Mutex
}

func NewFulfillmentService(sortingRobot gen.SortingRobotClient) gen.FulfillmentServer {
	return &fulfillmentService{
		//orderProcessor:     orderproc.New(sortingRobot),
		sortingRobot:            sortingRobot,
		state:                   state.New(),
		orders:                  make(chan []*gen.Order),
		areOrdersBeingProcessed: false,
		mu:                      &sync.Mutex{},
	}
}

func (fs *fulfillmentService) LoadOrders(ctx context.Context, in *gen.LoadOrdersRequest) (*gen.CompleteResponse, error) {
	fs.orders <- in.Orders
	if fs.areOrdersBeingProcessed {
		return &gen.CompleteResponse{Status: "Will start to process the request shortly", Orders: []*gen.PreparedOrder{}}, nil
	}

	return &gen.CompleteResponse{Status: "The request will be handled immediately", Orders: []*gen.PreparedOrder{}}, nil
}

func (fs *fulfillmentService) ProcessOrders(ctx context.Context, in *gen.Empty) (*gen.Empty, error) {
	for {
		orders := <-fs.orders
		fs.areOrdersBeingProcessed = true
		//fs.orderProcessor.ProcessOrders(ctx, orders)
		fs.StartProcessingOrder(ctx, orders)
		fs.areOrdersBeingProcessed = false
	}
	return &gen.Empty{}, nil
}

func (fs *fulfillmentService) StartProcessingOrder(ctx context.Context, orders []*gen.Order) []*gen.PreparedOrder {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	fmt.Println("Start Processing Order")
	fs.state.Clear()
	fs.state.SetOrdersState(orders, gen.OrderState_PENDING)

	preparedOrders := fs.state.GetPreparedOrders(orders)
	err := fs.fulfillOrders(ctx, orders)
	if err != nil {
		fmt.Println(err)
	}

	return preparedOrders
}

func (fs *fulfillmentService) fulfillOrders(ctx context.Context, preparedOrders []*gen.Order) error {
	for _, order := range preparedOrders {
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
		fs.state.SetOrderStateById(order.Id, gen.OrderState_READY)
	}

	return nil
}

func (fs *fulfillmentService) GetOrderStatusById(ctx context.Context, in *gen.OrderIdRequest) (*gen.OrdersStatusResponse, error) {
	fulfillmentStatus, err := fs.state.GetFulfillmentStatusByOrderId(in.OrderId)
	if err != nil {
		return nil, err
	}

	return &gen.OrdersStatusResponse{Status: fulfillmentStatus}, nil
}

func (fs *fulfillmentService) GetAllOrdersStatus(ctx context.Context, in *gen.Empty) (*gen.OrdersStatusResponse, error) {
	allOrdersFulfillmentStatus, err := fs.state.GetFulfillmentStatusOfAllOrders()
	if err != nil {
		return nil, err
	}

	return &gen.OrdersStatusResponse{Status: allOrdersFulfillmentStatus}, nil
}

func (fs *fulfillmentService) MarkFulfilled(ctx context.Context, in *gen.OrderIdRequest) (*gen.Empty, error) {
	err := fs.state.SetOrderStateById(in.OrderId, gen.OrderState_READY)
	if err != nil {
		return nil, err
	}

	return &gen.Empty{}, nil
}

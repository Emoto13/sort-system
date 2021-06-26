package main

import (
	"context"
	"fmt"
	"log"

	"github.com/Emoto13/sort-system/fulfillment-service/state"
	"github.com/Emoto13/sort-system/gen"
)

type fulfillmentService struct {
	sortingRobot      gen.SortingRobotClient
	state             state.State
	preparedOrders    chan []*gen.PreparedOrder
	areOrderProcessed bool
}

func newFulfillmentService(sortingRobot gen.SortingRobotClient) gen.FulfillmentServer {
	return &fulfillmentService{
		sortingRobot:      sortingRobot,
		state:             state.New(),
		preparedOrders:    make(chan []*gen.PreparedOrder),
		areOrderProcessed: false,
	}
}

func (fs *fulfillmentService) LoadOrders(ctx context.Context, in *gen.LoadOrdersRequest) (*gen.CompleteResponse, error) {
	fs.preparedOrders <- fs.StartProcessingOrder(ctx, in.Orders)
	return &gen.CompleteResponse{Status: "OK", Orders: []*gen.PreparedOrder{}}, nil
}

func (fs *fulfillmentService) ProcessOrders(ctx context.Context, in *gen.Empty) (*gen.Empty, error) {
	for {
		<-fs.preparedOrders
	}
	return &gen.Empty{}, nil
}

func (fs *fulfillmentService) StartProcessingOrder(ctx context.Context, orders []*gen.Order) []*gen.PreparedOrder {
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

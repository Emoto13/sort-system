package main

import (
	"context"
	"fmt"

	"github.com/Emoto13/sort-system/gen"
	"github.com/Emoto13/sort-system/fulfillment-service/state"
)

type fulfillmentService struct {
	sortingRobot gen.SortingRobotClient
	state *state.StateManager
}

func newFulfillmentService(sortingRobot gen.SortingRobotClient) gen.FulfillmentServer {
	return &fulfillmentService{
		sortingRobot: sortingRobot,
		state: state.NewStateManager(),
	}
}

func (fs *fulfillmentService) LoadOrders(ctx context.Context, in *gen.LoadOrdersRequest) (*gen.CompleteResponse, error) {
	preparedOrders := fs.state.GetPreparedOrders(in.Orders)
	err := fs.fullfillOrders(ctx)
	if err != nil {
		return nil, err
	}

	return &gen.CompleteResponse{Status: "200", Orders: preparedOrders}, nil
}

func (fs *fulfillmentService) fullfillOrders(ctx context.Context) error {

	for i := 0; i < fs.state.NumberOfItems; i++ {
		resp, err := fs.sortingRobot.SelectItem(ctx, &gen.Empty{})
		if err != nil {
			return err
		}

		cubby := fs.state.GetCubbyByItemCode(resp.Item.Code)
		_, err = fs.sortingRobot.MoveItem(ctx, &gen.MoveItemRequest{Cubby: cubby})
		if err != nil {
			return err
		}

		fmt.Println("Item with code ", resp.Item.Code, " is moved to: ", cubby.Id)
	}
	return nil
}

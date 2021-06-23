package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/Emoto13/sort_system/fulfillment-service/state"

	"github.com/Emoto13/sort_system/gen"
)

type fulfillmentService struct {
	sortingRobot gen.SortingRobotClient
	state        *StateManager
	mu           sync.Mutex
}

func newFulfillmentService(sortingRobot gen.SortingRobotClient) gen.FulfillmentServer {
	return &fulfillmentService{
		sortingRobot: sortingRobot,
		state:        newStateManager(),
	}
}

func (fs *fulfillmentService) LoadOrders(ctx context.Context, in *gen.LoadOrdersRequest) (*gen.CompleteResponse, error) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	preparedOrders := fs.state.getPreparedOrders(in.Orders)
	err := fs.fullfillOrders(ctx)
	if err != nil {
		return nil, err
	}

	return &gen.CompleteResponse{Status: "200", Orders: preparedOrders}, nil
}

func (fs *fulfillmentService) fullfillOrders(ctx context.Context) error {

	for i := 0; i < fs.state.numberOfItems; i++ {
		resp, err := fs.sortingRobot.SelectItem(ctx, &gen.SelectItemRequest{})
		if err != nil {
			return err
		}

		cubby := fs.state.getCubbyByItemCode(resp.Item.Code)
		_, err = fs.sortingRobot.MoveItem(ctx, &gen.MoveItemRequest{Cubby: cubby})
		if err != nil {
			return err
		}

		fmt.Println("Item with code ", resp.Item.Code, " is moved to: ", cubby.Id)
	}
	return nil
}

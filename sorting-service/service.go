package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/Emoto13/sort-system/gen"
)

type sortingService struct {
	Items        []*gen.Item
	SelectedItem *gen.Item
	m            sync.Mutex
}

func newSortingService() *sortingService {
	rand.Seed(time.Now().UnixNano())
	return &sortingService{
		m: sync.Mutex{},
	}
}

func (s *sortingService) LoadItems(ctx context.Context, in *gen.LoadItemsRequest) (*gen.Empty, error) {
	s.Items = append(s.Items, in.Items...)
	log.Println("Called LoadItems: ")
	log.Println(len(s.Items))
	return &gen.Empty{}, nil
}

func (s *sortingService) SelectItem(ctx context.Context, in *gen.Empty) (*gen.SelectItemResponse, error) {
	s.m.Lock()
	defer s.m.Unlock()

	log.Println("SelectedItem:", s.SelectedItem)

	if s.SelectedItem != nil {
		return nil, fmt.Errorf("item has already been selected")
	}

	if s.Items == nil || len(s.Items) == 0 {
		return nil, fmt.Errorf("no items in the cargo")
	}

	randomIndex := rand.Intn(len(s.Items))

	s.SelectedItem = s.Items[randomIndex]
	s.Items = append(s.Items[:randomIndex], s.Items[randomIndex+1:]...)

	return &gen.SelectItemResponse{Item: s.SelectedItem}, nil
}

func (s *sortingService) MoveItem(ctx context.Context, in *gen.MoveItemRequest) (*gen.Empty, error) {
	if s.SelectedItem == nil {
		return nil, fmt.Errorf("item is not selected")
	}

	s.SelectedItem = nil
	log.Println("Item moved. Items left: ", len(s.Items))
	return &gen.Empty{}, nil
}

func (s *sortingService) AuditState(ctx context.Context, in *gen.Empty) (*gen.AuditStateResponse, error) {
	return nil, fmt.Errorf("not implemented")
}

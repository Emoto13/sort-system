package main

import (
	"context"
	"testing"

	"github.com/Emoto13/sort-system/gen"
	"github.com/stretchr/testify/assert"
)

func TestLoadItems(t *testing.T) {
	items := []*gen.Item{
		&gen.Item{Code: "TestItem", Label: "TestItem"},
		&gen.Item{Code: "TestItem", Label: "TestItem"},
	}

	var tests = []struct {
		name     string
		items    []*gen.Item
		expected int
		message  string
	}{
		{"Test LoadItems When Called Once", items, 2, "There should be 2 items in the cargo"},
		{"Test LoadItems When Called More Than Once", items, 4, "There should be 4 items in the cargo"},
	}

	sorting_service := newSortingService()

	for _, test := range tests {
		sorting_service.LoadItems(context.Background(), &gen.LoadItemsRequest{Items: test.items})
		assert.Equal(t, len(sorting_service.Items), test.expected, test.message)
	}
}

func TestSelectItem(t *testing.T) {
	testItem := &gen.Item{Code: "TestItem", Label: "TestItem"}

	var tests = []struct {
		name    string
		item    *gen.Item
		message string
	}{
		{"Test SelectItems When Called Once", testItem, "There should be a selected item"},
	}

	sorting_service := newSortingService()
	items := []*gen.Item{testItem}
	sorting_service.LoadItems(context.Background(), &gen.LoadItemsRequest{Items: items})

	for _, _ = range tests {
		sorting_service.SelectItem(context.Background(), &gen.SelectItemRequest{})
		assert.Equal(t, sorting_service.SelectedItem, testItem, "There should be a selected item")
	}
}

func TestSelectItem_ErrorCases(t *testing.T) {
	sorting_service := newSortingService()
	testItem := &gen.Item{Code: "TestItem", Label: "TestItem"}
	items := []*gen.Item{testItem}

	sorting_service.LoadItems(context.Background(), &gen.LoadItemsRequest{Items: items})
	sorting_service.SelectItem(context.Background(), &gen.SelectItemRequest{})
	_, err := sorting_service.SelectItem(context.Background(), &gen.SelectItemRequest{})

	assert.NotEqual(t, err, nil, "When Item is selected, the method shoud return error")
}

func TestSelectItemWhenThereAreNoItemsLeft(t *testing.T) {
	sorting_service := newSortingService()
	sorting_service.SelectItem(context.Background(), &gen.SelectItemRequest{})
	_, err := sorting_service.SelectItem(context.Background(), &gen.SelectItemRequest{})
	assert.NotEqual(t, err, nil, "When there are no items in the cargo, the method shoud return error")
}

func TestMoveItem(t *testing.T) {
	sorting_service := newSortingService()
	testItem := &gen.Item{Code: "TestItem", Label: "TestItem"}
	items := []*gen.Item{testItem}

	sorting_service.LoadItems(context.Background(), &gen.LoadItemsRequest{Items: items})
	sorting_service.SelectItem(context.Background(), &gen.SelectItemRequest{})
	res, err := sorting_service.MoveItem(context.Background(), &gen.MoveItemRequest{})
	assert.NotEqual(t, res, nil, "Result should be empty MoveItemResponse")
	assert.Equal(t, err, nil, "There should be no error")
}

func TestMoveItemWhenNoItemIsSelected(t *testing.T) {
	sorting_service := newSortingService()
	sorting_service.SelectItem(context.Background(), &gen.SelectItemRequest{})
	_, err := sorting_service.SelectItem(context.Background(), &gen.SelectItemRequest{})
	assert.NotEqual(t, err, nil, "When there are no items in the cargo, the method shoud return error")
}

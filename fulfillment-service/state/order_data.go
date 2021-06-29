package state

import "github.com/Emoto13/sort-system/gen"

type orderData struct {
	id     string
	items  []*gen.Item
	cubby  *gen.Cubby
	status gen.OrderStatus
}

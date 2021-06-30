package state

import "github.com/Emoto13/sort-system/gen"

type OrderData struct {
	Id                     string
	Items                  []*gen.Item
	Cubby                  *gen.Cubby
	Status                 gen.OrderStatus
	itemsFulfillmentStatus []ItemStatus
}

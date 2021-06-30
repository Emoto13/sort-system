package state

type ItemStatus int

const (
	Ready ItemStatus = iota
	Pending
	Failed
)

package inventory

import (
	"time"
)

type Inventory struct {
	ID               int64
	ProductID        int64
	PhysicalQuantity int32
	ReservedQuantity int32
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (i *Inventory) AvailableQuantity() int32 {
	return i.PhysicalQuantity - i.ReservedQuantity
}

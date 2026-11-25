package inventory

import (
	"context"
	"fmt"
)

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{
		repository: repository,
	}
}

func (s *Service) GetInventory(ctx context.Context, productID int64) (*Inventory, error) {
	if productID <= 0 {
		return nil, ErrInvalidProductID
	}

	inventory, err := s.repository.GetByProductID(ctx, productID)
	if err != nil {
		return nil, fmt.Errorf("get inventory for product %d: %w", productID, err)
	}

	return inventory, nil
}

func (s *Service) AdjustPhysicalQuantity(
	ctx context.Context,
	productID int64,
	delta int32,
) error {
	if productID <= 0 {
		return ErrInvalidProductID
	}
	if delta == 0 {
		return ErrInvalidAdjustment
	}

	if err := s.repository.AdjustPhysicalQuantity(ctx, productID, delta); err != nil {
		return fmt.Errorf("adjust physical quantity for product %d by %d: %w", productID, delta, err)
	}

	return nil
}

func (s *Service) ReserveStock(
	ctx context.Context,
	productID int64,
	quantity int32,
) error {
	if productID <= 0 {
		return ErrInvalidProductID
	}
	if quantity <= 0 {
		return ErrInvalidReservation
	}

	if err := s.repository.Reserve(ctx, productID, quantity); err != nil {
		return fmt.Errorf("reserve %d units for product %d: %w", quantity, productID, err)
	}

	return nil
}

func (s *Service) ReleaseStock(
	ctx context.Context,
	productID int64,
	quantity int32,
) error {
	if productID <= 0 {
		return ErrInvalidProductID
	}
	if quantity <= 0 {
		return ErrInvalidReservation
	}

	if err := s.repository.Release(ctx, productID, quantity); err != nil {
		return fmt.Errorf("release %d units for product %d: %w", quantity, productID, err)
	}

	return nil
}

package mysql

import (
	"context"
	"order_service/model"

	"gorm.io/gorm"
)

func QueryOrder(ctx context.Context, orderId int64) (model.Order, error) {
	var data model.Order
	err := db.WithContext(ctx).Model(&model.Order{}).Where("order_id = ? ", orderId).First(&data).Error
	return data, err
}

func UpdateOrder(ctx context.Context, data model.Order) error {
	return db.WithContext(ctx).
		Model(&model.Order{}).
		Where("order_id = ?", data.OrderId).
		Updates(&data).Error
}

func CreateOrder(ctx context.Context, data *model.Order) error {
	return db.WithContext(ctx).Model(&model.Order{}).Save(data).Error
}

func CreateOrderDetail(ctx context.Context, data *model.OrderDetail) error {
	return db.WithContext(ctx).Model(&model.OrderDetail{}).Save(data).Error
}

func CreateOrderWithTransation(ctx context.Context, order *model.Order, orderDetail *model.OrderDetail) error {
	return db.WithContext(ctx).
		Transaction(func(tx *gorm.DB) error {
			if err := tx.Create(order).Error; err != nil {
				return err
			}

			if err := tx.Create(orderDetail).Error; err != nil {
				return err
			}

			return nil
		})
}

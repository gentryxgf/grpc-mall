package mysql

import (
	"context"
	"errors"
	"good_service/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GetGoodsByRoomId 根据roomId查询所有直播间的商品Id
func GetGoodsByRoomId(ctx context.Context, roomId int64) ([]*model.RoomGoods, error) {
	var data []*model.RoomGoods
	err := db.WithContext(ctx).
		Model(&model.RoomGoods{}).
		Where("room_id = ?", roomId).
		Order("weight").
		Find(&data).Error
	// 查询出错或则是空数据，返回错误
	if err != nil && err != gorm.ErrEmptySlice {
		return nil, errors.New("query mysql failed")
	}
	return data, nil
}

// GetGoodsById 根据id查询商品信息
func GetGoodsById(ctx context.Context, idList []int64) ([]*model.Goods, error) {
	var data []*model.Goods
	err := db.WithContext(ctx).
		Model(&model.Goods{}).
		Where("goods_id in ? ", idList).
		Clauses(clause.OrderBy{
			Expression: clause.Expr{SQL: "FIELD(goods_id, ?)", Vars: []interface{}{idList}, WithoutParentheses: true},
		}).
		Find(&data).Error
	if err != nil && err != gorm.ErrEmptySlice {
		return nil, errors.New("query mysql fail")
	}
	return data, nil
}

func GetGoodsDetail(ctx context.Context, goodsId int64) (*model.Goods, error) {
	var data model.Goods
	err := db.WithContext(ctx).
		Model(&model.Goods{}).
		Where("id = ?", goodsId).
		First(&data).
		Error

	if err != nil || err == gorm.ErrRecordNotFound {
		return nil, err
	}
	return &data, nil
}

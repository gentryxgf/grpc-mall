package model

import "time"

type BaseModel struct {
	ID       uint      `gorm:"primaryKey"`
	CreateAt time.Time `gorm:"autoCreateTime"`
	UpdateAt time.Time `gorm:"autoUpdateTime"`
	CreateBy string
	UpdateBy string
	Version  int16
	IsDel    int8 `gorm:"index"`
}

// OrderGoodsStockInfo 订单商品信息
type OrderGoodsStockInfo struct {
	OrderId int64
	GoodsId int64
	Num     int64
}

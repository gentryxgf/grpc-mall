package model

type RoomGoods struct {
	BaseMode

	RoomId    int64
	GoodsId   int64
	Weight    int64
	IsCurrent int8
}

func (RoomGoods) TableName() string {
	return "xx_room_goods"
}

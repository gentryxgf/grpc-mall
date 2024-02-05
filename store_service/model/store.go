package model

type Store struct {
	BaseModel

	GoodsId int64
	Num     int64
	Lock    int64
}

func (Store) TableName() string {
	return "xx_store"
}

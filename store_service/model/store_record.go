package model

type StoreRecord struct {
	BaseModel // 嵌入默认的7个字段

	OrderId int64
	GoodsId int64
	Num     int64
	Status  int32
}

// TableName 声明表名
func (StoreRecord) TableName() string {
	return "xx_store_record"
}

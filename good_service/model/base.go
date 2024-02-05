package model

import "time"

type BaseMode struct {
	ID       uint      `gorm:"primaryKey"`
	CreateAt time.Time `gorm:"autoCreateTime"`
	UpdateAt time.Time `gorm:"autoUpdateTime"`
	CreateBy string
	UpdateBy string
	Version  int16
	IsDel    int8 `gorm:"index"`
}

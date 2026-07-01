package models

type WebSite struct {
	Name string `gorm:"column:name;type:varchar(100);not null" json:"name"`
	Link string `gorm:"column:link;type:varchar(500);not null" json:"link"`
	Code int    `gorm:"column:code;type:smallint;not null;default:200" json:"code"`
}

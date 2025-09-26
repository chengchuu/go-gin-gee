package persistence

import (
	"errors"

	"github.com/jinzhu/gorm"
	"github.com/mazeyqian/go-gin-gee/internal/pkg/config"
	"github.com/mazeyqian/go-gin-gee/internal/pkg/db"
)

// Check
func checkDBDriver() (err error) {
	driver := config.GetConfig().Database.Driver
	if driver == "" {
		err = errors.New("unable to connect to the database")
		return
	}
	return
}

// Create
func Create(value interface{}) (err error) {
	if err = checkDBDriver(); err != nil {
		return
	}
	err = db.GetDB().Create(value).Error
	return
}

// Save
func Save(value interface{}) (err error) {
	if err = checkDBDriver(); err != nil {
		return
	}
	err = db.GetDB().Save(value).Error
	return
}

// Updates
func Updates(where interface{}, value interface{}) (err error) {
	if err = checkDBDriver(); err != nil {
		return
	}
	err = db.GetDB().Model(where).Updates(value).Error
	return
}

// Delete
func DeleteByModel(model interface{}) (count int64, err error) {
	if err = checkDBDriver(); err != nil {
		return
	}
	db := db.GetDB().Delete(model)
	err = db.Error
	if err != nil {
		return
	}
	count = db.RowsAffected
	return
}

// Delete
func DeleteByWhere(model, where interface{}) (count int64, err error) {
	if err = checkDBDriver(); err != nil {
		return
	}
	db := db.GetDB().Where(where).Delete(model)
	err = db.Error
	if err != nil {
		return
	}
	count = db.RowsAffected
	return
}

// Delete
func DeleteByID(model interface{}, id uint64) (count int64, err error) {
	if err = checkDBDriver(); err != nil {
		return
	}
	db := db.GetDB().Where("id=?", id).Delete(model)
	err = db.Error
	if err != nil {
		return
	}
	count = db.RowsAffected
	return
}

// Delete
func DeleteByIDS(model interface{}, ids []uint64) (count int64, err error) {
	if err = checkDBDriver(); err != nil {
		return
	}
	db := db.GetDB().Where("id in (?)", ids).Delete(model)
	err = db.Error
	if err != nil {
		return
	}
	count = db.RowsAffected
	return
}

// First
func FirstByID(out interface{}, id string) (notFound bool, err error) {
	if err = checkDBDriver(); err != nil {
		return
	}
	err = db.GetDB().First(out, id).Error
	if err != nil {
		notFound = gorm.IsRecordNotFoundError(err)
	}
	return
}

// First
func First(where interface{}, out interface{}, associations []string) (notFound bool, err error) {
	if err = checkDBDriver(); err != nil {
		return
	}
	db := db.GetDB()
	for _, a := range associations {
		db = db.Preload(a)
	}
	err = db.Where(where).First(out).Error
	if err != nil {
		notFound = gorm.IsRecordNotFoundError(err)
	}
	return
}

// Find
func Find(where interface{}, out interface{}, associations []string, orders ...string) (err error) {
	if err = checkDBDriver(); err != nil {
		return
	}
	db := db.GetDB()
	for _, a := range associations {
		db = db.Preload(a)
	}
	db = db.Where(where)
	if len(orders) > 0 {
		for _, order := range orders {
			db = db.Order(order)
		}
	}
	err = db.Find(out).Error
	return
}

// Scan
func Scan(model, where interface{}, out interface{}) (notFound bool, err error) {
	if err = checkDBDriver(); err != nil {
		return
	}
	err = db.GetDB().Model(model).Where(where).Scan(out).Error
	if err != nil {
		notFound = gorm.IsRecordNotFoundError(err)
	}
	return
}

// ScanList
func ScanList(model, where interface{}, out interface{}, orders ...string) (err error) {
	if err = checkDBDriver(); err != nil {
		return
	}
	db := db.GetDB().Model(model).Where(where)
	if len(orders) > 0 {
		for _, order := range orders {
			db = db.Order(order)
		}
	}
	err = db.Scan(out).Error
	return
}

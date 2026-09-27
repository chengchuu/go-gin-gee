package persistence

import (
	"errors"
	"hash/crc32"
	"sync"
	"time"

	"github.com/chengchuu/go-gin-gee/internal/pkg/config"
	"github.com/chengchuu/go-gin-gee/internal/pkg/db"
	"github.com/chengchuu/go-gin-gee/internal/pkg/models/kv"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrKVNotFound        = errors.New("key not found")
	ErrKVIncompatible    = errors.New("incompatible value type")
	ErrKVConcurrentWrite = errors.New("concurrent key-value write failed")
)

const kvWriteLockCount = 64

type KVRepository struct {
	writeLocks [kvWriteLockCount]sync.Mutex
}

var kvRepository = &KVRepository{}

func GetKVRepository() *KVRepository { return kvRepository }

func (repository *KVRepository) lockForKey(key string) *sync.Mutex {
	configuration := config.GetConfig()
	if configuration != nil && configuration.Database.Driver == "sqlite" {
		return &repository.writeLocks[0]
	}
	index := crc32.ChecksumIEEE([]byte(key)) % kvWriteLockCount
	return &repository.writeLocks[index]
}

func (repository *KVRepository) Get(key string) (*kv.Entry, error) {
	if err := checkDBDriver(); err != nil {
		return nil, err
	}

	var entry kv.Entry
	err := db.GetDB().Where(map[string]interface{}{"key": key}).First(&entry).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrKVNotFound
	}
	if err != nil {
		return nil, err
	}
	return &entry, nil
}

func (repository *KVRepository) Set(entry *kv.Entry) (bool, error) {
	if err := checkDBDriver(); err != nil {
		return false, err
	}
	keyLock := repository.lockForKey(entry.Key)
	keyLock.Lock()
	defer keyLock.Unlock()

	for attempt := 0; attempt < 2; attempt++ {
		created, retry, err := repository.set(entry)
		if !retry {
			return created, err
		}
	}
	return false, ErrKVConcurrentWrite
}

func (repository *KVRepository) set(entry *kv.Entry) (created bool, retry bool, err error) {
	tx := db.GetDB().Begin()
	if tx.Error != nil {
		return false, false, tx.Error
	}
	defer func() {
		if recoverValue := recover(); recoverValue != nil {
			tx.Rollback()
			panic(recoverValue)
		}
	}()

	var counter kv.Counter
	err = tx.Where(map[string]interface{}{"key": entry.Key}).First(&counter).Error
	if err == nil {
		tx.Rollback()
		return false, false, ErrKVIncompatible
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return false, false, err
	}

	var existing kv.Entry
	err = tx.Where(map[string]interface{}{"key": entry.Key}).First(&existing).Error
	created = errors.Is(err, gorm.ErrRecordNotFound)
	if err != nil && !created {
		tx.Rollback()
		return false, false, err
	}

	if created {
		if err = tx.Create(entry).Error; err != nil {
			tx.Rollback()
			if exists, lookupErr := kvKeyExists(db.GetDB(), entry.Key); lookupErr == nil && exists {
				// Another process created this key after our lookup.
				return false, true, nil
			}
			return false, false, err
		}
	} else {
		updates := map[string]interface{}{
			"value":        entry.Value,
			"content_type": entry.ContentType,
			"visibility":   entry.Visibility,
			"updated_at":   time.Now().UTC(),
		}
		if err = tx.Model(&existing).Updates(updates).Error; err != nil {
			tx.Rollback()
			return false, false, err
		}
		entry.Model = existing.Model
	}

	if err = tx.Commit().Error; err != nil {
		return false, false, err
	}
	return created, false, nil
}

func (repository *KVRepository) Increment(key string, delta int64) (int64, error) {
	if err := checkDBDriver(); err != nil {
		return 0, err
	}
	keyLock := repository.lockForKey(key)
	keyLock.Lock()
	defer keyLock.Unlock()

	return repository.increment(key, delta)
}

func (repository *KVRepository) increment(key string, delta int64) (value int64, err error) {
	tx := db.GetDB().Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}
	defer func() {
		if recoverValue := recover(); recoverValue != nil {
			tx.Rollback()
			panic(recoverValue)
		}
	}()

	var entry kv.Entry
	err = tx.Select("id").Where(map[string]interface{}{"key": key}).First(&entry).Error
	if err == nil {
		tx.Rollback()
		return 0, ErrKVIncompatible
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		tx.Rollback()
		return 0, err
	}

	// Upsert directly: updating a missing row before inserting takes MySQL gap
	// locks that can deadlock concurrent creation of otherwise unrelated keys.
	counter := kv.Counter{Key: key, Value: delta, Visibility: "private"}
	err = tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "key"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"value":      gorm.Expr("? + ?", clause.Column{Table: clause.CurrentTable, Name: "value"}, delta),
			"updated_at": time.Now().UTC(),
		}),
	}).Create(&counter).Error
	if err != nil {
		tx.Rollback()
		return 0, err
	}

	// Read by key rather than the insert result's engine-specific generated ID.
	counter = kv.Counter{}
	if err = tx.Where(map[string]interface{}{"key": key}).First(&counter).Error; err != nil {
		tx.Rollback()
		return 0, err
	}
	if err = tx.Commit().Error; err != nil {
		return 0, err
	}
	return counter.Value, nil
}

func kvKeyExists(database *gorm.DB, key string) (bool, error) {
	var count int64
	if err := database.Model(&kv.Entry{}).Where(map[string]interface{}{"key": key}).Count(&count).Error; err != nil {
		return false, err
	}
	if count > 0 {
		return true, nil
	}
	if err := database.Model(&kv.Counter{}).Where(map[string]interface{}{"key": key}).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

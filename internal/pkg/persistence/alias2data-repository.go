package persistence

import (
	"errors"
	"log"
	"strconv"

	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/alias2data"
)

type Alias2dataRepository struct{}

var alias2dataRepository *Alias2dataRepository

func GetAlias2dataRepository() *Alias2dataRepository {
	if alias2dataRepository == nil {
		alias2dataRepository = &Alias2dataRepository{}
	}
	return alias2dataRepository
}

func (r *Alias2dataRepository) Get(alias string) (*models.Alias2data, error) {
	if alias == "" {
		return nil, errors.New("alias is required")
	}
	var alias2data models.Alias2data
	where := models.Alias2data{}
	where.Alias = alias
	notFound, err := First(&where, &alias2data, []string{})
	log.Println("Get notFound", notFound)
	if err != nil {
		log.Println("Get err", err)
		return nil, err
	}
	return &alias2data, err
}

func (r *Alias2dataRepository) Add(alias2data *models.Alias2data) error {
	data, _ := r.Get(alias2data.Alias)
	if data != nil {
		log.Println("Exist", data)
		return errors.New("data exist")
	}
	err := Create(&alias2data)
	if err != nil {
		return err
	}
	err = Save(&alias2data)
	return err
}

func (r *Alias2dataRepository) CountByAlias(alias string) (int, error) {
	var err error
	if alias == "" {
		return 0, errors.New("alias is required")
	}
	alias2data := models.Alias2data{Alias: alias}
	notFound, _ := First(&alias2data, &alias2data, []string{})
	if notFound {
		alias2data.Data = "0"
		err = Create(&alias2data)
		if err != nil {
			return 0, err
		}
	}
	count, err := strconv.Atoi(alias2data.Data)
	if err != nil {
		return 0, err
	}
	count++
	alias2data.Data = strconv.Itoa(count)
	err = Save(&alias2data)
	if err != nil {
		return 0, err
	}
	return count, nil
}

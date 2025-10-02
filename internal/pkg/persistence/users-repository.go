package persistence

import (
	"strconv"

	"github.com/chengchuu/go-gin-gee/internal/pkg/db"
	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/users"
)

type UserRepository struct{}

var userRepository *UserRepository

func GetUserRepository() *UserRepository {
	if userRepository == nil {
		userRepository = &UserRepository{}
	}
	return userRepository
}

func (r *UserRepository) Get(id string) (*models.User, error) {
	var user models.User
	where := models.User{}
	where.ID, _ = strconv.ParseUint(id, 10, 64)
	_, err := First(&where, &user, []string{"Role"})
	if err != nil {
		return nil, err
	}
	return &user, err
}

func (r *UserRepository) GetByUsername(username string) (*models.User, error) {
	var user models.User
	where := models.User{}
	where.Username = username
	_, err := First(&where, &user, []string{"Role"})
	if err != nil {
		return nil, err
	}
	return &user, err
}

func (r *UserRepository) All() (*[]models.User, error) {
	var users []models.User
	err := Find(&models.User{}, &users, []string{"Role"}, "id asc")
	return &users, err
}

func (r *UserRepository) Query(q *models.User) (*[]models.User, error) {
	var users []models.User
	err := Find(&q, &users, []string{"Role"}, "id asc")
	return &users, err
}

func (r *UserRepository) Add(user *models.User) error {
	var err error
	err = Create(&user)
	if err != nil {
		return err
	}
	err = Save(&user)
	return err
}

func (r *UserRepository) Update(user *models.User) error {
	var userRole models.UserRole
	_, err := First(models.UserRole{UserID: user.ID}, &userRole, []string{})
	if err != nil {
		return err
	}
	userRole.RoleName = user.Role.RoleName
	err = Save(&userRole)
	if err != nil {
		return err
	}
	err = db.GetDB().Omit("Role").Save(&user).Error
	user.Role = userRole
	return err
}

func (r *UserRepository) Delete(user *models.User) error {
	err := db.GetDB().Unscoped().Delete(models.UserRole{UserID: user.ID}).Error
	if err != nil {
		return err
	}
	err = db.GetDB().Unscoped().Delete(&user).Error
	return err
}

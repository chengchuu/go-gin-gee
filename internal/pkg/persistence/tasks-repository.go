package persistence

import (
	"strconv"

	"github.com/chengchuu/go-gin-gee/internal/pkg/db"
	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/tasks"
)

type TaskRepository struct{}

var taskRepository *TaskRepository

func GetTaskRepository() *TaskRepository {
	if taskRepository == nil {
		taskRepository = &TaskRepository{}
	}
	return taskRepository
}

func (r *TaskRepository) Get(id string) (*models.Task, error) {
	var task models.Task
	where := models.Task{}
	where.ID, _ = strconv.ParseUint(id, 10, 64)
	_, err := First(&where, &task, []string{"User"})
	if err != nil {
		return nil, err
	}
	return &task, err
}

func (r *TaskRepository) All() (*[]models.Task, error) {
	var tasks []models.Task
	err := Find(&models.Task{}, &tasks, []string{"User"}, "id asc")
	return &tasks, err
}

func (r *TaskRepository) Query(q *models.Task) (*[]models.Task, error) {
	var tasks []models.Task
	err := Find(&q, &tasks, []string{"User"}, "id asc")
	return &tasks, err
}

func (r *TaskRepository) Add(task *models.Task) error {
	var err error
	err = Create(&task)
	if err != nil {
		return err
	}
	err = Save(&task)
	return err
}

func (r *TaskRepository) Update(task *models.Task) error {
	return db.GetDB().Omit("User").Save(&task).Error
}

func (r *TaskRepository) Delete(task *models.Task) error {
	return db.GetDB().Unscoped().Delete(&task).Error
}

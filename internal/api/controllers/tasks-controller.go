package controllers

import (
	"errors"
	"log"
	"net/http"

	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/tasks"
	"github.com/chengchuu/go-gin-gee/internal/pkg/persistence"
	http_err "github.com/chengchuu/go-gin-gee/pkg/http-err"
	"github.com/gin-gonic/gin"
)

// GetTaskById godoc
// @Summary Retrieves task based on given ID
// @Description get Task by ID
// @Produce json
// @Param id path integer true "Task ID"
// @Success 200 {object} tasks.Task
// @Router /api/tasks/{id} [get]
// @Security Authorization Token
func GetTaskById(c *gin.Context) {
	per := persistence.GetTaskRepository()
	id := c.Param("id")
	if task, err := per.Get(id); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("task not found"))
		log.Println(err)
	} else {
		c.JSON(http.StatusOK, task)
	}
}

// GetTasks godoc
// @Summary Retrieves tasks based on query
// @Description Get Tasks
// @Produce json
// @Param taskname query string false "Taskname"
// @Param firstname query string false "Firstname"
// @Param lastname query string false "Lastname"
// @Success 200 {array} []tasks.Task
// @Router /api/tasks [get]
// @Security Authorization Token
func GetTasks(c *gin.Context) {
	per := persistence.GetTaskRepository()
	var q models.Task
	_ = c.Bind(&q)
	if tasks, err := per.Query(&q); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("tasks not found"))
		log.Println(err)
	} else {
		c.JSON(http.StatusOK, tasks)
	}
}

func CreateTask(c *gin.Context) {
	per := persistence.GetTaskRepository()
	var taskInput models.Task
	_ = c.BindJSON(&taskInput)
	if err := per.Add(&taskInput); err != nil {
		http_err.NewError(c, http.StatusBadRequest, err)
		log.Println(err)
	} else {
		c.JSON(http.StatusCreated, taskInput)
	}
}

func UpdateTask(c *gin.Context) {
	per := persistence.GetTaskRepository()
	id := c.Params.ByName("id")
	var taskInput models.Task
	_ = c.BindJSON(&taskInput)
	if _, err := per.Get(id); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("task not found"))
		log.Println(err)
	} else {
		if err := per.Update(&taskInput); err != nil {
			http_err.NewError(c, http.StatusNotFound, err)
			log.Println(err)
		} else {
			c.JSON(http.StatusOK, taskInput)
		}
	}
}

func DeleteTask(c *gin.Context) {
	per := persistence.GetTaskRepository()
	id := c.Params.ByName("id")
	var taskInput models.Task
	_ = c.BindJSON(&taskInput)
	if task, err := per.Get(id); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("task not found"))
		log.Println(err)
	} else {
		if err := per.Delete(task); err != nil {
			http_err.NewError(c, http.StatusNotFound, err)
			log.Println(err)
		} else {
			c.JSON(http.StatusNoContent, "")
		}
	}
}

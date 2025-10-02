package controllers

import (
	"errors"
	"log"
	"net/http"

	models "github.com/chengchuu/go-gin-gee/internal/pkg/models/users"
	"github.com/chengchuu/go-gin-gee/internal/pkg/persistence"
	"github.com/chengchuu/go-gin-gee/pkg/crypto"
	http_err "github.com/chengchuu/go-gin-gee/pkg/http-err"
	"github.com/gin-gonic/gin"
)

type UserInput struct {
	Username  string `json:"username" binding:"required"`
	Lastname  string `json:"lastname"`
	Firstname string `json:"firstname"`
	Password  string `json:"password" binding:"required"`
	Role      string `json:"role"`
}

// GetUserById godoc
// @Summary Retrieves user based on given ID
// @Description get User by ID
// @Produce json
// @Param id path integer true "User ID"
// @Success 200 {object} users.User
// @Router /api/users/{id} [get]
// @Security Authorization Token
func GetUserById(c *gin.Context) {
	per := persistence.GetUserRepository()
	id := c.Param("id")
	if user, err := per.Get(id); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("user not found"))
		log.Println(err)
	} else {
		c.JSON(http.StatusOK, user)
	}
}

// GetUsers godoc
// @Summary Retrieves users based on query
// @Description Get Users
// @Produce json
// @Param username query string false "Username"
// @Param firstname query string false "Firstname"
// @Param lastname query string false "Lastname"
// @Success 200 {array} []users.User
// @Router /api/users [get]
// @Security Authorization Token
func GetUsers(c *gin.Context) {
	per := persistence.GetUserRepository()
	var q models.User
	_ = c.Bind(&q)
	if users, err := per.Query(&q); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("users not found"))
		log.Println(err)
	} else {
		c.JSON(http.StatusOK, users)
	}
}

func CreateUser(c *gin.Context) {
	per := persistence.GetUserRepository()
	var userInput UserInput
	_ = c.BindJSON(&userInput)
	user := models.User{
		Username:  userInput.Username,
		Firstname: userInput.Firstname,
		Lastname:  userInput.Lastname,
		Hash:      crypto.HashAndSalt([]byte(userInput.Password)),
		Role:      models.UserRole{RoleName: userInput.Role},
	}
	if err := per.Add(&user); err != nil {
		http_err.NewError(c, http.StatusBadRequest, err)
		log.Println(err)
	} else {
		c.JSON(http.StatusCreated, user)
	}
}

func UpdateUser(c *gin.Context) {
	per := persistence.GetUserRepository()
	id := c.Params.ByName("id")
	var userInput UserInput
	_ = c.BindJSON(&userInput)
	if user, err := per.Get(id); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("user not found"))
		log.Println(err)
	} else {
		user.Username = userInput.Username
		user.Lastname = userInput.Lastname
		user.Firstname = userInput.Firstname
		user.Hash = crypto.HashAndSalt([]byte(userInput.Password))
		user.Role = models.UserRole{RoleName: userInput.Role}
		if err := per.Update(user); err != nil {
			http_err.NewError(c, http.StatusNotFound, err)
			log.Println(err)
		} else {
			c.JSON(http.StatusOK, user)
		}
	}
}

func DeleteUser(c *gin.Context) {
	per := persistence.GetUserRepository()
	id := c.Params.ByName("id")
	var userInput UserInput
	_ = c.BindJSON(&userInput)
	if user, err := per.Get(id); err != nil {
		http_err.NewError(c, http.StatusNotFound, errors.New("user not found"))
		log.Println(err)
	} else {
		if err := per.Delete(user); err != nil {
			http_err.NewError(c, http.StatusNotFound, err)
			log.Println(err)
		} else {
			c.JSON(http.StatusNoContent, "")
		}
	}
}

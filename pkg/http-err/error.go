package http_err

import "github.com/gin-gonic/gin"

const (
	CodeSuccess               = 0
	CodeInvalidRequest        = 40001
	CodeInvalidKey            = 40002
	CodeInvalidValue          = 40003
	CodeAPIKeyRequired        = 40101
	CodeAccessDenied          = 40301
	CodeKeyNotFound           = 40401
	CodeIncompatibleValueType = 40901
	CodeInternalServerError   = 50001
)

// APIResponse is the response envelope for new JSON APIs.
type APIResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data" swaggertype:"object"`
}

// Success writes a successful API response.
func Success(c *gin.Context, status int, data interface{}) {
	c.JSON(status, APIResponse{Code: CodeSuccess, Message: "success", Data: data})
}

// Failure writes an API failure without exposing internal error details.
func Failure(c *gin.Context, status int, code int, message string) {
	c.JSON(status, APIResponse{Code: code, Message: message, Data: nil})
}

// NewError example
func NewError(c *gin.Context, status int, err error) {
	er := HTTPError{
		Code:    status,
		Message: err.Error(),
	}
	c.JSON(status, er)
}

// HTTPError example
type HTTPError struct {
	Code    int    `json:"code" example:"400"`
	Message string `json:"message" example:"status bad request"`
}

package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type HTTPResponse[T any] struct {
	Code int `json:"code,omitempty"`
	Msg string `json:"msg,omitempty"`
	Data T `json:"data,omitempty"`
}

func RespSuccess[T any](c *gin.Context, data T) {
	resp := HTTPResponse[T]{
		Code: http.StatusOK,
		Data: data,
	}
	c.JSON(http.StatusOK, resp)
}

func RespSuccessNone(c *gin.Context) {
	resp := HTTPResponse[int]{
		Code: http.StatusOK,
		Data: 0,
	}
	c.JSON(http.StatusOK, resp)
}

func RespError(c *gin.Context, code int, msg string) {
	resp := HTTPResponse[int]{
		Code: code,
		Msg:  msg,
		Data: 0,
	}
	c.JSON(http.StatusOK, resp)
}

package response

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// 错误码
const (
	CodeSuccess          = 0
	CodeParamInvalid     = 10001
	CodeUnauthorized     = 10002
	CodeForbidden        = 10003
	CodeKeyExists        = 20001
	CodeKeyNotFound      = 20002
	CodeRevisionNotFound = 20003
	CodeEnvExists        = 20004
	CodeEnvHasConfigs    = 20005
	CodeEtcdConnFailed   = 30001
	CodeEtcdOpFailed     = 30002
	CodeUserExists       = 40001
	CodeAuthFailed       = 40002
	CodeImportFormat     = 50001
	CodeImportPartial    = 50002
	CodeInternalError    = 99999
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

type PageData struct {
	List     interface{} `json:"list"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{Code: CodeSuccess, Message: "ok", Data: data})
}

func OKPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: "ok",
		Data:    PageData{List: list, Total: total, Page: page, PageSize: pageSize},
	})
}

func Fail(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{Code: code, Message: message, Data: nil})
}

func FailUnauthorized(c *gin.Context, message string) {
	c.JSON(http.StatusUnauthorized, Response{Code: CodeUnauthorized, Message: message, Data: nil})
}

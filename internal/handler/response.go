package handler

import (
	"github.com/gin-gonic/gin"

	"github.com/dysodeng/etcd-manager/internal/response"
)

// Re-export constants from response package for handler convenience
const (
	CodeSuccess          = response.CodeSuccess
	CodeParamInvalid     = response.CodeParamInvalid
	CodeUnauthorized     = response.CodeUnauthorized
	CodeForbidden        = response.CodeForbidden
	CodeKeyExists        = response.CodeKeyExists
	CodeKeyNotFound      = response.CodeKeyNotFound
	CodeRevisionNotFound = response.CodeRevisionNotFound
	CodeEnvExists        = response.CodeEnvExists
	CodeEnvHasConfigs    = response.CodeEnvHasConfigs
	CodeEtcdConnFailed   = response.CodeEtcdConnFailed
	CodeEtcdOpFailed     = response.CodeEtcdOpFailed
	CodeUserExists       = response.CodeUserExists
	CodeAuthFailed       = response.CodeAuthFailed
	CodeImportFormat     = response.CodeImportFormat
	CodeImportPartial    = response.CodeImportPartial
	CodeInternalError    = response.CodeInternalError
)

// Response is re-exported for use in handlers
type Response = response.Response

func OK(c *gin.Context, data interface{}) {
	response.OK(c, data)
}

func OKPage(c *gin.Context, list interface{}, total int64, page, pageSize int) {
	response.OKPage(c, list, total, page, pageSize)
}

func Fail(c *gin.Context, code int, message string) {
	response.Fail(c, code, message)
}

func FailUnauthorized(c *gin.Context, message string) {
	response.FailUnauthorized(c, message)
}

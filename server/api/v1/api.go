package v1

import (
	"github.com/gin-gonic/gin"
	"github.com/jaronnie/deploy-dagger/server/middlewares"
	"github.com/jaronnie/deploy-dagger/server/service"
)

func ApiRouter(rg *gin.RouterGroup) {
	rg.GET("/health", func(ctx *gin.Context) {
		ctx.String(200, "success")
	})

	rg.GET("/deploy", middlewares.HeadersMiddleware(), service.Deploy)
	rg.POST("/dingding/send", service.Send)
}

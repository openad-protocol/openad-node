package restful

import (
	"github.com/gin-gonic/gin"
	"github.com/ontio/layer2deploy/middleware/cors"
	"github.com/ontio/layer2deploy/restful/api"
)

func NewRouter() *gin.Engine {
	gin.DisableConsoleColor()
	root := gin.Default()
	root.Use(cors.Cors())
	api.RoutesApi(root)
	return root
}

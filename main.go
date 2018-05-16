package main

import (
	"github.com/gin-gonic/gin"
	"github.com/henly2/go-swagger-doc"
	"strings"
	"net/http"
	"fmt"
	"api_router/base/data"
	"bastionpay_api/apigroup"
)

func main()  {
	engine := gin.Default()
	startSwagger(engine)
	engine.Run(":8040")
}

func DocLoader(key string) ([]byte, error){
	fmt.Println("key:", key)
	return []byte("what"), nil
}

func startSwagger(engine *gin.Engine)  {
	config := swagger.Config{}
	config.Title = "BastionPay Api"
	config.Description  = "BastionPay Api"
	config.DocVersion = "1.0"
	//config.SwaggerUrlPrefix = docRelativePath
	//config.SwaggerUiUrl = "http://127.0.0.1:8030"
	swagger.InitializeApiRoutes(engine, &config, DocLoader)

	initDoc := func(userLevel int){
		apiGroupName := ""
		if userLevel == data.APILevel_client{
			apiGroupName = "api"
		} else if userLevel == data.APILevel_admin {
			apiGroupName = "user"
		} else if userLevel == data.APILevel_genesis {
			apiGroupName = "admin"
		}

		router := engine.Group("/"+apiGroupName, func(ctx *gin.Context) {

		})
		router.Use(func(ctx *gin.Context) {
			origin := ctx.Request.Header.Get("origin")
			ctx.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			ctx.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			ctx.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, XMLHttpRequest, " +
				"Accept-Encoding, X-CSRF-Token, Authorization")
			if ctx.Request.Method == "OPTIONS" {
				ctx.String(200, "ok")
				return
			}
			ctx.Next()
		})

		apiAll := apigroup.ListApiGroup()
		for _, apiGroup := range apiAll {
			for _, apiProxy := range apiGroup{
				if apiProxy.ApiDocInfo.Level > userLevel {
					continue
				}
				router.POST(apiProxy.ApiDocInfo.Path, func(ctx *gin.Context) {
					path := ctx.Request.URL.Path
					paths := strings.Split(path, "/")
					if len(paths) != 4{
						ctx.JSON(http.StatusOK, swagger.SuccessResp{Result:1, Message:"path corrupt"})
						return
					}
					ver := paths[1]
					srv := paths[2]
					function := paths[3]

					_, err := apigroup.FindApiBySrvFunction(ver, srv, function)
					if err != nil {
						ctx.JSON(http.StatusOK, swagger.SuccessResp{Result:1, Message:"not find function"})
						return
					}

					// TODO: run test
					ctx.JSON(http.StatusOK, swagger.SuccessResp{Result:0, Message:"in developing"})
				})

				swagger.Swagger3(router, apiGroupName, apiProxy.ApiDocInfo.Path,"post", &swagger.StructParam{
					JsonData: apiProxy.ApiDocInfo.Input,
					ResponseData: apiProxy.ApiDocInfo.Output,
					Tags:[]string{apiProxy.ApiDocInfo.Comment},
					Summary:apiProxy.ApiDocInfo.Comment,
				})
			}
		}
	}

	initDoc(data.APILevel_client)
	initDoc(data.APILevel_admin)
	initDoc(data.APILevel_genesis)
}
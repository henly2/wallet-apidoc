package main

import (
	"github.com/gin-gonic/gin"
	"github.com/henly2/go-swagger-doc"
	"strings"
	"net/http"
	"fmt"
	"bastionpay_api/apigroup"
	"bastionpay_api/gateway"
	"flag"
	"io/ioutil"
	"gopkg.in/yaml.v2"
	"os"
	"github.com/gin-contrib/cors"
	"bastionpay_api/apibackend"
	"bastionpay_api/apidoc"
	"sort"
)

var confFile = flag.String("c", "config.yml", "conf file.")

type Config struct{
	Server struct {
		Port string `yaml:"port"`
	}
	Wallet struct {
		Dir    		string `yaml:"dir"`
		CfgName    	string `yaml:"cfgName"`
	}
}
func main()  {
	flag.Parse()

	conf := new(Config)
	data, err := ioutil.ReadFile(*confFile)
	if err != nil {
		fmt.Println("read yml config file error:", err.Error())
		os.Exit(1)
	}

	err = yaml.Unmarshal([]byte(data), conf)
	if err != nil {
		fmt.Println("Unmarshal yml config file error:", err.Error())
		os.Exit(1)
	}

	err = gateway.Init(conf.Wallet.Dir, conf.Wallet.CfgName)
	if err != nil {
		fmt.Println("gateway.Init error:", err.Error())
	}

	engine := gin.Default()
	engine.Use(cors.New(cors.Config{
		AllowAllOrigins:true,
		AllowMethods:     []string{"POST", "GET", "OPTIONS", "PUT", "DELETE"},
		AllowHeaders:     []string{"Authorization", "X-Requested-With", "X_Requested_With", "Content-Type", "Access-Token", "Accept-Language"},
		//AllowOrigins:     []string{"*"},
		//AllowCredentials: true,
		//AllowOriginFunc: func(origin string) bool {
		//	return true;//origin == "https://github.com"
		//},
		//MaxAge: 12 * time.Hour,
	}))
	//engine.Static("/api", "swagger-dist")
	engine.Static("/documents", "documents")

	startSwagger(engine)
	engine.Run(":" + conf.Server.Port)
}

func isInGroup(group []string, name string) bool {
	for _, v := range group {
		if v == name {
			return true
		}
	}

	return false
}

func docLoader(key string) ([]byte, error){
	fmt.Println("key:", key)
	return []byte("what"), nil
}

func startSwagger(engine *gin.Engine)  {
	swagger.InitializeApiRoutesByGroup(engine, "apidoc")

	initDoc := func(apiGroupName string){
		config := swagger.Config{}
		config.Title = "BastionPay " + apiGroupName
		apiGroupInfo, err := apigroup.GetApiGroupInfo(apiGroupName)
		if err != nil {
			config.Description = "BastionPay " + apiGroupName
		} else {
			config.Description = apiGroupInfo.Description
		}
		config.Description += " Refer <a href='#tag/Api-uniform-data-layer'><b>Api uniform data layer</b></a>"

		config.DocVersion = "1.0"
		swagger.AddGroupOption(apiGroupName, &config, docLoader)

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
		for _, apiGroupAll := range apiAll {
			for _, apiProxy := range apiGroupAll{
				if !isInGroup(apiProxy.ApiDocInfo.Group, apiGroupName) {
					continue
				}
				router.POST(apiProxy.ApiDocInfo.Path(), func(ctx *gin.Context) {
					path := ctx.Request.URL.Path
					path = strings.TrimRight(path, "/")
					path = strings.TrimLeft(path, "/")

					paths := strings.Split(path, "/")
					if len(paths) != 4{
						ctx.JSON(http.StatusOK, swagger.SuccessResp{Result:1, Message:"path corrupt"})
						return
					}
					groupPath := paths[0]
					ver := paths[1]
					srv := paths[2]
					function := paths[3]

					apiProxy, err := apigroup.FindApiBySrvFunction(ver, srv, function)
					if err != nil {
						ctx.JSON(http.StatusOK, swagger.SuccessResp{Result:1, Message:"not find function"})
						return
					}

					ctx.ShouldBindJSON(apiProxy.ApiDocInfo.Input)

					apiErr := gateway.RunApi("/" + groupPath + apiProxy.ApiDocInfo.Path(), apiProxy.ApiDocInfo.Input, apiProxy.ApiDocInfo.Output)
					if apiErr != nil {
						fmt.Println("api err: ", apiErr)
						ctx.JSON(http.StatusOK, swagger.SuccessResp{Result:1, Message:apiErr.ErrMsg})
						return
					}

					ctx.JSON(http.StatusOK, apiProxy.ApiDocInfo.Output)
				})

				swagger.Swagger2ByGroup(router, apiGroupName, apiProxy.ApiDocInfo.Path(),"post", &swagger.StructParam{
					JsonData: apiProxy.ApiDocInfo.Input,
					ResponseData: apiProxy.ApiDocInfo.Output,
					Tags:[]string{apiProxy.ApiDocInfo.SrvName},
					Summary:apiProxy.ApiDocInfo.Name,
					Description:apiProxy.ApiDocInfo.Description,
				})
			}
		}

		// api gateway
		apiDataEntry := apidoc.ApiDocDataEntry
		swagger.Swagger2ByGroup(router, apiGroupName, apiDataEntry.Path(),"post", &swagger.StructParam{
			JsonData: apiDataEntry.Input,
			ResponseData: apiDataEntry.Output,
			Tags:[]string{"Api uniform data layer"},
			Summary:apiDataEntry.Name,
			Description:apiDataEntry.Description + buildErrMsg(apiGroupName),
		})
	}

	initDoc(apibackend.HttpRouterApi)
	initDoc(apibackend.HttpRouterUser)
	initDoc(apibackend.HttpRouterAdmin)
}

func buildErrMsg(group string) string {
	var codes []int
	for c, _ := range apibackend.GetGroupErrMsg(group) {
		codes = append(codes, c)
	}
	sort.Ints(codes)

	errs := "<br><b>错误码：</b>"
	for _, c := range codes {
		errs += fmt.Sprintf("<br>%d-%s", c, apibackend.GetErrMsg(c))
	}

	return errs
}
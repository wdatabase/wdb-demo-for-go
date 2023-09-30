package routes

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	wdb_drive "github.com/wdatabase/wdb-drive-for-go"
)

var (
	router = gin.Default()
	wdb    = wdb_drive.Wdb{Host: "http://127.0.0.1:8000", Key: "key"}
	logger = log.New(os.Stdout, "<wdb demo>", log.Lshortfile|log.Ldate|log.Ltime)
)

func Run() {
	router.ForwardedByClientIP = true
	router.SetTrustedProxies([]string{"127.0.0.1"})
	router.Use(Cors())
	router.MaxMultipartMemory = 2 * 1024 * 1024 * 1024

	getRoutes()
	router.Run(":8081")
}

func getRoutes() {
	api := router.Group("/api")
	api.POST("/reg", ApiReg)
	api.POST("/login", ApiLogin)

	text := router.Group("/text")
	text.POST("/post", TextPost)
	text.GET("/info", GetTextInfo)
	text.GET("/list", TextList)
	text.DELETE("/del", TextDel)

	search := router.Group("/search")
	search.POST("/post", SearchPost)
	search.GET("/info", GetSearchInfo)
	search.POST("/list", SearchList)
	search.DELETE("/del", SearchDel)

	img := router.Group("/img")
	img.POST("/post", ImgPost)
	img.GET("/info", GetImgInfo)
	img.GET("/data", GetImgData)
	img.GET("/list", ImgList)
	img.DELETE("/del", ImgDel)

	video := router.Group("/video")
	video.POST("/post", VideoPost)
	video.GET("/info", GetVideoInfo)
	video.GET("/data", GetVideoData)
	video.GET("/list", VideoList)
	video.DELETE("/del", VideoDel)

	file := router.Group("/file")
	file.POST("/post", FilePost)
	file.GET("/info", GetFileInfo)
	file.GET("/data", GetFileData)
	file.GET("/list", FileList)
	file.DELETE("/del", FileDel)

	shop := router.Group("/shop")
	shop.POST("/balance", ShopBalance)
	shop.GET("/info", GetShopInfo)
	shop.POST("/categorize/post", ShopCategorizePost)
	shop.POST("/categorize/list", ShopCategorizeList)
	shop.GET("/categorize/info", GetShopCategorizeInfo)
	shop.DELETE("/categorize/del", ShopCategorizeDel)
	shop.POST("/cart/add", ShopCartAdd)
	shop.GET("/cart/list", ShopCartList)
	shop.DELETE("/cart/del", ShopCartDel)
	shop.POST("/pro/post", ShopProPost)
	shop.GET("/pro/info", GetShopProInfo)
	shop.POST("/pro/list", ShopProList)
	shop.DELETE("/pro/del", ShopProDel)
	shop.POST("/pro/img/post", ShopImgProPost)
	shop.GET("/pro/img/data", ShopProImgData)
	shop.POST("/order/create", ShopOrderCreate)
	shop.GET("/order/info", OrderInfo)
	shop.POST("/order/list", ShopOrderList)

	big := router.Group("/big")
	big.POST("/file/upload", BigUpload)
	big.GET("/file/down", BigDown)
}

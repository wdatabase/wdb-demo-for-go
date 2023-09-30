package routes

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ShopProInfo struct {
	Uuid       string   `json:"uuid"`
	Title      string   `json:"title"`
	Price      float64  `json:"price"`
	Weight     float64  `json:"weight"`
	Inventory  uint64   `json:"inventory"`
	Tps        []string `json:"tps"`
	Imgid      string   `json:"imgid"`
	CreateTime uint64   `json:"createTime"`
	UpdateTime uint64   `json:"updateTime"`
}

func ShopProPost(c *gin.Context) {
	var body struct {
		O         string   `json:"o"`
		Uuid      string   `json:"uuid"`
		Title     string   `json:"title"`
		Price     float64  `json:"price"`
		Weight    float64  `json:"weight"`
		Inventory uint64   `json:"inventory"`
		Tps       []string `json:"tps"`
		Imgid     string   `json:"imgid"`
	}
	if err := c.BindJSON(&body); err != nil {
		rsp_err(c, 500, err)
		return
	}
	is_login, uid := auth(body.O)
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	tm := uint64(time.Now().Unix())

	index_keys := []string{}
	for _, ctp := range body.Tps {
		index_keys = append(index_keys, fmt.Sprintf("shop_pro_tp_%s", ctp))
	}
	index_keys = append(index_keys, fmt.Sprintf("all_shop_pro_tp_%s", uid))

	index_raw := []string{
		fmt.Sprintf("title:str:=%s", body.Title),
		fmt.Sprintf("price:num:=%f", body.Price),
		fmt.Sprintf("weight:num:=%f", body.Weight),
		fmt.Sprintf("updateTime:num:=%d", tm),
	}

	var rsp InfoRsp
	if body.Uuid == "" {
		cuuid := uuid.New().String()

		var shopProInfo ShopProInfo
		shopProInfo.Uuid = cuuid
		shopProInfo.Title = body.Title
		shopProInfo.Price = body.Price
		shopProInfo.Weight = body.Weight
		shopProInfo.Inventory = body.Inventory
		shopProInfo.Tps = body.Tps
		shopProInfo.Imgid = body.Imgid
		shopProInfo.CreateTime = tm
		shopProInfo.UpdateTime = tm

		data, err := obj_to_json(c, &shopProInfo)
		if err != nil {
			rsp_err(c, 500, err)
			return
		}

		apiRsp := wdb.CreateObj(cuuid, data, []string{})
		if apiRsp.Code == 200 {
			idxRsp := wdb.CreateIndex(index_keys, cuuid, index_raw)
			if idxRsp.Code != 200 {
				rsp_err(c, 500, idxRsp.Msg)
				return
			}
		} else {
			rsp_err(c, 500, err)
			return
		}
	} else {
		var shopProInfo ShopProInfo
		if err := load_obj(c, body.Uuid, &shopProInfo); err != nil {
			rsp_err(c, 500, err)
			return
		}

		old_index_keys := []string{}
		for _, ctp := range shopProInfo.Tps {
			old_index_keys = append(old_index_keys, fmt.Sprintf("shop_pro_tp_%s", ctp))
		}
		old_index_keys = append(old_index_keys, fmt.Sprintf("all_shop_pro_tp_%s", uid))

		shopProInfo.Title = body.Title
		shopProInfo.Price = body.Price
		shopProInfo.Weight = body.Weight
		shopProInfo.Inventory = body.Inventory
		shopProInfo.Tps = body.Tps
		shopProInfo.Imgid = body.Imgid
		shopProInfo.UpdateTime = tm

		data, err := obj_to_json(c, &shopProInfo)
		if err != nil {
			rsp_err(c, 500, err)
			return
		}

		upRsp := wdb.UpdateObj(body.Uuid, data)
		if upRsp.Code == 200 {
			idxRsp := wdb.UpdateIndex(old_index_keys, index_keys, shopProInfo.Uuid, index_raw)
			if idxRsp.Code != 200 {
				rsp_err(c, 500, idxRsp.Msg)
				return
			}
		} else {
			rsp_err(c, 500, rsp.Msg)
			return
		}
	}
	rsp_ok(c, body.Uuid)
}

func GetShopProInfo(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}
	cuuid := c.Query("uuid")
	var info ShopProInfo
	if err := load_obj(c, cuuid, &info); err != nil {
		rsp_err(c, 500, err)
		return
	}

	var info_rsp struct {
		Code int64       `json:"code"`
		Msg  string      `json:"msg"`
		Info ShopProInfo `json:"info"`
	}
	info_rsp.Code = 200
	info_rsp.Msg = ""
	info_rsp.Info = info

	c.JSON(http.StatusOK, info_rsp)
}

func ShopProList(c *gin.Context) {
	var body struct {
		O        string `json:"o"`
		Indexkey string `json:"indexkey"`
		Titlekey string `json:"titlekey"`
		Begin    string `json:"begin"`
		End      string `json:"end"`
		Offset   uint64 `json:"offset"`
		Limit    uint64 `json:"limit"`
		Order    string `json:"order"`
	}
	if err := c.BindJSON(&body); err != nil {
		rsp_err(c, 500, err)
		return
	}
	is_login, uid := auth(body.O)
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	indexkey := body.Indexkey
	if indexkey == "all" {
		indexkey = fmt.Sprintf("all_shop_pro_tp_%s", uid)
	} else {
		indexkey = fmt.Sprintf("shop_pro_tp_%s", indexkey)
	}

	arr := []string{}
	if body.Titlekey != "" {
		arr = append(arr, fmt.Sprintf("[\"title\",\"reg\",\"^.*%s.*$\"]", body.Titlekey))
	}
	if body.Begin != "" && body.End == "" {
		arr = append(arr, fmt.Sprintf("[\"price\",\">=\",%s]", body.Begin))
	}
	if body.End == "" && body.End != "" {
		arr = append(arr, fmt.Sprintf("[\"price\",\"<=\",%s]", body.End))
	}
	if body.Begin != "" && body.End != "" {
		arr = append(arr, fmt.Sprintf("[\"price\",\">=\",%s,\"<=\",%s]", body.Begin, body.End))
	}

	condition := ""
	if len(arr) == 1 {
		condition = arr[0]
	} else if len(arr) > 1 {
		condition = fmt.Sprintf("{\"and\":[%s]}", strings.Join(arr, ","))
	}

	order := "updateTime DESC"
	if body.Order == "tasc" {
		order = "updateTime ASC"
	} else if body.Order == "tdesc" {
		order = "updateTime DESC"
	} else if body.Order == "pasc" {
		order = "price ASC"
	} else if body.Order == "pdesc" {
		order = "price DESC"
	} else if body.Order == "wasc" {
		order = "weight ASC"
	} else if body.Order == "wdesc" {
		order = "weight DESC"
	}

	listRsp := wdb.ListIndex(indexkey, condition, body.Offset, body.Limit, order)
	if listRsp.Code != 200 {
		rsp_err(c, 500, listRsp.Msg)
		return
	}

	blist := []ShopProInfo{}
	for _, item := range listRsp.List {
		var cinfo ShopProInfo
		if err := load_obj_by_str(c, item, &cinfo); err != nil {
			rsp_err(c, 500, err)
			return
		}

		blist = append(blist, cinfo)
	}

	var proListRsp struct {
		Code  uint64        `json:"code"`
		Msg   string        `json:"msg"`
		Total uint64        `json:"total"`
		List  []ShopProInfo `json:"list"`
	}
	proListRsp.Code = 200
	proListRsp.Msg = ""
	proListRsp.Total = listRsp.Total
	proListRsp.List = blist

	c.JSON(http.StatusOK, proListRsp)
}

func ShopProDel(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	cuuid := c.Query("uuid")
	apiRsp := wdb.DelObj(cuuid)
	if apiRsp.Code != 200 {
		rsp_err(c, 500, apiRsp.Msg)
		return
	}

	c.JSON(http.StatusOK, InfoRsp{
		Code: 200,
		Msg:  "",
		Uuid: cuuid,
	})
}

func ShopImgProPost(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	file, err := c.FormFile("img")
	if err != nil {
		rsp_err(c, 500, err)
		return
	}
	src, err := file.Open()
	if err != nil {
		rsp_err(c, 500, err)
		return
	}
	defer src.Close()
	buf := bytes.NewBuffer([]byte{})
	if _, err := io.Copy(buf, src); err != nil {
		rsp_err(c, 500, err)
		return
	}
	contentType := http.DetectContentType(buf.Bytes())
	data := base64.StdEncoding.EncodeToString(buf.Bytes())
	img_raw := ImgRaw{Raw: data}
	fileUuid := uuid.New().String()

	rawdata, err := obj_to_json(c, &img_raw)
	if err != nil {
		rsp_err(c, 500, err)
		return
	}

	imgRsp := wdb.CreateObj(fileUuid, rawdata, []string{})
	if imgRsp.Code != 200 {
		rsp_err(c, 500, imgRsp.Msg)
		return
	}

	cuuid := uuid.New().String()

	var img_info ImgInfo
	img_info.Uuid = cuuid
	img_info.Name = "img"
	img_info.FileName = file.Filename
	img_info.ContentType = contentType
	img_info.Size = uint64(file.Size)
	img_info.FileUuid = fileUuid
	img_info.CreateTime = uint64(time.Now().Unix())
	img_info.UpdateTime = uint64(time.Now().Unix())

	infodata, errc := obj_to_json(c, &img_info)
	if errc != nil {
		rsp_err(c, 500, errc)
		return
	}

	infoRsp := wdb.CreateObj(cuuid, infodata, []string{})
	if infoRsp.Code == 200 {
		c.JSON(http.StatusOK, ImgRsp{
			Code: 200,
			Msg:  "",
			Uuid: cuuid,
		})
	} else {
		c.JSON(http.StatusOK, ImgRsp{
			Code: 400,
			Msg:  infoRsp.Msg,
			Uuid: "",
		})
	}

}

func ShopProImgData(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}
	cuuid := c.Query("uuid")
	var info ImgInfo
	if err := load_obj(c, cuuid, &info); err != nil {
		rsp_err(c, 500, err)
		return
	}

	var raw_info ImgRaw
	if err := load_obj(c, info.FileUuid, &raw_info); err != nil {
		rsp_err(c, 500, err)
		return
	}

	raw, err := base64.StdEncoding.DecodeString(raw_info.Raw)
	if err != nil {
		rsp_err(c, 500, err)
		return
	}

	c.Data(http.StatusOK, info.ContentType, raw)
}

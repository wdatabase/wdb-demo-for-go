package routes

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ShopCategorizeInfo struct {
	Uuid       string `json:"uuid"`
	Name       string `json:"name"`
	Sort       uint64 `json:"sort"`
	CreateTime uint64 `json:"createTime"`
	UpdateTime uint64 `json:"updateTime"`
}

type ShopCategorizeListInfo struct {
	Uuid string `json:"uuid"`
	Name string `json:"name"`
	Sort uint64 `json:"sort"`
}

type ShopCategorizeListRsp struct {
	Code  uint64                   `json:"code"`
	Msg   string                   `json:"msg"`
	Total uint64                   `json:"total"`
	List  []ShopCategorizeListInfo `json:"list"`
}

func ShopCategorizePost(c *gin.Context) {
	var body struct {
		O    string `json:"o"`
		Uuid string `json:"uuid"`
		Name string `json:"name"`
		Sort uint64 `json:"sort"`
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
	if body.Uuid == "" {
		cuuid := uuid.New().String()

		var categorizeInfo_info ShopCategorizeInfo
		categorizeInfo_info.Uuid = cuuid
		categorizeInfo_info.Name = body.Name
		categorizeInfo_info.Sort = body.Sort
		categorizeInfo_info.CreateTime = uint64(time.Now().Unix())
		categorizeInfo_info.UpdateTime = uint64(time.Now().Unix())

		data, err := obj_to_json(c, &categorizeInfo_info)
		if err != nil {
			rsp_err(c, 500, err)
			return
		}

		category := fmt.Sprintf("shop_categorize_%s", uid)
		apiRsp := wdb.CreateObj(cuuid, data, []string{category})
		if apiRsp.Code == 200 {
			c.JSON(http.StatusOK, InfoRsp{
				Code: 200,
				Msg:  "",
				Uuid: cuuid,
			})
		} else {
			c.JSON(http.StatusOK, InfoRsp{
				Code: 400,
				Msg:  fmt.Sprintf("%s", err),
				Uuid: "",
			})
		}
	} else {
		var categorizeInfo_info ShopCategorizeInfo
		if err := load_obj(c, body.Uuid, &categorizeInfo_info); err != nil {
			rsp_err(c, 500, err)
			return
		}

		categorizeInfo_info.Name = body.Name
		categorizeInfo_info.Sort = body.Sort
		categorizeInfo_info.UpdateTime = uint64(time.Now().Unix())

		data, err := obj_to_json(c, &categorizeInfo_info)
		if err != nil {
			rsp_err(c, 500, err)
			return
		}

		upRsp := wdb.UpdateObj(body.Uuid, data)
		if upRsp.Code == 200 {
			c.JSON(http.StatusOK, InfoRsp{
				Code: 200,
				Msg:  "",
				Uuid: body.Uuid,
			})
		} else {
			c.JSON(http.StatusOK, InfoRsp{
				Code: 400,
				Msg:  fmt.Sprintf("%s", err),
				Uuid: "",
			})
		}
	}
}

func ShopCategorizeList(c *gin.Context) {
	var body struct {
		O      string `json:"o"`
		Offset uint64 `json:"offset"`
		Limit  uint64 `json:"limit"`
		Order  string `json:"order"`
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

	category := fmt.Sprintf("shop_categorize_%s", uid)

	order := body.Order

	listRsp := wdb.ListObj(category, body.Offset, body.Limit, order)
	if listRsp.Code != 200 {
		rsp_err(c, 500, listRsp.Msg)
		return
	}

	blist := []ShopCategorizeListInfo{}
	for _, item := range listRsp.List {
		var cinfo ShopCategorizeInfo
		if err := load_obj_by_str(c, item, &cinfo); err != nil {
			return
		}

		blist = append(blist, ShopCategorizeListInfo{
			Uuid: cinfo.Uuid,
			Name: cinfo.Name,
			Sort: cinfo.Sort,
		})
	}
	sort.Slice(blist, func(i, j int) bool {
		return blist[i].Sort > blist[j].Sort
	})

	c.JSON(http.StatusOK, ShopCategorizeListRsp{
		Code:  200,
		Msg:   "",
		Total: listRsp.Total,
		List:  blist,
	})
}

func GetShopCategorizeInfo(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}
	cuuid := c.Query("uuid")
	var info ShopCategorizeInfo
	if err := load_obj(c, cuuid, &info); err != nil {
		rsp_err(c, 500, err)
		return
	}

	var info_rsp struct {
		Code int64              `json:"code"`
		Msg  string             `json:"msg"`
		Info ShopCategorizeInfo `json:"info"`
	}
	info_rsp.Code = 200
	info_rsp.Msg = ""
	info_rsp.Info = info

	c.JSON(http.StatusOK, info_rsp)
}

func ShopCategorizeDel(c *gin.Context) {
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

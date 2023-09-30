package routes

import (
	"fmt"
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ShopCartInfo struct {
	Uuid       string   `json:"uuid,omitempty"`
	Uid        string   `json:"uid,omitempty"`
	Ids        []string `json:"ids,omitempty"`
	Nums       []uint64 `json:"nums,omitempty"`
	CreateTime uint64   `json:"createTime,omitempty"`
	UpdateTime uint64   `json:"updateTime,omitempty"`
}

type ShopCartListInfo struct {
	Proid     string  `json:"proid,omitempty"`
	Title     string  `json:"title,omitempty"`
	Price     float64 `json:"price,omitempty"`
	Inventory uint64  `json:"inventory,omitempty"`
	Imgid     string  `json:"imgid,omitempty"`
	Num       uint64  `json:"num,omitempty"`
}

func ShopCartAdd(c *gin.Context) {
	var body struct {
		O    string `json:"o,omitempty"`
		Uuid string `json:"uuid,omitempty"`
		Num  uint64 `json:"num,omitempty"`
	}
	if err := c.BindJSON(&body); err != nil {
		logger.Println(err)
		rsp_err(c, 500, err)
		return
	}
	is_login, uid := auth(body.O)
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	var cartInfoRsp struct {
		Code uint64       `json:"code"`
		Msg  string       `json:"msg"`
		Info ShopCartInfo `json:"list"`
	}

	key := fmt.Sprintf("shop_cart_%s", uid)
	infoRsp := wdb.GetObj(key)
	if infoRsp.Code == 200 {
		var info ShopCartInfo
		if err := load_obj_by_str(c, infoRsp.Data, &info); err != nil {
			logger.Println(err)
			rsp_err(c, 500, err)
			return
		}

		ids := info.Ids
		nums := info.Nums

		is_contain, index := contains(ids, body.Uuid)
		if is_contain {
			nums[index] = body.Num
		} else {
			ids = append(ids, body.Uuid)
			nums = append(nums, body.Num)
		}

		info.Ids = ids
		info.Nums = nums

		data, err := obj_to_json(c, &info)
		if err != nil {
			logger.Println(err)
			rsp_err(c, 500, err)
			return
		}
		upRsp := wdb.UpdateObj(key, data)
		if upRsp.Code != 200 {
			logger.Println(upRsp.Msg)
			rsp_err(c, 500, upRsp.Msg)
			return
		}

		cartInfoRsp.Code = 200
		cartInfoRsp.Msg = ""
		cartInfoRsp.Info = info

		c.JSON(http.StatusOK, cartInfoRsp)
	} else {
		if infoRsp.Msg == "not found key" {
			cuuid := uuid.New().String()
			tm := uint64(time.Now().Unix())
			var info ShopCartInfo
			info.Uuid = cuuid
			info.Uid = uid
			info.Ids = []string{body.Uuid}
			info.Nums = []uint64{body.Num}
			info.CreateTime = tm
			info.UpdateTime = tm

			data, err := obj_to_json(c, &info)
			if err != nil {
				logger.Println(err)
				rsp_err(c, 500, err)
				return
			}

			createRsp := wdb.CreateObj(key, data, []string{})
			if createRsp.Code == 200 {

				cartInfoRsp.Code = 200
				cartInfoRsp.Msg = ""
				cartInfoRsp.Info = info

				c.JSON(http.StatusOK, cartInfoRsp)
			} else {
				logger.Println(createRsp.Msg)
				rsp_err(c, 500, createRsp.Msg)
			}
		} else {
			logger.Println(infoRsp.Msg)
			rsp_err(c, 500, infoRsp.Msg)
		}
	}
}

func ShopCartList(c *gin.Context) {
	is_login, uid := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	var info ShopCartInfo
	key := fmt.Sprintf("shop_cart_%s", uid)
	if err := load_obj(c, key, &info); err != nil {
		rsp_err(c, 500, err)
		return
	}

	nums := info.Nums

	blist := []ShopCartListInfo{}
	total := 0.0
	for idx, ckey := range info.Ids {
		cnum := nums[idx]
		var proInfo ShopProInfo
		if err := load_obj(c, ckey, &proInfo); err != nil {
			rsp_err(c, 500, err)
			return
		}

		total += proInfo.Price * float64(cnum)

		blist = append(blist, ShopCartListInfo{
			Proid:     proInfo.Uuid,
			Title:     proInfo.Title,
			Price:     proInfo.Price,
			Inventory: proInfo.Inventory,
			Imgid:     proInfo.Imgid,
			Num:       cnum,
		})
	}
	total = math.Round(total*100.0) / 100.0
	var cartListRsp struct {
		Code  uint64             `json:"code"`
		Msg   string             `json:"msg"`
		Total float64            `json:"total"`
		List  []ShopCartListInfo `json:"listinfo"`
	}
	cartListRsp.Code = 200
	cartListRsp.Msg = ""
	cartListRsp.Total = total
	cartListRsp.List = blist

	c.JSON(http.StatusOK, cartListRsp)
}

func ShopCartDel(c *gin.Context) {
	is_login, uid := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}
	cuuid := c.Query("uuid")

	var info ShopCartInfo
	key := fmt.Sprintf("shop_cart_%s", uid)
	if err := load_obj(c, key, &info); err != nil {
		rsp_err(c, 500, err)
		return
	}

	if cuuid == "all" {
		info.Ids = []string{}
		info.Nums = []uint64{}
	} else {
		ids := info.Ids
		nums := info.Nums

		is_contain, index := contains(ids, cuuid)
		if is_contain {
			info.Ids = remove(ids, index)
			info.Nums = remove(nums, index)
		}
	}

	data, err := obj_to_json(c, &info)
	if err != nil {
		rsp_err(c, 500, err)
		return
	}
	upRsp := wdb.UpdateObj(key, data)
	if upRsp.Code != 200 {
		rsp_err(c, 500, upRsp.Msg)
		return
	}

	c.JSON(http.StatusOK, InfoRsp{
		Code: 200,
		Msg:  "",
		Uuid: cuuid,
	})
}

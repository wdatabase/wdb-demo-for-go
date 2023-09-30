package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ShopInfo struct {
	Uuid       string  `json:"uuid"`
	Uid        string  `json:"uid"`
	Balance    float64 `json:"balance"`
	Point      float64 `json:"point"`
	CreateTime uint64  `json:"createTime"`
	UpdateTime uint64  `json:"updateTime"`
}

type ShopBalanceLog struct {
	Uuid       string  `json:"uuid"`
	Uid        string  `json:"uid"`
	Balance    float64 `json:"balance"`
	Op         string  `json:"op"`
	CreateTime uint64  `json:"createTime"`
	UpdateTime uint64  `json:"updateTime"`
}

type ShopRsp struct {
	Code uint64 `json:"code"`
	Msg  string `json:"msg"`
	Uuid string `json:"uuid"`
}

type ShopInfoRsp struct {
	Code uint64   `json:"code"`
	Msg  string   `json:"msg"`
	Info ShopInfo `json:"info"`
}

func ShopBalance(c *gin.Context) {
	var body struct {
		O       string  `json:"o"`
		Balance float64 `json:"balance"`
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

	key := fmt.Sprintf("shop_info_%s", uid)
	tm := uint64(time.Now().Unix())

	tsBeginRsp := wdb.TransBegin([]string{key})
	tsid := ""
	if tsBeginRsp.Code == 200 {
		tsid = tsBeginRsp.Data
	} else {
		logger.Println(tsBeginRsp.Msg)
		rsp_err(c, 500, tsBeginRsp.Msg)
		return
	}

	shopInfoRsp := wdb.TransGet(tsid, key)
	if shopInfoRsp.Code != 200 {
		wdb.TransRollBack(tsid)
		logger.Println(shopInfoRsp.Msg)
		rsp_err(c, 500, shopInfoRsp.Msg)
		return
	}
	var shopInfo ShopInfo
	if err := load_obj_by_str(c, shopInfoRsp.Data, &shopInfo); err != nil {
		wdb.TransRollBack(tsid)
		logger.Println(err)
		rsp_err(c, 500, err)
		return
	}

	var balanceLog ShopBalanceLog
	cuuid := uuid.New().String()
	balanceLog.Uuid = cuuid
	balanceLog.Uid = uid
	balanceLog.Balance = body.Balance
	balanceLog.Op = "in"
	balanceLog.CreateTime = tm
	balanceLog.UpdateTime = tm

	logdata, err := obj_to_json(c, &balanceLog)
	if err != nil {
		wdb.TransRollBack(tsid)
		logger.Println(err)
		rsp_err(c, 500, err)
		return
	}
	logRsp := wdb.TransCreateObj(tsid, cuuid, logdata, []string{fmt.Sprintf("shop_balance_log_%s", uid)})
	if logRsp.Code != 200 {
		wdb.TransRollBack(tsid)
		logger.Println(logRsp.Msg)
		rsp_err(c, 500, logRsp.Msg)
		return
	}

	shopInfo.Balance = shopInfo.Balance + body.Balance
	shopInfo.UpdateTime = tm
	infodata, err := obj_to_json(c, &shopInfo)
	if err != nil {
		wdb.TransRollBack(tsid)
		logger.Println(err)
		rsp_err(c, 500, err)
		return
	}
	upInfoRsp := wdb.TransUpdateObj(tsid, key, infodata)
	if upInfoRsp.Code != 200 {
		wdb.TransRollBack(tsid)
		logger.Println(upInfoRsp.Msg)
		rsp_err(c, 500, upInfoRsp.Msg)
		return
	}

	commitRsp := wdb.TransCommit(tsid)
	if commitRsp.Code == 200 {
		c.JSON(http.StatusOK, ShopRsp{
			Code: 200,
			Msg:  "",
			Uuid: "",
		})
	} else {
		wdb.TransRollBack(tsid)
		logger.Println(commitRsp.Msg)
		c.JSON(http.StatusOK, InfoRsp{
			Code: 400,
			Msg:  commitRsp.Msg,
			Uuid: "",
		})
	}
}

func GetShopInfo(c *gin.Context) {
	is_login, uid := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	key := fmt.Sprintf("shop_info_%s", uid)
	infoRsp := wdb.GetObj(key)
	if infoRsp.Code == 200 {
		var info ShopInfo
		if err := load_obj_by_str(c, infoRsp.Data, &info); err != nil {
			logger.Println(err)
			rsp_err(c, 500, err)
			return
		}
		info.Balance = fixfloat64(info.Balance)
		info.Point = fixfloat64(info.Point)
		c.JSON(http.StatusOK, ShopInfoRsp{
			Code: 200,
			Msg:  "",
			Info: info,
		})
	} else {
		if infoRsp.Msg == "not found key" {
			cuuid := uuid.New().String()
			tm := uint64(time.Now().Unix())
			var info ShopInfo
			info.Uuid = cuuid
			info.Uid = uid
			info.Balance = 0.0
			info.Point = 0.0
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
				c.JSON(http.StatusOK, ShopInfoRsp{
					Code: 200,
					Msg:  "",
					Info: info,
				})
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

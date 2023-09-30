package routes

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ShopOrderInfo struct {
	Uuid       string    `json:"uuid"`
	Title      string    `json:"title"`
	Imgid      string    `json:"imgid"`
	Total      float64   `json:"total"`
	Ids        []string  `json:"ids"`
	Nums       []uint64  `json:"nums"`
	Prices     []float64 `json:"prices"`
	CreateTime uint64    `json:"createTime"`
	UpdateTime uint64    `json:"updateTime"`
}

type ShopOrderItem struct {
	Uuid       string  `json:"uuid"`
	Title      string  `json:"title"`
	Imgid      string  `json:"imgid"`
	Price      float64 `json:"price"`
	Num        uint64  `json:"num"`
	CreateTime uint64  `json:"createTime"`
	UpdateTime uint64  `json:"updateTime"`
}

func ShopOrderCreate(c *gin.Context) {
	var body struct {
		O      string    `json:"o"`
		Total  float64   `json:"total"`
		Ids    []string  `json:"ids"`
		Nums   []uint64  `json:"nums"`
		Prices []float64 `json:"prices"`
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

	tm := uint64(time.Now().Unix())
	req_ids := body.Ids
	req_nums := body.Nums
	req_price := body.Prices

	lock_ids := make([]string, len(req_ids))
	copy(lock_ids, req_ids)
	shop_info_key := fmt.Sprintf("shop_info_%s", uid)
	lock_ids = append(lock_ids, shop_info_key)

	//开始事务
	tsBeginRsp := wdb.TransBegin(lock_ids)
	tsid := ""
	if tsBeginRsp.Code == 200 {
		tsid = tsBeginRsp.Data
	} else {
		logger.Println(tsBeginRsp.Msg)
		rsp_err(c, 500, tsBeginRsp.Msg)
		return
	}

	//校验余额
	var shopInfo ShopInfo
	if err := load_trans_obj(c, tsid, shop_info_key, &shopInfo); err != nil {
		logger.Println(err)
		wdb.TransRollBack(tsid)
		rsp_err(c, 500, err)
		return
	}
	if float64cmp(shopInfo.Balance, body.Total) == -1 {
		logger.Println("余额不足")
		wdb.TransRollBack(tsid)
		rsp_err(c, 500, "余额不足")
		return
	}

	orderid := uuid.New().String()
	imgid := ""
	title_list := []string{}
	category_item := fmt.Sprintf("shop_order_item_%s", orderid)

	//遍历购物车相关商品
	for idx, ckey := range req_ids {
		cnum := req_nums[idx]
		cprice := req_price[idx]

		tsRsp := wdb.TransGet(tsid, ckey)
		if tsRsp.Code != 200 {
			logger.Println(tsRsp.Msg)
			wdb.TransRollBack(tsid)
			rsp_err(c, 500, tsRsp.Msg)
			return
		}
		var proInfo ShopProInfo
		if err := load_obj_by_str(c, tsRsp.Data, &proInfo); err != nil {
			logger.Println(err)
			wdb.TransRollBack(tsid)
			rsp_err(c, 500, err)
			return
		}

		//校验价格
		if float64cmp(proInfo.Price, cprice) != 0 {
			logger.Println("商品价格变动，请重新确认。")
			wdb.TransRollBack(tsid)
			rsp_err(c, 500, "商品价格变动，请重新确认。")
			return
		}

		//校验库存
		if cnum > proInfo.Inventory {
			logger.Println("库存不足。")
			wdb.TransRollBack(tsid)
			rsp_err(c, 500, "库存不足。")
			return
		}

		title_list = append(title_list, proInfo.Title)
		imgid = proInfo.Imgid

		//保存订单产品详情
		var orderItem ShopOrderItem
		itemid := uuid.New().String()
		orderItem.Uuid = itemid
		orderItem.Title = proInfo.Title
		orderItem.Imgid = proInfo.Imgid
		orderItem.Num = cnum
		orderItem.Price = cprice
		orderItem.CreateTime = tm
		orderItem.UpdateTime = tm

		itemdata, err := obj_to_json(c, &orderItem)
		if err != nil {
			logger.Println(err)
			wdb.TransRollBack(tsid)
			rsp_err(c, 500, err)
			return
		}

		itemRsp := wdb.TransCreateObj(tsid, itemid, itemdata, []string{category_item})
		if itemRsp.Code != 200 {
			logger.Println(itemRsp.Msg)
			wdb.TransRollBack(tsid)
			rsp_err(c, 500, itemRsp.Msg)
			return
		}

		//减库存
		proInfo.Inventory = proInfo.Inventory - cnum
		prodata, err := obj_to_json(c, &proInfo)
		if err != nil {
			logger.Println(err)
			wdb.TransRollBack(tsid)
			rsp_err(c, 500, err)
			return
		}
		proRsp := wdb.TransUpdateObj(tsid, proInfo.Uuid, prodata)
		if proRsp.Code != 200 {
			logger.Println(proRsp.Msg)
			wdb.TransRollBack(tsid)
			rsp_err(c, 500, proRsp.Msg)
			return
		}
	}

	titles := strings.Join(title_list, "/")

	//保存订单信息
	var orderInfo ShopOrderInfo
	orderInfo.Uuid = orderid
	orderInfo.Title = titles
	orderInfo.Imgid = imgid
	orderInfo.Total = body.Total
	orderInfo.Ids = req_ids
	orderInfo.Nums = req_nums
	orderInfo.Prices = req_price
	orderInfo.CreateTime = tm
	orderInfo.UpdateTime = tm
	oddata, err := obj_to_json(c, &orderInfo)
	if err != nil {
		logger.Println(err)
		wdb.TransRollBack(tsid)
		rsp_err(c, 500, err)
		return
	}
	odRsp := wdb.TransCreateObj(tsid, orderid, oddata, []string{})
	if odRsp.Code != 200 {
		logger.Println(odRsp.Msg)
		wdb.TransRollBack(tsid)
		rsp_err(c, 500, odRsp.Msg)
		return
	}

	//创建索引
	index_keys := []string{fmt.Sprintf("shop_order_index_%s", uid)}
	indexraw := []string{
		fmt.Sprintf("title:str:=%s", orderInfo.Title),
		fmt.Sprintf("total:num:=%f", orderInfo.Total),
		fmt.Sprintf("updateTime:num:=%d", orderInfo.UpdateTime),
	}
	idxRsp := wdb.CreateIndex(index_keys, orderid, indexraw)
	if idxRsp.Code != 200 {
		wdb.TransRollBack(tsid)
		rsp_err(c, 500, idxRsp.Msg)
		return
	}

	//更新余额积分
	shopInfo.Balance = shopInfo.Balance - orderInfo.Total
	shopInfo.Point = shopInfo.Point + orderInfo.Total
	spdata, err := obj_to_json(c, &shopInfo)
	if err != nil {
		wdb.TransRollBack(tsid)
		rsp_err(c, 500, err)
		return
	}
	upInfoRsp := wdb.TransUpdateObj(tsid, shop_info_key, spdata)
	if upInfoRsp.Code != 200 {
		wdb.TransRollBack(tsid)
		rsp_err(c, 500, upInfoRsp.Msg)
		return
	}

	//提交事务
	commitRsp := wdb.TransCommit(tsid)
	if commitRsp.Code != 200 {
		rsp_err(c, 500, commitRsp.Msg)
		wdb.TransRollBack(tsid)
		return
	}
	rsp_ok(c, tsid)
}

func OrderInfo(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	category := fmt.Sprintf("shop_order_item_%s", c.Query("uuid"))
	listRsp := wdb.ListObj(category, 0, 100, "ASC")
	if listRsp.Code != 200 {
		rsp_err(c, 500, listRsp.Msg)
		return
	}

	blist := []ShopOrderItem{}
	for _, item := range listRsp.List {
		var cinfo ShopOrderItem
		if err := load_obj_by_str(c, item, &cinfo); err != nil {
			rsp_err(c, 500, err)
			return
		}

		blist = append(blist, cinfo)
	}

	var orderItemRsp struct {
		Code uint64          `json:"code"`
		Msg  string          `json:"msg"`
		List []ShopOrderItem `json:"list"`
	}
	orderItemRsp.Code = 200
	orderItemRsp.Msg = ""
	orderItemRsp.List = blist

	c.JSON(http.StatusOK, orderItemRsp)
}

func ShopOrderList(c *gin.Context) {
	var body struct {
		O        string `json:"o"`
		Titlekey string `json:"titlekey"`
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

	arr := []string{}
	if body.Titlekey != "" {
		arr = append(arr, fmt.Sprintf("[\"title\",\"reg\",\"^.*%s.*$\"]", body.Titlekey))
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
		order = "total ASC"
	} else if body.Order == "pdesc" {
		order = "total DESC"
	}

	indexkey := fmt.Sprintf("shop_order_index_%s", uid)
	listRsp := wdb.ListIndex(indexkey, condition, body.Offset, body.Limit, order)
	if listRsp.Code != 200 {
		rsp_err(c, 500, listRsp.Msg)
		return
	}

	blist := []ShopOrderInfo{}
	for _, item := range listRsp.List {
		var cinfo ShopOrderInfo
		if err := load_obj_by_str(c, item, &cinfo); err != nil {
			rsp_err(c, 500, err)
			return
		}

		blist = append(blist, cinfo)
	}

	var orderListRsp struct {
		Code  uint64          `json:"code"`
		Msg   string          `json:"msg"`
		Total uint64          `json:"total"`
		List  []ShopOrderInfo `json:"list"`
	}
	orderListRsp.Code = 200
	orderListRsp.Msg = ""
	orderListRsp.Total = listRsp.Total
	orderListRsp.List = blist

	c.JSON(http.StatusOK, orderListRsp)
}

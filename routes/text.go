package routes

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type TextInfo struct {
	Uuid       string `json:"uuid"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	CreateTime uint64 `json:"createTime"`
	UpdateTime uint64 `json:"updateTime"`
}

type InfoRsp struct {
	Code uint64 `json:"code"`
	Msg  string `json:"msg"`
	Uuid string `json:"uuid"`
}

type TextListInfo struct {
	Uuid  string `json:"uuid"`
	Title string `json:"title"`
	Time  uint64 `json:"time"`
}

type TextListRsp struct {
	Code  uint64         `json:"code"`
	Msg   string         `json:"msg"`
	Total uint64         `json:"total"`
	List  []TextListInfo `json:"list"`
}

func TextPost(c *gin.Context) {
	var body struct {
		O       string `json:"o,omitempty"`
		Uuid    string `json:"uuid,omitempty"`
		Title   string `json:"title,omitempty"`
		Content string `json:"content,omitempty"`
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

		var text_info TextInfo
		text_info.Uuid = cuuid
		text_info.Title = body.Title
		text_info.Content = body.Content
		text_info.CreateTime = uint64(time.Now().Unix())
		text_info.UpdateTime = uint64(time.Now().Unix())

		data, err := obj_to_json(c, &text_info)
		if err != nil {
			rsp_err(c, 500, err)
			return
		}

		category := fmt.Sprintf("my_text_%s", uid)
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
		var text_info TextInfo
		if err := load_obj(c, body.Uuid, &text_info); err != nil {
			rsp_err(c, 500, err)
			return
		}

		text_info.Title = body.Title
		text_info.Content = body.Content
		text_info.UpdateTime = uint64(time.Now().Unix())

		data, err := obj_to_json(c, &text_info)
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

func GetTextInfo(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}
	cuuid := c.Query("uuid")
	var info TextInfo
	if err := load_obj(c, cuuid, &info); err != nil {
		return
	}

	var info_rsp struct {
		Code int64    `json:"code"`
		Msg  string   `json:"msg"`
		Info TextInfo `json:"info"`
	}
	info_rsp.Code = 200
	info_rsp.Msg = ""
	info_rsp.Info = info

	c.JSON(http.StatusOK, info_rsp)
}

func TextList(c *gin.Context) {
	is_login, uid := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	category := fmt.Sprintf("my_text_%s", uid)
	offset, err := strconv.ParseUint(c.Query("offset"), 10, 64)
	if err != nil {
		rsp_err(c, 500, err)
		return
	}
	limit, err := strconv.ParseUint(c.Query("limit"), 10, 64)
	if err != nil {
		rsp_err(c, 500, err)
		return
	}
	order := c.Query("order")

	listRsp := wdb.ListObj(category, offset, limit, order)
	if listRsp.Code != 200 {
		rsp_err(c, 500, listRsp.Msg)
		return
	}

	blist := []TextListInfo{}
	for _, item := range listRsp.List {
		var cinfo TextInfo
		if err := load_obj_by_str(c, item, &cinfo); err != nil {
			return
		}

		blist = append(blist, TextListInfo{
			Uuid:  cinfo.Uuid,
			Title: cinfo.Title,
			Time:  cinfo.CreateTime,
		})
	}

	c.JSON(http.StatusOK, TextListRsp{
		Code:  200,
		Msg:   "",
		Total: listRsp.Total,
		List:  blist,
	})
}

func TextDel(c *gin.Context) {
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

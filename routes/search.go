package routes

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SearchInfo struct {
	Uuid       string  `json:"uuid,omitempty"`
	Title      string  `json:"title,omitempty"`
	Score      float64 `json:"score,omitempty"`
	Content    string  `json:"content,omitempty"`
	CreateTime uint64  `json:"createTime,omitempty"`
	UpdateTime uint64  `json:"updateTime,omitempty"`
}

type SearchListInfo struct {
	Uuid  string  `json:"uuid,omitempty"`
	Title string  `json:"title,omitempty"`
	Time  uint64  `json:"time,omitempty"`
	Score float64 `json:"score,omitempty"`
}

func SearchPost(c *gin.Context) {
	var body struct {
		O       string  `json:"o,omitempty"`
		Uuid    string  `json:"uuid,omitempty"`
		Title   string  `json:"title,omitempty"`
		Content string  `json:"content,omitempty"`
		Score   float64 `json:"score,omitempty"`
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

	index_keys := []string{fmt.Sprintf("my_search_index_%s", uid)}
	index_raw := []string{
		fmt.Sprintf("title:str:=%s", body.Title),
		fmt.Sprintf("score:num:=%f", body.Score),
		fmt.Sprintf("updateTime:num:=%d", tm),
	}

	var rsp InfoRsp
	if body.Uuid == "" {
		cuuid := uuid.New().String()

		var searchInfo SearchInfo
		searchInfo.Uuid = cuuid
		searchInfo.Title = body.Title
		searchInfo.Score = body.Score
		searchInfo.Content = body.Content
		searchInfo.CreateTime = tm
		searchInfo.UpdateTime = tm

		data, err := obj_to_json(c, &searchInfo)
		if err != nil {
			rsp_err(c, 500, err)
			return
		}

		apiRsp := wdb.CreateObj(cuuid, data, []string{})
		if apiRsp.Code == 200 {
			idxRsp := wdb.CreateIndex(index_keys, cuuid, index_raw)
			if idxRsp.Code == 200 {
				rsp.Code = 200
				rsp.Uuid = cuuid
			} else {
				rsp.Code = 500
				rsp.Msg = idxRsp.Msg
			}
		} else {
			rsp.Code = 500
			rsp.Msg = apiRsp.Msg
		}
	} else {
		var searchInfo SearchInfo
		if err := load_obj(c, body.Uuid, &searchInfo); err != nil {
			rsp_err(c, 500, err)
			return
		}

		searchInfo.Title = body.Title
		searchInfo.Score = body.Score
		searchInfo.Content = body.Content
		searchInfo.UpdateTime = tm

		data, err := obj_to_json(c, &searchInfo)
		if err != nil {
			rsp_err(c, 500, err)
			return
		}

		upRsp := wdb.UpdateObj(body.Uuid, data)
		if upRsp.Code == 200 {
			idxRsp := wdb.UpdateIndex(index_keys, index_keys, searchInfo.Uuid, index_raw)
			if idxRsp.Code != 200 {
				rsp_err(c, 500, idxRsp.Msg)
				return
			}
			rsp.Code = 200
			rsp.Uuid = body.Uuid
		} else {
			rsp.Code = 500
			rsp.Msg = upRsp.Msg
		}
	}
	c.JSON(http.StatusOK, rsp)
}

func GetSearchInfo(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}
	cuuid := c.Query("uuid")
	var info SearchInfo
	if err := load_obj(c, cuuid, &info); err != nil {
		rsp_err(c, 500, err)
		return
	}

	var info_rsp struct {
		Code int64      `json:"code"`
		Msg  string     `json:"msg"`
		Info SearchInfo `json:"info"`
	}
	info_rsp.Code = 200
	info_rsp.Msg = ""
	info_rsp.Info = info

	c.JSON(http.StatusOK, info_rsp)
}

func SearchList(c *gin.Context) {
	var body struct {
		O      string `json:"o"`
		Title  string `json:"title"`
		Score  string `json:"score"`
		Begin  string `json:"begin"`
		End    string `json:"end"`
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

	arr := []string{}
	if body.Title != "" {
		arr = append(arr, fmt.Sprintf("[\"title\",\"reg\",\"^.*%s.*$\"]", body.Title))
	}
	if body.Score != "" {
		arr = append(arr, fmt.Sprintf("[\"score\",\">=\",%s]", body.Score))
	}
	if body.Begin != "" && body.End != "" {
		ts, _ := time.Parse("2006-01-02 15:04", body.Begin)
		te, _ := time.Parse("2006-01-02 15:04", body.End)
		arr = append(arr, fmt.Sprintf("[\"updateTime\",\">=\",%d,\"<=\",%d]", ts.Unix(), te.Unix()))
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
	} else if body.Order == "sasc" {
		order = "score ASC"
	} else if body.Order == "sdesc" {
		order = "score DESC"
	}

	indexkey := fmt.Sprintf("my_search_index_%s", uid)
	listRsp := wdb.ListIndex(indexkey, condition, body.Offset, body.Limit, order)
	if listRsp.Code != 200 {
		rsp_err(c, 500, listRsp.Msg)
		return
	}

	blist := []SearchListInfo{}
	for _, item := range listRsp.List {
		fmt.Println(item)
		var cinfo SearchInfo
		if err := load_obj_by_str(c, item, &cinfo); err != nil {
			return
		}

		blist = append(blist, SearchListInfo{
			Uuid:  cinfo.Uuid,
			Title: cinfo.Title,
			Score: cinfo.Score,
			Time:  cinfo.UpdateTime,
		})
	}

	var orderListRsp struct {
		Code  uint64           `json:"code"`
		Msg   string           `json:"msg"`
		Total uint64           `json:"total"`
		List  []SearchListInfo `json:"list"`
	}
	orderListRsp.Code = 200
	orderListRsp.Msg = ""
	orderListRsp.Total = listRsp.Total
	orderListRsp.List = blist

	c.JSON(http.StatusOK, orderListRsp)
}

func SearchDel(c *gin.Context) {
	is_login, uid := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}
	cuuid := c.Query("uuid")
	index_key := []string{fmt.Sprintf("my_search_index_%s", uid)}
	apiRsp := wdb.DelObj(cuuid)
	if apiRsp.Code != 200 {
		idxRsp := wdb.DelIndex(index_key, cuuid)
		if idxRsp.Code != 200 {
			rsp_err(c, 500, apiRsp.Msg)
			return
		}
	}
	c.JSON(http.StatusOK, InfoRsp{
		Code: 200,
		Msg:  "",
		Uuid: cuuid,
	})
}

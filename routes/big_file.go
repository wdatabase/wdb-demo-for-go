package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type BigFileReq struct {
	O    string `json:"o,omitempty"`
	Key  string `json:"key,omitempty"`
	Path string `json:"path,omitempty"`
}

func BigUpload(c *gin.Context) {
	var req BigFileReq
	if err := c.BindJSON(&req); err != nil {
		rsp_err(c, 500, err)
		return
	}
	is_login, _ := auth(req.O)
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	apiRsp := wdb.UploadByPath(req.Path, req.Key, []string{})
	if apiRsp.Code != 200 {
		rsp_err(c, 500, apiRsp.Msg)
		return
	}

	c.JSON(http.StatusOK, InfoRsp{
		Code: 200,
		Msg:  "",
		Uuid: "",
	})
}

func BigDown(c *gin.Context) {
	var req BigFileReq
	if err := c.BindJSON(&req); err != nil {
		rsp_err(c, 500, err)
		return
	}
	is_login, _ := auth(req.O)
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	apiRsp := wdb.DownToPath(req.Path, req.Key)
	if apiRsp.Code != 200 {
		rsp_err(c, 500, apiRsp.Msg)
		return
	}

	c.JSON(http.StatusOK, InfoRsp{
		Code: 200,
		Msg:  "",
		Uuid: "",
	})
}

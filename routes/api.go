package routes

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type UserInfo struct {
	Uuid       string `json:"uuid"`
	User       string `json:"user"`
	Pwd        string `json:"pwd"`
	Mail       string `json:"mail"`
	CreateTime uint64 `json:"createTime"`
	UpdateTime uint64 `json:"updateTime"`
}

type UserRsp struct {
	Code uint64 `json:"code"`
	Msg  string `json:"msg"`
	Uid  string `json:"uid"`
	Time uint64 `json:"time"`
	Sign string `json:"sign"`
}

func ApiReg(c *gin.Context) {
	var body struct {
		User string `json:"user"`
		Pwd  string `json:"pwd"`
		Mail string `json:"mail"`
	}
	if err := c.BindJSON(&body); err != nil {
		return
	}

	cuuid := uuid.New().String()
	pwd := hash(fmt.Sprintf("%s_%s", body.User, body.Pwd))
	var user_info UserInfo
	user_info.Uuid = cuuid
	user_info.User = body.User
	user_info.Pwd = pwd
	user_info.Mail = body.Mail
	user_info.CreateTime = uint64(time.Now().Unix())
	user_info.UpdateTime = uint64(time.Now().Unix())

	data, err := obj_to_json(c, &user_info)
	if err != nil {
		return
	}

	key := fmt.Sprintf("user_%s", body.User)
	rsp := wdb.CreateObj(key, data, []string{"user_list"})
	if rsp.Code == 200 {
		c.JSON(http.StatusOK, UserRsp{
			Code: 200,
			Msg:  "",
			Uid:  cuuid,
			Time: 0,
			Sign: "",
		})
	} else {
		c.JSON(http.StatusOK, UserRsp{
			Code: 400,
			Msg:  fmt.Sprintf("%s", err),
			Uid:  "",
			Time: 0,
			Sign: "",
		})
	}
}

func ApiLogin(c *gin.Context) {
	var body struct {
		User string `json:"user"`
		Pwd  string `json:"pwd"`
	}
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusOK, UserRsp{
			Code: 400,
			Msg:  fmt.Sprintf("bind req %s", err),
			Uid:  "",
			Time: 0,
			Sign: "",
		})
		return
	}

	key := fmt.Sprintf("user_%s", body.User)
	var user_info UserInfo
	if err := load_obj(c, key, &user_info); err != nil {
		return
	}

	cpwd := hash(fmt.Sprintf("%s_%s", body.User, body.Pwd))
	if user_info.Pwd == cpwd {
		tm := uint64(time.Now().Unix())
		c.JSON(http.StatusOK, UserRsp{
			Code: 200,
			Msg:  "",
			Uid:  user_info.Uuid,
			Time: tm,
			Sign: sign(user_info.Uuid, tm),
		})
	} else {
		c.JSON(http.StatusOK, UserRsp{
			Code: 400,
			Msg:  "pwd err",
			Uid:  "",
			Time: 0,
			Sign: "",
		})
	}
}

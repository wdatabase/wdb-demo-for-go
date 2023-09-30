package routes

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ImgInfo struct {
	Uuid        string `json:"uuid"`
	Name        string `json:"name"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Size        uint64 `json:"size"`
	FileUuid    string `json:"fileUuid"`
	CreateTime  uint64 `json:"createTime"`
	UpdateTime  uint64 `json:"updateTime"`
}

type ImgRsp struct {
	Code uint64 `json:"code"`
	Msg  string `json:"msg"`
	Uuid string `json:"uuid"`
}

type ImgListInfo struct {
	Uuid        string `json:"uuid"`
	Name        string `json:"name"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Size        uint64 `json:"size"`
	FileUuid    string `json:"fileUuid"`
	Time        uint64 `json:"time"`
}

type ImgListRsp struct {
	Code  uint64        `json:"code"`
	Msg   string        `json:"msg"`
	Total uint64        `json:"total"`
	List  []ImgListInfo `json:"list"`
}

type ImgRaw struct {
	Raw string `json:"raw"`
}

func ImgPost(c *gin.Context) {
	is_login, uid := auth(c.Query("o"))
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
		return
	}

	imgRsp := wdb.CreateObj(fileUuid, rawdata, []string{})
	if imgRsp.Code != 200 {
		rsp_err(c, 500, imgRsp.Msg)
		return
	}

	ouuid := c.DefaultQuery("uuid", "")
	if ouuid == "" {
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

		data, err := obj_to_json(c, &img_info)
		if err != nil {
			rsp_err(c, 500, err)
			return
		}

		category := fmt.Sprintf("my_img_%s", uid)
		infoRsp := wdb.CreateObj(cuuid, data, []string{category})
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
	} else {
		var img_info ImgInfo
		if err := load_obj(c, ouuid, &img_info); err != nil {
			rsp_err(c, 500, err)
			return
		}

		img_info.FileName = file.Filename
		img_info.ContentType = contentType
		img_info.Size = uint64(file.Size)
		img_info.FileUuid = fileUuid
		img_info.UpdateTime = uint64(time.Now().Unix())

		data, err := obj_to_json(c, &img_info)
		if err != nil {
			rsp_err(c, 500, err)
			return
		}

		upRsp := wdb.UpdateObj(ouuid, data)
		if upRsp.Code == 200 {
			c.JSON(http.StatusOK, ImgRsp{
				Code: 200,
				Msg:  "",
				Uuid: ouuid,
			})
		} else {
			c.JSON(http.StatusOK, ImgRsp{
				Code: 400,
				Msg:  upRsp.Msg,
				Uuid: "",
			})
		}
	}
}

func GetImgInfo(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}
	cuuid := c.Query("uuid")
	var info ImgInfo
	if err := load_obj(c, cuuid, &info); err != nil {
		return
	}

	var info_rsp struct {
		Code int64   `json:"code"`
		Msg  string  `json:"msg"`
		Info ImgInfo `json:"info"`
	}
	info_rsp.Code = 200
	info_rsp.Msg = ""
	info_rsp.Info = info

	c.JSON(http.StatusOK, info_rsp)
}

func GetImgData(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}
	cuuid := c.Query("uuid")
	var info ImgInfo
	if err := load_obj(c, cuuid, &info); err != nil {
		return
	}

	var raw_info ImgRaw
	if err := load_obj(c, info.FileUuid, &raw_info); err != nil {
		return
	}

	raw, err := base64.StdEncoding.DecodeString(raw_info.Raw)
	if err != nil {
		rsp_err(c, 500, err)
		return
	}

	c.Data(http.StatusOK, info.ContentType, raw)
}

func ImgList(c *gin.Context) {
	is_login, uid := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	category := fmt.Sprintf("my_img_%s", uid)
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

	blist := []ImgListInfo{}
	for _, item := range listRsp.List {
		var cinfo ImgInfo
		if err := load_obj_by_str(c, item, &cinfo); err != nil {
			return
		}

		blist = append(blist, ImgListInfo{
			Uuid:        cinfo.Uuid,
			Name:        cinfo.Name,
			FileName:    cinfo.FileName,
			ContentType: cinfo.ContentType,
			Size:        cinfo.Size,
			FileUuid:    cinfo.FileUuid,
			Time:        cinfo.CreateTime,
		})
	}

	c.JSON(http.StatusOK, ImgListRsp{
		Code:  200,
		Msg:   "",
		Total: listRsp.Total,
		List:  blist,
	})
}

func ImgDel(c *gin.Context) {
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
	c.JSON(http.StatusOK, ImgRsp{
		Code: 200,
		Msg:  "",
		Uuid: cuuid,
	})
}

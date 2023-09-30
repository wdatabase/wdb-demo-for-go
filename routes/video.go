package routes

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type VideoInfo struct {
	Uuid        string `json:"uuid"`
	Name        string `json:"name"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Size        uint64 `json:"size"`
	FileUuid    string `json:"fileUuid"`
	CreateTime  uint64 `json:"createTime"`
	UpdateTime  uint64 `json:"updateTime"`
}

type VideoRsp struct {
	Code uint64 `json:"code"`
	Msg  string `json:"msg"`
	Uuid string `json:"uuid"`
}

type VideoListInfo struct {
	Uuid        string `json:"uuid"`
	Name        string `json:"name"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Size        uint64 `json:"size"`
	FileUuid    string `json:"fileUuid"`
	Time        uint64 `json:"time"`
}

type VideoListRsp struct {
	Code  uint64          `json:"code"`
	Msg   string          `json:"msg"`
	Total uint64          `json:"total"`
	List  []VideoListInfo `json:"list"`
}

func VideoPost(c *gin.Context) {
	is_login, uid := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	file, err := c.FormFile("video")
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

	fileUuid := uuid.New().String()
	fileRsp := wdb.CreateRawData(fileUuid, buf.Bytes(), []string{})
	if fileRsp.Code != 200 {
		rsp_err(c, 500, fileRsp.Msg)
		return
	}

	ouuid := c.DefaultQuery("uuid", "")
	if ouuid == "" {
		cuuid := uuid.New().String()

		var video_info VideoInfo
		video_info.Uuid = cuuid
		video_info.Name = "video"
		video_info.FileName = file.Filename
		video_info.ContentType = contentType
		video_info.Size = uint64(file.Size)
		video_info.FileUuid = fileUuid
		video_info.CreateTime = uint64(time.Now().Unix())
		video_info.UpdateTime = uint64(time.Now().Unix())

		data, err := obj_to_json(c, &video_info)
		if err != nil {
			return
		}

		category := fmt.Sprintf("my_video_%s", uid)
		videoRsp := wdb.CreateObj(cuuid, data, []string{category})
		if videoRsp.Code == 200 {
			c.JSON(http.StatusOK, VideoRsp{
				Code: 200,
				Msg:  "",
				Uuid: cuuid,
			})
		} else {
			c.JSON(http.StatusOK, VideoRsp{
				Code: 400,
				Msg:  fmt.Sprintf("%s", err),
				Uuid: "",
			})
		}
	} else {
		var video_info VideoInfo
		if err := load_obj(c, ouuid, &video_info); err != nil {
			return
		}

		video_info.FileName = file.Filename
		video_info.ContentType = contentType
		video_info.Size = uint64(file.Size)
		video_info.FileUuid = fileUuid
		video_info.UpdateTime = uint64(time.Now().Unix())

		data, err := obj_to_json(c, &video_info)
		if err != nil {
			return
		}

		upRsp := wdb.UpdateObj(ouuid, data)
		if upRsp.Code == 200 {
			c.JSON(http.StatusOK, VideoRsp{
				Code: 200,
				Msg:  "",
				Uuid: ouuid,
			})
		} else {
			c.JSON(http.StatusOK, VideoRsp{
				Code: 400,
				Msg:  upRsp.Msg,
				Uuid: "",
			})
		}
	}
}

func GetVideoInfo(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}
	cuuid := c.Query("uuid")
	var info VideoInfo
	if err := load_obj(c, cuuid, &info); err != nil {
		return
	}

	var info_rsp struct {
		Code uint64    `json:"code"`
		Msg  string    `json:"msg"`
		Info VideoInfo `json:"info"`
	}
	info_rsp.Code = 200
	info_rsp.Msg = ""
	info_rsp.Info = info

	c.JSON(http.StatusOK, info_rsp)
}

func GetVideoData(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}
	cuuid := c.Query("uuid")
	var info VideoInfo
	if err := load_obj(c, cuuid, &info); err != nil {
		return
	}

	ctype, data := "", []byte{}
	stat := http.StatusOK
	if hrange, is_ok := c.Request.Header["Range"]; is_ok {
		stat = 206
		cr := strings.Trim(hrange[0], "bytes=")
		arr := strings.Split(cr, "-")
		start, err := strconv.ParseUint(arr[0], 10, 64)
		if err != nil {
			rsp_err(c, 500, err)
			return
		}
		end, err := strconv.ParseUint(arr[1], 10, 64)
		if err != nil {
			end = start + 1024*1024
		}
		rawRsp := wdb.GetRangeData(info.FileUuid, start, end)
		if rawRsp.Code != 200 {
			rsp_err(c, 500, rawRsp.Msg)
			return
		}
		if end > rawRsp.Size {
			end = rawRsp.Size - 1
		}
		ctype = info.ContentType
		data = rawRsp.Raw
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, rawRsp.Size))
	} else {
		rawRsp := wdb.GetRawData(info.FileUuid)
		if rawRsp.Code != 200 {
			rsp_err(c, 500, rawRsp.Msg)
			return
		}
		ctype = info.ContentType
		data = rawRsp.Raw
		c.Header("Accept-Range", "bytes")
	}

	c.Header("Last-Modified", fmt.Sprintf("%v", time.Unix(int64(info.UpdateTime), 0)))
	c.Header("Etag", info.FileUuid)

	c.Data(stat, ctype, data)
}

func VideoList(c *gin.Context) {
	is_login, uid := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	category := fmt.Sprintf("my_video_%s", uid)
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

	blist := []VideoListInfo{}
	for _, item := range listRsp.List {
		var cinfo VideoInfo
		if err := load_obj_by_str(c, item, &cinfo); err != nil {
			return
		}

		blist = append(blist, VideoListInfo{
			Uuid:        cinfo.Uuid,
			Name:        cinfo.Name,
			FileName:    cinfo.FileName,
			ContentType: cinfo.ContentType,
			Size:        cinfo.Size,
			FileUuid:    cinfo.FileUuid,
			Time:        cinfo.CreateTime,
		})
	}

	c.JSON(http.StatusOK, VideoListRsp{
		Code:  200,
		Msg:   "",
		Total: listRsp.Total,
		List:  blist,
	})
}

func VideoDel(c *gin.Context) {
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

	c.JSON(http.StatusOK, VideoRsp{
		Code: 200,
		Msg:  "",
		Uuid: cuuid,
	})
}

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

type FileInfo struct {
	Uuid        string `json:"uuid"`
	Name        string `json:"name"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Size        uint64 `json:"size"`
	FileUuid    string `json:"fileUuid"`
	CreateTime  uint64 `json:"createTime"`
	UpdateTime  uint64 `json:"updateTime"`
}

type FileRsp struct {
	Code uint64 `json:"code"`
	Msg  string `json:"msg"`
	Uuid string `json:"uuid"`
}

type FileListInfo struct {
	Uuid        string `json:"uuid"`
	Name        string `json:"name"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	Size        uint64 `json:"size"`
	FileUuid    string `json:"fileUuid"`
	Time        uint64 `json:"time"`
}

type FileListRsp struct {
	Code  uint64         `json:"code"`
	Msg   string         `json:"msg"`
	Total uint64         `json:"total"`
	List  []FileListInfo `json:"list"`
}

func FilePost(c *gin.Context) {
	is_login, uid := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	file, err := c.FormFile("file")
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
	apiRsp := wdb.CreateRawData(fileUuid, buf.Bytes(), []string{})
	if apiRsp.Code != 200 {
		rsp_err(c, 500, apiRsp.Msg)
		return
	}

	ouuid := c.DefaultQuery("uuid", "")
	if ouuid == "" {
		cuuid := uuid.New().String()

		var video_info FileInfo
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

		category := fmt.Sprintf("my_file_%s", uid)
		infoRsp := wdb.CreateObj(cuuid, data, []string{category})
		if infoRsp.Code == 200 {
			c.JSON(http.StatusOK, FileRsp{
				Code: 200,
				Msg:  "",
				Uuid: cuuid,
			})
		} else {
			c.JSON(http.StatusOK, FileRsp{
				Code: 400,
				Msg:  infoRsp.Msg,
				Uuid: "",
			})
		}
	} else {
		var video_info FileInfo
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
			c.JSON(http.StatusOK, FileRsp{
				Code: 200,
				Msg:  "",
				Uuid: ouuid,
			})
		} else {
			c.JSON(http.StatusOK, FileRsp{
				Code: 400,
				Msg:  upRsp.Msg,
				Uuid: "",
			})
		}
	}
}

func GetFileInfo(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}
	cuuid := c.Query("uuid")
	var info FileInfo
	if err := load_obj(c, cuuid, &info); err != nil {
		return
	}

	var info_rsp struct {
		Code int64    `json:"code"`
		Msg  string   `json:"msg"`
		Info FileInfo `json:"info"`
	}
	info_rsp.Code = 200
	info_rsp.Msg = ""
	info_rsp.Info = info

	c.JSON(http.StatusOK, info_rsp)
}

func GetFileData(c *gin.Context) {
	is_login, _ := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}
	cuuid := c.Query("uuid")
	var info FileInfo
	if err := load_obj(c, cuuid, &info); err != nil {
		return
	}

	data := []byte{}
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
			end = 102400000000
		}
		rgRsp := wdb.GetRangeData(info.FileUuid, start, end)
		if rgRsp.Code != 200 {
			rsp_err(c, 500, rgRsp.Msg)
			return
		}
		if end > rgRsp.Size {
			end = rgRsp.Size
		}
		data = rgRsp.Raw
		c.Header("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, rgRsp.Size))
	} else {
		rawRsp := wdb.GetRawData(info.FileUuid)
		if rawRsp.Code != 200 {
			rsp_err(c, 500, rawRsp.Msg)
			return
		}
		data = rawRsp.Raw
		c.Header("Accept-Range", "bytes")
	}

	c.Header("Last-Modified", fmt.Sprintf("%v", time.Unix(int64(info.UpdateTime), 0)))
	c.Header("Etag", info.FileUuid)

	c.Header("Content-Disposition", fmt.Sprintf("attachment;filename=%s", info.FileName))
	c.Data(stat, "application/octet-stream", data)
}

func FileList(c *gin.Context) {
	is_login, uid := auth(c.Query("o"))
	if !is_login {
		rsp_err(c, 403, "auth fail")
		return
	}

	category := fmt.Sprintf("my_file_%s", uid)
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

	blist := []FileListInfo{}
	for _, item := range listRsp.List {
		var cinfo FileInfo
		if err := load_obj_by_str(c, item, &cinfo); err != nil {
			return
		}

		blist = append(blist, FileListInfo{
			Uuid:        cinfo.Uuid,
			Name:        cinfo.Name,
			FileName:    cinfo.FileName,
			ContentType: cinfo.ContentType,
			Size:        cinfo.Size,
			FileUuid:    cinfo.FileUuid,
			Time:        cinfo.CreateTime,
		})
	}

	c.JSON(http.StatusOK, FileListRsp{
		Code:  200,
		Msg:   "",
		Total: listRsp.Total,
		List:  blist,
	})
}

func FileDel(c *gin.Context) {
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

	c.JSON(http.StatusOK, FileRsp{
		Code: 200,
		Msg:  "",
		Uuid: cuuid,
	})
}

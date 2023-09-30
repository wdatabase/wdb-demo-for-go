package routes

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/gin-gonic/gin"
)

func StringToBytes(s string) []byte {
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

func BytesToString(b []byte) string {
	return unsafe.String(&b[0], len(b))
}

func auth(o string) (bool, string) {
	slice := strings.Split(o, "_")
	if len(slice) == 3 {
		uid := slice[0]
		otm, err := strconv.ParseInt(slice[1], 10, 64)
		if err != nil {
			return false, ""
		}
		tm := uint64(otm)
		ctm := uint64(time.Now().Unix())
		if ctm < tm || ctm-tm > 6000 {
			return false, ""
		}
		if slice[2] == sign(uid, tm) {
			return true, uid
		} else {
			return false, ""
		}
	}
	return false, ""
}

func hash(text string) string {
	sum := sha256.Sum256([]byte(text))
	return fmt.Sprintf("%x", sum)
}

func sign(uid string, tm uint64) string {
	return hash(fmt.Sprintf("E43FGDsefsf_33ss%s%d", uid, tm))
}

func rsp_err(c *gin.Context, code uint64, err interface{}) {
	switch i := err.(type) {
	case string:
		c.JSON(http.StatusOK, InfoRsp{Code: code, Msg: err.(string), Uuid: ""})
		return
	case error:
		c.JSON(http.StatusOK, InfoRsp{Code: code, Msg: fmt.Sprintf("%s", err), Uuid: ""})
		return
	default:
		_ = i
		return
	}
}

func rsp_ok(c *gin.Context, data string) {
	c.JSON(http.StatusOK, InfoRsp{Code: 200, Msg: "", Uuid: data})
}

func load_obj(c *gin.Context, uuid string, obj interface{}) error {
	getRsp := wdb.GetObj(uuid)
	if getRsp.Code != 200 {
		rsp_err(c, 500, getRsp.Msg)
		return errors.New("api get obj err")
	}
	if err := json.Unmarshal([]byte(getRsp.Data), &obj); err != nil {
		rsp_err(c, 500, err)
		return errors.New("json decode err")
	}
	return nil
}

func load_trans_obj(c *gin.Context, tsid string, key string, obj interface{}) error {
	getRsp := wdb.TransGet(tsid, key)
	if getRsp.Code != 200 {
		rsp_err(c, 500, getRsp.Msg)
		return errors.New("api get obj err")
	}
	if err := json.Unmarshal([]byte(getRsp.Data), &obj); err != nil {
		rsp_err(c, 500, err)
		return errors.New("json decode err")
	}
	return nil
}

func load_obj_by_str(c *gin.Context, data string, obj interface{}) error {
	if err := json.Unmarshal([]byte(data), &obj); err != nil {
		rsp_err(c, 500, err)
		return errors.New("json decode err")
	}
	return nil
}

func obj_to_json(c *gin.Context, obj interface{}) (string, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		rsp_err(c, 500, err)
		return "", errors.New("json encode err")
	}
	return BytesToString(data), nil
}

func contains[T comparable](elems []T, v T) (bool, int) {
	for i, s := range elems {
		if v == s {
			return true, i
		}
	}
	return false, 0
}

func remove[T comparable](slice []T, s int) []T {
	return append(slice[:s], slice[s+1:]...)
}

func float64cmp(num1 float64, num2 float64) int {
	if num1-num2 > 0.000001 {
		return 1
	} else if num1-num2 < -0.000001 {
		return -1
	} else {
		return 0
	}
}

func fixfloat64(val float64) float64 {
	return math.Round(val*100.0) / 100.0
}

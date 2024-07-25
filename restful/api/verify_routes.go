package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/ontio/layer2deploy/common"
	"github.com/ontio/layer2deploy/core"
	"github.com/ontio/ontology/common/log"
	"net/http"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

func RoutesApi(parent *gin.Engine) {
	apiRoute := parent.Group("/api")
	apiRoute.GET("/verifyhash/:hash", VerifyHash)
	apiRoute.POST("/enablesendservice/:enableSendService", EnableSendService)
	apiRoute.POST("/storehash", StoreHash)
}

func StoreHash(c *gin.Context) {
	param := []string{}
	err := c.BindJSON(&param)
	if err != nil {
		c.JSON(404, gin.H{
			"message": err.Error(),
		})
		return
	}
	result := []map[string]interface{}{}
	for _, item := range param {
		log.Infof("item:%s", item)
		time := uint64(time.Now().UnixNano() / int64(time.Millisecond))
		item = strings.ReplaceAll(item, "txtime", strconv.Itoa(int(time)))
		bytes := []byte(item)
		hash := sha256.Sum256(bytes)
		hashStr := hex.EncodeToString(hash[:])
		txHash, err := core.DefVerifyService.StoreHashCore(hash[:])
		if err != nil {
			c.JSON(400, gin.H{
				"message": err.Error(),
			})
			return
		}
		log.Infof("hashStr:%s", hashStr)
		var out = map[string]interface{}{}
		err = json.Unmarshal(bytes, &out)
		if err != nil {
			c.JSON(400, gin.H{
				"message": err.Error(),
			})
			return
		}
		out["sha256"] = hashStr
		out["txHash"] = txHash
		result = append(result, out)
	}

	//for _, item := range param {
	//	if d, ok := item.(map[string]interface{}); ok {
	//		d["timestamp"] = uint64(time.Now().UnixNano() / int64(time.Millisecond))
	//		bytes, err := json.Marshal(d)
	//		if err != nil {
	//			log.Debugf("Marshal: N.0 %s", err)
	//			c.JSON(http.StatusOK, common.VerifyResponse{
	//				Code:    common.HASHDATAERROR,
	//				Message: common.CodeMessageMap[common.HASHDATAERROR] + fmt.Sprintf("%v", err),
	//			})
	//			return
	//		}
	//		hash := sha256.Sum256(bytes)
	//
	//		hashStr := hex.EncodeToString(hash[:])
	//		txHash, err := core.DefVerifyService.StoreHashCore(hash[:])
	//		if err != nil {
	//			c.JSON(400, gin.H{
	//				"message": err.Error(),
	//			})
	//			return
	//		}
	//
	//		d["sha256"] = hashStr
	//		d["txHash"] = txHash
	//	}
	//}

	c.JSON(http.StatusOK, common.VerifyResponse{
		Code:    common.SUCCESS,
		Message: common.CodeMessageMap[common.SUCCESS],
		Result:  result,
	})
}

func EnableSendService(c *gin.Context) {
	enable := c.Param("enableSendService")
	if enable == "true" {
		log.Infof("EnableSendService true")
		atomic.StoreUint32(&core.DefSendService.Enabled, 1)
	} else {
		log.Infof("EnableSendService false")
		atomic.StoreUint32(&core.DefSendService.Enabled, 0)
	}
}

func VerifyHash(c *gin.Context) {
	hash := c.Param("hash")

	if len(hash) != sha256.Size*2 {
		c.JSON(http.StatusOK, common.VerifyResponse{
			Code:    common.HASHLENERROR,
			Message: common.CodeMessageMap[common.HASHLENERROR],
		})
		return
	}

	log.Debugf("VerifyHash: Y.0 %s", hash)

	_, err := hex.DecodeString(hash)
	if err != nil {
		log.Debugf("VerifyHash: N.0 %s", err)
		c.JSON(http.StatusOK, common.VerifyResponse{
			Code:    common.HASHDATAERROR,
			Message: common.CodeMessageMap[common.HASHDATAERROR] + fmt.Sprintf("%v", err),
		})
		return
	}

	result, err := core.DefVerifyService.VerifyHashCore(hash)
	if err != nil {
		log.Debugf("VerifyHash: N.1 %s", err)
		c.JSON(http.StatusOK, common.VerifyResponse{
			Code:    common.SERVERERROR,
			Message: common.CodeMessageMap[common.SERVERERROR] + fmt.Sprintf(" %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, common.VerifyResponse{
		Code:    common.SUCCESS,
		Message: common.CodeMessageMap[common.SUCCESS],
		Result:  result,
	})
}

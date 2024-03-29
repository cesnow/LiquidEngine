package LiquidSDK

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"github.com/cesnow/liquid-engine/liquid-db"
	"github.com/cesnow/liquid-engine/logger"
	"os"
	"strconv"
	"time"
)

var liquidKeyTemplate = "LiquidServer_%s"

func (server *LiquidServer) SetCodeName(codename string) {
	if codename == "" {
		logger.SysLog.Errorf("[Engine] Codename is empty !!!")
		os.Exit(100)
	}
	logger.SysLog.Infof("[Engine] Codename -> %s", codename)
	server.CodeName = codename
}

func (server *LiquidServer) GetKeyStatic() string {
	if time.Now().Sub(server.LiquidKeyUpdate).Minutes() > 60 {
		return server.GetKey()
	}
	return server.LiquidKey
}

func (server *LiquidServer) GetKey() string {
	RedisLiquidKeyName := fmt.Sprintf(liquidKeyTemplate, server.CodeName)
	LiquidKey, GetKeyErr := LiquidDB.GetInstance().GetCacheDb().Get(RedisLiquidKeyName)
	if GetKeyErr != nil {
		server.GenerateKey()
		server.LiquidKeyUpdate = time.Now()
		return server.LiquidKey
	}
	ReceivedLiquidKey := string(LiquidKey)
	if ReceivedLiquidKey != server.LiquidKey {
		server.LiquidKey = ReceivedLiquidKey
	}
	server.LiquidKeyUpdate = time.Now()
	return server.LiquidKey
}

func (server *LiquidServer) InitCodenameKey() {
	RedisLiquidKeyName := fmt.Sprintf(liquidKeyTemplate, server.CodeName)
	LiquidKey, GetKeyErr := LiquidDB.GetInstance().GetCacheDb().Get(RedisLiquidKeyName)
	if GetKeyErr != nil {
		server.GenerateKey()
	} else {
		server.LiquidKey = string(LiquidKey)
	}
	logger.SysLog.Infof("[Engine] System Key -> %s", server.LiquidKey)
}

func (server *LiquidServer) SetTokenExpireTime(seconds int) {
	if seconds < int(time.Hour.Seconds()) {
		logger.SysLog.Errorf("[Engine] Token Expire Time Must Over 3600 Seconds !!!")
		return
	}
	server.TokenExpireTime = seconds
}

func (server *LiquidServer) GenerateKey() {
	conJunctions := "-LiquidSDK-"
	md5Generate := md5.New()
	var keyOriginConcat bytes.Buffer
	keyOriginConcat.Write([]byte(server.CodeName))
	keyOriginConcat.Write([]byte(conJunctions))
	keyOriginConcat.Write([]byte(strconv.FormatInt(time.Now().Unix(), 10)))
	md5Generate.Write(keyOriginConcat.Bytes())
	RedisLiquidKeyName := fmt.Sprintf(liquidKeyTemplate, server.CodeName)
	LiquidKey := hex.EncodeToString(md5Generate.Sum(nil))
	SaveKey2RedisErr := LiquidDB.GetInstance().GetCacheDb().SetString(RedisLiquidKeyName, LiquidKey, -1)
	if SaveKey2RedisErr != nil {
		logger.SysLog.Errorf("[System] Save System Key To Redis Error, %s", SaveKey2RedisErr)
	}
	server.LiquidKey = LiquidKey
}

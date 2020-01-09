package GameFoundation

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/cesnow/LiquidEngine/Logger"
	"github.com/cesnow/LiquidEngine/Middlewares"
	"github.com/cesnow/LiquidEngine/Modules/LiquidRpc"
	"github.com/cesnow/LiquidEngine/Modules/LiquidSDK"
	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"
	"net/http"
)

func RouteCommand(c *gin.Context) {

	var command *LiquidSDK.CmdCommand
	_ = json.Unmarshal(c.MustGet("CommandData").([]byte), &command)
	Logger.SysLog.Debugf("[CMD][Command] %+v", command)

	result := &LiquidSDK.CmdCommandResponse{
		CmdData: nil,
		CmdSn:   nil,
	}

	if command.LiquidId == nil || command.LiquidToken == nil {
		c.String(http.StatusOK, Middlewares.GetLiquidResult(result))
		return
	}

	if command.Platform == nil {
		platformMain := "main"
		command.Platform = &platformMain
	}

	tokenKey := fmt.Sprintf("token_%s_%s", *command.LiquidId, *command.Platform)
	authToken, authTokenErr := LiquidSDK.GetServer().GetCacheDb().Get(tokenKey)
	liquidToken := string(authToken)

	if authTokenErr != nil || liquidToken != *command.LiquidToken {
		c.String(http.StatusOK, Middlewares.GetLiquidResult(result))
		return
	}

	// TODO: Server Maintain States (Unsupported)

	setUserTokenErr := LiquidSDK.GetServer().GetCacheDb().SetString(tokenKey, liquidToken, 1800)
	if setUserTokenErr != nil {
		Logger.SysLog.Warnf("[CMD][Command] Refresh User Token Failed, %s", setUserTokenErr)
	}

	// gRpc Routing Mode Checking
	if rpcResult, rpcErr := CommandGRpc(command); rpcErr != nil {
		gameFeature := LiquidSDK.GetServer().GetGameFeature(*command.CmdId)
		if gameFeature == nil {
			c.String(http.StatusOK, Middlewares.GetLiquidResult(result))
			return
		}
		runCommandData := gameFeature.RunCommand(command)
		result.CmdData = runCommandData
	} else {
		var CmdResult interface{}
		_ = json.Unmarshal(rpcResult, &CmdResult)
		result.CmdData = CmdResult
	}

	result.CmdSn = command.CmdSn
	c.String(http.StatusOK, Middlewares.GetLiquidResult(result))
}

func CommandGRpc(command *LiquidSDK.CmdCommand) ([]byte, error) {
	conn, err := grpc.Dial("localhost:9999", grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	c := LiquidRpc.NewGameAdapterClient(conn)
	marshalCmdData, _ := json.Marshal(command.CmdData)
	r, err := c.Command(context.Background(), &LiquidRpc.CmdCommand{
		UserID:    *command.LiquidId,
		UserToken: *command.LiquidToken,
		Platform:  *command.Platform,
		CmdId:     *command.CmdId,
		CmdSn:     uint64(*command.CmdSn),
		CmdName:   *command.CmdName,
		CmdData:   marshalCmdData,
	})
	if err != nil {
		return nil, err
	}
	return r.CmdData, nil
}

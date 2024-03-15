package foundation

import (
	"encoding/json"
	LiquidModels "github.com/cesnow/liquid-engine/liquid-models"
	LiquidSDK "github.com/cesnow/liquid-engine/liquid-sdk"

	"github.com/cesnow/liquid-engine/internal/middlewares"
	"github.com/cesnow/liquid-engine/logger"
	"github.com/gin-gonic/gin"
	"net/http"
)

func RouteBind(c *gin.Context) {

	var command *LiquidSDK.CmdBind
	_ = json.Unmarshal(c.MustGet("CommandData").([]byte), &command)
	logger.SysLog.Debugf("[CMD][Bind] %+v", command)

	result := &LiquidSDK.CmdAccountResponse{}

	liquidUser := LiquidModels.FindLiquidUserByAutoId(command.AutoId, command.InviteCode)

	if liquidUser == nil {
		c.String(http.StatusForbidden, middlewares.GetLiquidResult(gin.H{
			"code":  1601,
			"error": "user is not found",
		}))
		return
	}

	_, bindErr := LiquidModels.BindLiquidUser(command.AutoId, command.FromId, command.FromType, command.FromToken)

	if bindErr != nil {
		logger.SysLog.Warnf("[CMD][Bind] Duplicate Bind Failed, %s", bindErr)
		c.String(http.StatusOK, middlewares.GetLiquidResult(gin.H{
			"code":  1602,
			"error": "account is already bind",
		}))
		return
	}

	result.AutoId = &liquidUser.AutoId
	result.InviteCode = &liquidUser.InviteCode
	c.String(http.StatusOK, middlewares.GetLiquidResult(result))

}

package api

import (
	"encoding/json"
	"strings"

	apiv1 "github.com/alireza0/s-ui/api/v1"
	"github.com/alireza0/s-ui/service"
	"github.com/alireza0/s-ui/util/common"

	"github.com/gin-gonic/gin"
)

type APIHandler struct {
	ApiService
	apiv2  *APIv2Handler
	logSvc service.ApiLogService
}

func NewAPIHandler(g *gin.RouterGroup, a2 *APIv2Handler) {
	a := &APIHandler{
		apiv2: a2,
	}
	a.initRouter(g)
}

func (a *APIHandler) initRouter(g *gin.RouterGroup) {
	g.Use(func(c *gin.Context) {
		path := c.Request.URL.Path
		if !strings.HasSuffix(path, "login") && !strings.HasSuffix(path, "logout") {
			checkLogin(c)
		}
	})
	g.POST("/:postAction", a.postHandler)
	g.GET("/:getAction", a.getHandler)
}

func (a *APIHandler) postHandler(c *gin.Context) {
	loginUser := GetLoginUser(c)
	action := c.Param("postAction")

	switch action {
	case "login":
		a.ApiService.Login(c)
	case "changePass":
		a.ApiService.ChangePass(c)
	case "save":
		a.ApiService.Save(c, loginUser)
	case "restartApp":
		a.ApiService.RestartApp(c)
	case "restartSb":
		a.ApiService.RestartSb(c)
	case "linkConvert":
		a.ApiService.LinkConvert(c)
	case "subConvert":
		a.ApiService.SubConvert(c)
	case "importdb":
		a.ApiService.ImportDb(c)
	case "addToken":
		a.ApiService.AddToken(c)
		a.apiv2.ReloadTokens()
		_ = apiv1.Reload()
	case "deleteToken":
		a.ApiService.DeleteToken(c)
		a.apiv2.ReloadTokens()
		_ = apiv1.Reload()
	case "setting":
		a.ApiService.UpdateSettingsAPI(c)
	case "clearApiLogs":
		err := a.logSvc.Clear()
		jsonMsg(c, "clearApiLogs", err)
	case "cfListZones":
		a.ApiService.CFListZones(c)
	case "cfUpsertA":
		a.ApiService.CFUpsertA(c)
	case "cfIssueTls":
		a.ApiService.CFIssueTLS(c)
	default:
		jsonMsg(c, "failed", common.NewError("unknown action: ", action))
	}
}

func (a *APIHandler) getHandler(c *gin.Context) {
	action := c.Param("getAction")

	switch action {
	case "logout":
		a.ApiService.Logout(c)
	case "load":
		a.ApiService.LoadData(c)
	case "inbounds", "outbounds", "endpoints", "services", "tls", "clients", "config":
		err := a.ApiService.LoadPartialData(c, []string{action})
		if err != nil {
			jsonMsg(c, action, err)
		}
		return
	case "users":
		a.ApiService.GetUsers(c)
	case "settings":
		a.ApiService.GetSettings(c)
	case "stats":
		a.ApiService.GetStats(c)
	case "status":
		a.ApiService.GetStatus(c)
	case "onlines":
		a.ApiService.GetOnlines(c)
	case "logs":
		a.ApiService.GetLogs(c)
	case "changes":
		a.ApiService.CheckChanges(c)
	case "keypairs":
		a.ApiService.GetKeypairs(c)
	case "getdb":
		a.ApiService.GetDb(c)
	case "tokens":
		a.ApiService.GetTokens(c)
	case "singbox-config":
		a.ApiService.GetSingboxConfig(c)
	case "checkOutbound":
		a.ApiService.GetCheckOutbound(c)
	case "apiLogs":
		a.handleApiLogs(c)
	default:
		jsonMsg(c, "failed", common.NewError("unknown action: ", action))
	}
}

func (a *APIHandler) handleApiLogs(c *gin.Context) {
	method := c.Query("method")
	path := c.Query("path")
	username := c.Query("username")
	var since, until int64
	json.Unmarshal([]byte(c.DefaultQuery("since", "0")), &since)
	json.Unmarshal([]byte(c.DefaultQuery("until", "0")), &until)
	var limit, offset int
	json.Unmarshal([]byte(c.DefaultQuery("limit", "200")), &limit)
	json.Unmarshal([]byte(c.DefaultQuery("offset", "0")), &offset)

	logs, total, err := a.logSvc.List(method, path, username, since, until, limit, offset)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, gin.H{"logs": logs, "total": total}, nil)
}

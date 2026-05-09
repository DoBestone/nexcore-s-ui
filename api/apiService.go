package api

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/logger"
	"github.com/alireza0/s-ui/service"
	"github.com/alireza0/s-ui/util"

	"github.com/gin-gonic/gin"
)

type ApiService struct {
	service.SettingService
	service.UserService
	service.ConfigService
	service.ClientService
	service.TlsService
	service.InboundService
	service.OutboundService
	service.EndpointService
	service.ServicesService
	service.PanelService
	service.StatsService
	service.ServerService
	service.CloudflareService
}

func (a *ApiService) LoadData(c *gin.Context) {
	data, err := a.getData(c)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, data, nil)
}

func (a *ApiService) getData(c *gin.Context) (interface{}, error) {
	data := make(map[string]interface{}, 0)
	lu := c.Query("lu")
	isUpdated, err := a.ConfigService.CheckChanges(lu)
	if err != nil {
		return "", err
	}
	onlines, err := a.StatsService.GetOnlines()

	sysInfo := a.ServerService.GetSingboxInfo()
	if sysInfo["running"] == false {
		logs := a.ServerService.GetLogs("1", "debug")
		if len(logs) > 0 {
			data["lastLog"] = logs[0]
		}
	}

	if err != nil {
		return "", err
	}
	if isUpdated {
		config, err := a.SettingService.GetConfig()
		if err != nil {
			return "", err
		}
		clients, err := a.ClientService.GetAll()
		if err != nil {
			return "", err
		}
		tlsConfigs, err := a.TlsService.GetAll()
		if err != nil {
			return "", err
		}
		inbounds, err := a.InboundService.GetAll()
		if err != nil {
			return "", err
		}
		outbounds, err := a.OutboundService.GetAll()
		if err != nil {
			return "", err
		}
		endpoints, err := a.EndpointService.GetAll()
		if err != nil {
			return "", err
		}
		services, err := a.ServicesService.GetAll()
		if err != nil {
			return "", err
		}
		subURI, err := a.SettingService.GetFinalSubURI(getHostname(c))
		if err != nil {
			return "", err
		}
		trafficAge, err := a.SettingService.GetTrafficAge()
		if err != nil {
			return "", err
		}
		data["config"] = json.RawMessage(config)
		data["clients"] = clients
		data["tls"] = tlsConfigs
		data["inbounds"] = inbounds
		data["outbounds"] = outbounds
		data["endpoints"] = endpoints
		data["services"] = services
		data["subURI"] = subURI
		data["enableTraffic"] = trafficAge > 0
		data["onlines"] = onlines
	} else {
		data["onlines"] = onlines
	}

	return data, nil
}

func (a *ApiService) LoadPartialData(c *gin.Context, objs []string) error {
	data := make(map[string]interface{}, 0)
	id := c.Query("id")

	for _, obj := range objs {
		switch obj {
		case "inbounds":
			inbounds, err := a.InboundService.Get(id)
			if err != nil {
				return err
			}
			data[obj] = inbounds
		case "outbounds":
			outbounds, err := a.OutboundService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = outbounds
		case "endpoints":
			endpoints, err := a.EndpointService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = endpoints
		case "services":
			services, err := a.ServicesService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = services
		case "tls":
			tlsConfigs, err := a.TlsService.GetAll()
			if err != nil {
				return err
			}
			data[obj] = tlsConfigs
		case "clients":
			clients, err := a.ClientService.Get(id)
			if err != nil {
				return err
			}
			data[obj] = clients
		case "config":
			config, err := a.SettingService.GetConfig()
			if err != nil {
				return err
			}
			data[obj] = json.RawMessage(config)
		case "settings":
			settings, err := a.SettingService.GetAllSetting()
			if err != nil {
				return err
			}
			data[obj] = settings
		}
	}

	jsonObj(c, data, nil)
	return nil
}

func (a *ApiService) GetUsers(c *gin.Context) {
	users, err := a.UserService.GetUsers()
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, *users, nil)
}

func (a *ApiService) GetSettings(c *gin.Context) {
	data, err := a.SettingService.GetAllSetting()
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, data, err)
}

func (a *ApiService) GetStats(c *gin.Context) {
	resource := c.Query("resource")
	tag := c.Query("tag")
	limit, err := strconv.Atoi(c.Query("limit"))
	if err != nil {
		limit = 100
	}
	data, err := a.StatsService.GetStats(resource, tag, limit)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, data, err)
}

func (a *ApiService) GetStatus(c *gin.Context) {
	request := c.Query("r")
	result := a.ServerService.GetStatus(request)
	jsonObj(c, result, nil)
}

func (a *ApiService) GetOnlines(c *gin.Context) {
	onlines, err := a.StatsService.GetOnlines()
	jsonObj(c, onlines, err)
}

func (a *ApiService) GetLogs(c *gin.Context) {
	count := c.Query("c")
	level := c.Query("l")
	logs := a.ServerService.GetLogs(count, level)
	jsonObj(c, logs, nil)
}

func (a *ApiService) CheckChanges(c *gin.Context) {
	actor := c.Query("a")
	chngKey := c.Query("k")
	count := c.Query("c")
	changes := a.ConfigService.GetChanges(actor, chngKey, count)
	jsonObj(c, changes, nil)
}

func (a *ApiService) GetKeypairs(c *gin.Context) {
	kType := c.Query("k")
	options := c.Query("o")
	keypair := a.ServerService.GenKeypair(kType, options)
	jsonObj(c, keypair, nil)
}

func (a *ApiService) GetDb(c *gin.Context) {
	exclude := c.Query("exclude")
	db, err := database.GetDb(exclude)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	c.Header("Content-Type", "application/octet-stream")
	c.Header("Content-Disposition", "attachment; filename=s-ui_"+time.Now().Format("20060102-150405")+".db")
	c.Writer.Write(db)
}

func (a *ApiService) postActions(c *gin.Context) (string, json.RawMessage, error) {
	var data map[string]json.RawMessage
	err := c.ShouldBind(&data)
	if err != nil {
		return "", nil, err
	}
	return string(data["action"]), data["data"], nil
}

func (a *ApiService) Login(c *gin.Context) {
	remoteIP := getRemoteIp(c)
	loginUser, err := a.UserService.Login(c.Request.FormValue("user"), c.Request.FormValue("pass"), remoteIP)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}

	sessionMaxAge, err := a.SettingService.GetSessionMaxAge()
	if err != nil {
		logger.Infof("Unable to get session's max age from DB")
	}

	err = SetLoginUser(c, loginUser, sessionMaxAge)
	if err == nil {
		logger.Info("user ", loginUser, " login success")
	} else {
		logger.Warning("login failed: ", err)
	}

	jsonMsg(c, "", nil)
}

func (a *ApiService) ChangePass(c *gin.Context) {
	id := c.Request.FormValue("id")
	oldPass := c.Request.FormValue("oldPass")
	newUsername := c.Request.FormValue("newUsername")
	newPass := c.Request.FormValue("newPass")
	err := a.UserService.ChangePass(id, oldPass, newUsername, newPass)
	if err == nil {
		logger.Info("change user credentials success")
		jsonMsg(c, "save", nil)
	} else {
		logger.Warning("change user credentials failed:", err)
		jsonMsg(c, "", err)
	}
}

func (a *ApiService) Save(c *gin.Context, loginUser string) {
	hostname := getHostname(c)
	obj := c.Request.FormValue("object")
	act := c.Request.FormValue("action")
	data := c.Request.FormValue("data")
	initUsers := c.Request.FormValue("initUsers")
	objs, err := a.ConfigService.Save(obj, act, json.RawMessage(data), initUsers, loginUser, hostname)
	if err != nil {
		jsonMsg(c, "save", err)
		return
	}
	err = a.LoadPartialData(c, objs)
	if err != nil {
		jsonMsg(c, obj, err)
	}
}

func (a *ApiService) RestartApp(c *gin.Context) {
	err := a.PanelService.RestartPanel(3)
	jsonMsg(c, "restartApp", err)
}

func (a *ApiService) RestartSb(c *gin.Context) {
	err := a.ConfigService.RestartCore()
	jsonMsg(c, "restartSb", err)
}

func (a *ApiService) LinkConvert(c *gin.Context) {
	link := c.Request.FormValue("link")
	result, _, err := util.GetOutbound(link, 0)
	jsonObj(c, result, err)
}

func (a *ApiService) SubConvert(c *gin.Context) {
	link := c.Request.FormValue("link")
	result, err := util.GetExternalSub(link)
	jsonObj(c, result, err)
}

func (a *ApiService) ImportDb(c *gin.Context) {
	file, _, err := c.Request.FormFile("db")
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	defer file.Close()
	err = database.ImportDB(file)
	jsonMsg(c, "", err)
}

func (a *ApiService) Logout(c *gin.Context) {
	loginUser := GetLoginUser(c)
	if loginUser != "" {
		logger.Infof("user %s logout", loginUser)
	}
	ClearSession(c)
	jsonMsg(c, "", nil)
}

func (a *ApiService) LoadTokens() ([]byte, error) {
	return a.UserService.LoadTokens()
}

func (a *ApiService) GetTokens(c *gin.Context) {
	loginUser := GetLoginUser(c)
	tokens, err := a.UserService.GetUserTokens(loginUser)
	jsonObj(c, tokens, err)
}

func (a *ApiService) AddToken(c *gin.Context) {
	loginUser := GetLoginUser(c)
	expiry := c.Request.FormValue("expiry")
	expiryInt, err := strconv.ParseInt(expiry, 10, 64)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	desc := c.Request.FormValue("desc")
	token, err := a.UserService.AddToken(loginUser, expiryInt, desc)
	jsonObj(c, token, err)
}

func (a *ApiService) DeleteToken(c *gin.Context) {
	tokenId := c.Request.FormValue("id")
	err := a.UserService.DeleteToken(tokenId)
	jsonMsg(c, "", err)
}

// addTokenForUser - v2 入口:用调用方持有的 token 推断出 username,而不是 session。
// 表单字段同 v1 (expiry / desc),返回新 token 字符串。
func (a *ApiService) addTokenForUser(c *gin.Context, username string) {
	if username == "" {
		jsonMsg(c, "", errEmpty("username"))
		return
	}
	expiry := c.Request.FormValue("expiry")
	expiryInt, err := strconv.ParseInt(expiry, 10, 64)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	desc := c.Request.FormValue("desc")
	token, err := a.UserService.AddToken(username, expiryInt, desc)
	jsonObj(c, token, err)
}

func (a *ApiService) getTokensForUser(c *gin.Context, username string) {
	if username == "" {
		jsonMsg(c, "", errEmpty("username"))
		return
	}
	tokens, err := a.UserService.GetUserTokens(username)
	jsonObj(c, tokens, err)
}

// UpdateSettingsAPI - v2 入口:与 v1 的 setting CLI / sui setting 等价的写接口。
// 接受 form 字段 port / path / subPort / subPath / listen / domain / subListen / subDomain。
// 这里只覆盖最常用的几个,够脚本化部署用;前端继续走 save 走 ConfigService.Save 改 settings 表。
func (a *ApiService) UpdateSettingsAPI(c *gin.Context) {
	type req struct {
		Port      *int    `json:"port" form:"port"`
		Path      *string `json:"path" form:"path"`
		SubPort   *int    `json:"subPort" form:"subPort"`
		SubPath   *string `json:"subPath" form:"subPath"`
	}
	var r req
	_ = c.ShouldBind(&r)

	if r.Port != nil && *r.Port > 0 {
		if err := a.SettingService.SetPort(*r.Port); err != nil {
			jsonMsg(c, "setting", err)
			return
		}
	}
	if r.Path != nil && *r.Path != "" {
		if err := a.SettingService.SetWebPath(*r.Path); err != nil {
			jsonMsg(c, "setting", err)
			return
		}
	}
	if r.SubPort != nil && *r.SubPort > 0 {
		if err := a.SettingService.SetSubPort(*r.SubPort); err != nil {
			jsonMsg(c, "setting", err)
			return
		}
	}
	if r.SubPath != nil && *r.SubPath != "" {
		if err := a.SettingService.SetSubPath(*r.SubPath); err != nil {
			jsonMsg(c, "setting", err)
			return
		}
	}
	jsonMsg(c, "setting", nil)
}

func errEmpty(field string) error {
	type emptyErr struct{ s string }
	return &simpleErr{msg: field + " can not be empty"}
}

type simpleErr struct{ msg string }

func (e *simpleErr) Error() string { return e.msg }

// ---------- Cloudflare ----------

func (a *ApiService) CFListZones(c *gin.Context) {
	token := c.Request.FormValue("token")
	if token == "" {
		token = c.Query("token")
	}
	if err := a.CloudflareService.VerifyToken(token); err != nil {
		jsonMsg(c, "", err)
		return
	}
	zones, err := a.CloudflareService.ListZones(token)
	jsonObj(c, zones, err)
}

func (a *ApiService) CFUpsertA(c *gin.Context) {
	type req struct {
		Token   string `form:"token" json:"token"`
		ZoneId  string `form:"zoneId" json:"zoneId"`
		Name    string `form:"name" json:"name"`
		Random  bool   `form:"random" json:"random"`
		Prefix  string `form:"prefix" json:"prefix"`
		IP      string `form:"ip" json:"ip"`
		Proxied bool   `form:"proxied" json:"proxied"`
	}
	var r req
	if err := c.ShouldBind(&r); err != nil {
		jsonMsg(c, "", err)
		return
	}
	if r.Token == "" || r.ZoneId == "" || r.IP == "" {
		jsonMsg(c, "", errEmpty("token / zoneId / ip"))
		return
	}
	subname := r.Name
	if r.Random {
		subname = a.CloudflareService.RandomSubdomain(r.Prefix)
	}
	fqdn, recId, err := a.CloudflareService.UpsertARecord(r.Token, r.ZoneId, subname, r.IP, r.Proxied)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, gin.H{"fqdn": fqdn, "recordId": recId}, nil)
}

func (a *ApiService) CFIssueTLS(c *gin.Context) {
	type req struct {
		Name    string `form:"name" json:"name"`
		Fqdn    string `form:"fqdn" json:"fqdn"`
		Email   string `form:"email" json:"email"`
		Token   string `form:"token" json:"token"`
		DataDir string `form:"dataDir" json:"dataDir"`
	}
	var r req
	if err := c.ShouldBind(&r); err != nil {
		jsonMsg(c, "", err)
		return
	}
	if r.Fqdn == "" || r.Email == "" || r.Token == "" {
		jsonMsg(c, "", errEmpty("fqdn / email / token"))
		return
	}
	id, err := a.CloudflareService.IssueTLS(r.Name, r.Fqdn, r.Email, r.Token, r.DataDir)
	if err != nil {
		jsonMsg(c, "", err)
		return
	}
	jsonObj(c, gin.H{"id": id, "fqdn": r.Fqdn}, nil)
}

func (a *ApiService) GetSingboxConfig(c *gin.Context) {
	rawConfig, err := a.ConfigService.GetConfig("")
	if err != nil {
		c.Status(400)
		c.Writer.WriteString(err.Error())
		return
	}
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=config_"+time.Now().Format("20060102-150405")+".json")
	c.Writer.Write(*rawConfig)
}

func (a *ApiService) GetCheckOutbound(c *gin.Context) {
	tag := c.Query("tag")
	link := c.Query("link")
	result := a.ConfigService.CheckOutbound(tag, link)
	jsonObj(c, result, nil)
}

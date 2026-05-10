package v1

// 客户端 CRUD —— x-ui 兼容面(per-inbound 操作)
//
// 路由(register 在 v1.go):
//   POST   /inbounds/:id/clients               增加,body 是 client 对象
//   PUT    /inbounds/:id/clients/:identifier   更新(identifier = client.name 或 numeric id)
//   DELETE /inbounds/:id/clients/:identifier   从该入站移除(若移完没绑定入站则整条删)
//
// 数据模型差异:s-ui Client 是全局表,Client.Inbounds 是 []uint;x-ui Client 嵌
// 在 inbound.settings.clients。这里把 :id 当成"该 client 必须挂在的入站",
// POST 时强制并入 client.Inbounds,DELETE 时从 client.Inbounds 移除一项。
//
// identifier 优先按 client.name(全局唯一)匹配,再 fallback numeric id(s-ui 内部主键)。

import (
	"encoding/json"
	"errors"
	"strconv"

	"github.com/alireza0/s-ui/database"
	"github.com/alireza0/s-ui/database/model"

	"github.com/gin-gonic/gin"
)

// findClientByIdentifier:identifier 先按 name 找,再按 numeric id 找。
// 这俩是 s-ui 唯一的客户端寻址维度(s-ui 没有 email 字段,name 即 email 等价物)。
func findClientByIdentifier(identifier string) (*model.Client, error) {
	db := database.GetDB()
	var cli model.Client
	if err := db.Where("name = ?", identifier).First(&cli).Error; err == nil {
		return &cli, nil
	}
	if id, perr := strconv.Atoi(identifier); perr == nil {
		if err := db.Where("id = ?", uint(id)).First(&cli).Error; err == nil {
			return &cli, nil
		}
	}
	return nil, errors.New("client not found: " + identifier)
}

// ensureInboundInList 把 inboundId 加进 inboundsRaw([]uint JSON)。已在则不变。
// 返回新的 raw JSON 和"是否实际新增了"的布尔。
func ensureInboundInList(inboundsRaw json.RawMessage, inboundId uint) (json.RawMessage, bool, error) {
	var ids []uint
	if len(inboundsRaw) > 0 {
		if err := json.Unmarshal(inboundsRaw, &ids); err != nil {
			return nil, false, err
		}
	}
	for _, id := range ids {
		if id == inboundId {
			return inboundsRaw, false, nil
		}
	}
	ids = append(ids, inboundId)
	out, err := json.Marshal(ids)
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

// removeInboundFromList 从 []uint JSON 里去掉 inboundId。返回新 raw、是否真删了、剩余长度。
func removeInboundFromList(inboundsRaw json.RawMessage, inboundId uint) (json.RawMessage, bool, int, error) {
	var ids []uint
	if err := json.Unmarshal(inboundsRaw, &ids); err != nil {
		return nil, false, 0, err
	}
	out := make([]uint, 0, len(ids))
	removed := false
	for _, id := range ids {
		if id == inboundId {
			removed = true
			continue
		}
		out = append(out, id)
	}
	raw, err := json.Marshal(out)
	if err != nil {
		return nil, false, 0, err
	}
	return raw, removed, len(out), nil
}

// ---------- handlers ----------

// POST /inbounds/:id/clients
// body 是 client 对象({name, enable, config, inbounds?, volume?, expiry?, ...})
// :id 入站会被强制并入 client.Inbounds — 这是 x-ui per-inbound 调用语义。
func (a *Controller) createClientByInbound(c *gin.Context) {
	idStr := c.Param("id")
	inboundId, err := strconv.Atoi(idStr)
	if err != nil {
		BadRequest(c, "invalid_id", "invalid inbound id: "+idStr)
		return
	}
	if _, err := a.findInboundByID(idStr); err != nil {
		NotFound(c, "inbound_not_found", err.Error())
		return
	}
	body, err := c.GetRawData()
	if err != nil {
		BadRequest(c, "invalid_body", err.Error())
		return
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		BadRequest(c, "invalid_body", err.Error())
		return
	}
	// 强制把 :id 加入 client.inbounds —— 即使 body 没传也补上
	merged, _, err := ensureInboundInList(raw["inbounds"], uint(inboundId))
	if err != nil {
		BadRequest(c, "invalid_inbounds", err.Error())
		return
	}
	raw["inbounds"] = merged
	patched, err := json.Marshal(raw)
	if err != nil {
		Internal(c, "marshal_failed", err)
		return
	}

	username := c.GetString("api_token_user")
	if username == "" {
		username = "api"
	}
	objs, err := a.configSvc.Save("clients", "new", json.RawMessage(patched), "", username, getPanelHost(c))
	if err != nil {
		BadRequest(c, mapSaveErr(err, "save_failed"), err.Error())
		return
	}
	OK(c, gin.H{"object": "clients", "action": "new", "affected": objs})
}

// PUT /inbounds/:id/clients/:identifier
// body 是 client 字段补丁。merge 到现存 client(只覆盖 body 里出现的字段),
// 并保证 :id 入站仍在 client.Inbounds。
func (a *Controller) updateClientByInbound(c *gin.Context) {
	idStr := c.Param("id")
	identifier := c.Param("identifier")
	inboundId, err := strconv.Atoi(idStr)
	if err != nil {
		BadRequest(c, "invalid_id", "invalid inbound id: "+idStr)
		return
	}
	cli, err := findClientByIdentifier(identifier)
	if err != nil {
		NotFound(c, "client_not_found", err.Error())
		return
	}

	body, err := c.GetRawData()
	if err != nil {
		BadRequest(c, "invalid_body", err.Error())
		return
	}
	var patch map[string]json.RawMessage
	if err := json.Unmarshal(body, &patch); err != nil {
		BadRequest(c, "invalid_body", err.Error())
		return
	}

	// 起点:把 cli marshal 回去做基底,再把 patch 覆盖上
	baseBytes, err := json.Marshal(cli)
	if err != nil {
		Internal(c, "marshal_failed", err)
		return
	}
	var base map[string]json.RawMessage
	if err := json.Unmarshal(baseBytes, &base); err != nil {
		Internal(c, "unmarshal_failed", err)
		return
	}
	for k, v := range patch {
		base[k] = v
	}
	// 强制 id 字段 = 现存 id(避免 patch 里漏掉 / 改成 0 触发 ConfigService 走 new 分支)
	idJson, _ := json.Marshal(cli.Id)
	base["id"] = idJson

	// 保证 :id 入站还在 inbounds 里
	merged, _, err := ensureInboundInList(base["inbounds"], uint(inboundId))
	if err != nil {
		BadRequest(c, "invalid_inbounds", err.Error())
		return
	}
	base["inbounds"] = merged

	patched, err := json.Marshal(base)
	if err != nil {
		Internal(c, "marshal_failed", err)
		return
	}

	username := c.GetString("api_token_user")
	if username == "" {
		username = "api"
	}
	objs, err := a.configSvc.Save("clients", "edit", json.RawMessage(patched), "", username, getPanelHost(c))
	if err != nil {
		BadRequest(c, mapSaveErr(err, "save_failed"), err.Error())
		return
	}
	OK(c, gin.H{"object": "clients", "action": "edit", "affected": objs})
}

// DELETE /inbounds/:id/clients/:identifier
// 从 client.Inbounds 移除 :id;若 client 此后再无任何入站绑定,则整条删除。
// 这条语义跟 x-ui 一致 — 在 x-ui 里 client 嵌在 inbound.settings.clients,
// 从某 inbound 删掉 = 该 client 在该 inbound 不存在;别的 inbound 不受影响。
func (a *Controller) deleteClientByInbound(c *gin.Context) {
	idStr := c.Param("id")
	identifier := c.Param("identifier")
	inboundId, err := strconv.Atoi(idStr)
	if err != nil {
		BadRequest(c, "invalid_id", "invalid inbound id: "+idStr)
		return
	}
	cli, err := findClientByIdentifier(identifier)
	if err != nil {
		NotFound(c, "client_not_found", err.Error())
		return
	}

	newRaw, removed, remaining, err := removeInboundFromList(cli.Inbounds, uint(inboundId))
	if err != nil {
		BadRequest(c, "invalid_inbounds", err.Error())
		return
	}
	if !removed {
		NotFound(c, "client_not_in_inbound", "client "+identifier+" not bound to inbound "+idStr)
		return
	}

	username := c.GetString("api_token_user")
	if username == "" {
		username = "api"
	}

	// 全脱钩 → 整条删,免留尸体
	if remaining == 0 {
		idJson, _ := json.Marshal(cli.Id)
		objs, err := a.configSvc.Save("clients", "del", json.RawMessage(idJson), "", username, getPanelHost(c))
		if err != nil {
			BadRequest(c, mapSaveErr(err, "delete_failed"), err.Error())
			return
		}
		OK(c, gin.H{"object": "clients", "action": "del", "affected": objs})
		return
	}

	// 仍绑定其它入站 → edit 改 inbounds 字段
	cli.Inbounds = newRaw
	patched, err := json.Marshal(cli)
	if err != nil {
		Internal(c, "marshal_failed", err)
		return
	}
	objs, err := a.configSvc.Save("clients", "edit", json.RawMessage(patched), "", username, getPanelHost(c))
	if err != nil {
		BadRequest(c, mapSaveErr(err, "save_failed"), err.Error())
		return
	}
	OK(c, gin.H{"object": "clients", "action": "edit", "affected": objs})
}

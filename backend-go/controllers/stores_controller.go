package controllers

import (
	"net/http"
	"strings"

	"github.com/beego/beego/v2/client/orm"

	"ai-multi-platform-release/backend-go/models"
	"ai-multi-platform-release/backend-go/services"
)

type StoresController struct {
	BaseController
}

type storeCreateRequest struct {
	Name    string `json:"name"`
	Code    string `json:"code"`
	Address string `json:"address"`
	Contact string `json:"contact"`
	Phone   string `json:"phone"`
	Status  string `json:"status"`
}

type storeUpdateRequest struct {
	Name    *string `json:"name"`
	Code    *string `json:"code"`
	Address *string `json:"address"`
	Contact *string `json:"contact"`
	Phone   *string `json:"phone"`
	Status  *string `json:"status"`
}

// List GET /api/stores/
func (c *StoresController) List() {
	if !c.CheckPermission("stores:read") {
		return
	}
	page, pageSize := c.ParsePagination()
	qs := services.GetOrm().QueryTable(new(models.Store))
	if keyword := strings.TrimSpace(c.GetQuery("keyword")); keyword != "" {
		cond := orm.NewCondition().
			Or("name__icontains", keyword).
			Or("code__icontains", keyword).
			Or("address__icontains", keyword)
		qs = qs.SetCond(cond)
	}
	if status := c.GetQuery("status"); status != "" {
		qs = qs.Filter("status", status)
	}
	total, err := qs.Count()
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "查询门店失败")
		return
	}
	var stores []models.Store
	if _, err := qs.OrderBy("-created_at").Limit(pageSize, (page-1)*pageSize).All(&stores); err != nil {
		c.WriteError(http.StatusInternalServerError, "查询门店失败")
		return
	}
	c.OK(map[string]interface{}{
		"items":     stores,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// ListAll GET /api/stores/all
func (c *StoresController) ListAll() {
	if !c.CheckPermission("stores:read") {
		return
	}
	var stores []models.Store
	if _, err := services.GetOrm().QueryTable(new(models.Store)).
		Filter("status", "active").
		OrderBy("name").
		All(&stores); err != nil {
		c.WriteError(http.StatusInternalServerError, "查询门店失败")
		return
	}
	c.OK(stores)
}

// Create POST /api/stores/
func (c *StoresController) Create() {
	if !c.CheckPermission("stores:create:write") {
		return
	}
	var req storeCreateRequest
	if err := c.ParseBody(&req); err != nil || strings.TrimSpace(req.Name) == "" {
		c.WriteError(http.StatusBadRequest, "门店名称不能为空")
		return
	}
	status := req.Status
	if status == "" {
		status = "active"
	}
	store := &models.Store{
		ID:      newID(),
		Name:    req.Name,
		Code:    req.Code,
		Address: req.Address,
		Contact: req.Contact,
		Phone:   req.Phone,
		Status:  status,
	}
	if _, err := services.GetOrm().Insert(store); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建门店失败")
		return
	}
	c.Created(store)
}

func findStore(storeID string) (*models.Store, error) {
	var store models.Store
	err := services.GetOrm().QueryTable(new(models.Store)).
		Filter("id", storeID).
		One(&store)
	if err != nil {
		return nil, err
	}
	return &store, nil
}

// Get GET /api/stores/:store_id
func (c *StoresController) Get() {
	if !c.CheckPermission("stores:read") {
		return
	}
	store, err := findStore(c.GetPathParam("store_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "门店不存在")
		return
	}
	c.OK(store)
}

// Update PUT /api/stores/:store_id
func (c *StoresController) Update() {
	if !c.CheckPermission("stores:update:write") {
		return
	}
	store, err := findStore(c.GetPathParam("store_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "门店不存在")
		return
	}
	var req storeUpdateRequest
	if err := c.ParseBody(&req); err != nil {
		c.WriteError(http.StatusBadRequest, "请求体格式错误")
		return
	}
	if req.Name != nil {
		store.Name = *req.Name
	}
	if req.Code != nil {
		store.Code = *req.Code
	}
	if req.Address != nil {
		store.Address = *req.Address
	}
	if req.Contact != nil {
		store.Contact = *req.Contact
	}
	if req.Phone != nil {
		store.Phone = *req.Phone
	}
	if req.Status != nil {
		store.Status = *req.Status
	}
	if _, err := services.GetOrm().Update(store); err != nil {
		c.WriteError(http.StatusInternalServerError, "更新门店失败")
		return
	}
	c.OK(store)
}

// Delete DELETE /api/stores/:store_id
func (c *StoresController) Delete() {
	if !c.CheckPermission("stores:delete:write") {
		return
	}
	store, err := findStore(c.GetPathParam("store_id"))
	if err != nil {
		c.WriteError(http.StatusNotFound, "门店不存在")
		return
	}
	hasInspection := services.GetOrm().QueryTable(new(models.Inspection)).
		Filter("store_id", store.ID).
		Exist()
	if hasInspection {
		c.WriteError(http.StatusConflict, "该门店存在巡店记录，无法删除")
		return
	}
	if _, err := services.GetOrm().Delete(store); err != nil {
		c.WriteError(http.StatusInternalServerError, "删除门店失败")
		return
	}
	c.WriteNoContent()
}

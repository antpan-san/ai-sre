package handlers

import (
	"strings"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/models"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminListCommercialProducts lists sellable skill/pack products.
func AdminListCommercialProducts(c *gin.Context) {
	rows, err := services.ListCommercialProducts()
	if err != nil {
		logger.Error("AdminListCommercialProducts: %v", err)
		response.ServerError(c, "查询商品包失败")
		return
	}
	response.OK(c, gin.H{"products": rows, "policy_rev": services.CommercialPolicyRev()})
}

// AdminListCommercialBindings lists product-to-tree bindings.
func AdminListCommercialBindings(c *gin.Context) {
	rows, err := services.ListProductBindings(c.Query("product_key"))
	if err != nil {
		logger.Error("AdminListCommercialBindings: %v", err)
		response.ServerError(c, "查询绑定失败")
		return
	}
	response.OK(c, gin.H{"bindings": rows})
}

type adminCreateBindingRequest struct {
	ProductKey    string `json:"product_key" binding:"required"`
	NodePath      string `json:"node_path" binding:"required"`
	SkillKey      string `json:"skill_key"`
	CapabilityKey string `json:"capability_key"`
	PackKey       string `json:"pack_key"`
	GrantScope    string `json:"grant_scope"`
}

// AdminCreateCommercialBinding adds a product binding.
func AdminCreateCommercialBinding(c *gin.Context) {
	var req adminCreateBindingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效参数: "+err.Error())
		return
	}
	b := models.SkillProductNodeBinding{
		ProductKey:    req.ProductKey,
		NodePath:      req.NodePath,
		SkillKey:      strings.TrimSpace(req.SkillKey),
		CapabilityKey: strings.TrimSpace(req.CapabilityKey),
		PackKey:       strings.TrimSpace(req.PackKey),
		GrantScope:    strings.TrimSpace(req.GrantScope),
	}
	if err := services.CreateProductBinding(b); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	response.OK(c, gin.H{"created": true, "policy_rev": services.CommercialPolicyRev()})
}

// AdminDeleteCommercialBinding removes a binding by id.
func AdminDeleteCommercialBinding(c *gin.Context) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		response.BadRequest(c, "无效 ID")
		return
	}
	if err := services.DeleteProductBinding(id); err != nil {
		response.ServerError(c, "删除绑定失败")
		return
	}
	response.OK(c, gin.H{"deleted": true, "policy_rev": services.CommercialPolicyRev()})
}

// AdminCommercialProductsForNode returns which products cover a node path.
func AdminCommercialProductsForNode(c *gin.Context) {
	path := strings.TrimSpace(c.Query("node_path"))
	if path == "" {
		response.BadRequest(c, "node_path 必填")
		return
	}
	response.OK(c, gin.H{
		"node_path":   path,
		"product_keys": services.ProductsForNodePath(path),
	})
}

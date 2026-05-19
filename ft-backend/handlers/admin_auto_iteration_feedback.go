package handlers

import (
	"strconv"

	"ft-backend/common/logger"
	"ft-backend/common/response"
	"ft-backend/services"

	"github.com/gin-gonic/gin"
)

// AdminListAutoIterationFeedbacks lists recent CLI feedback analyze records.
func AdminListAutoIterationFeedbacks(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	rows, err := services.ListAutoIterationFeedbacks(limit)
	if err != nil {
		logger.Error("AdminListAutoIterationFeedbacks: %v", err)
		response.ServerError(c, "查询反馈记录失败")
		return
	}
	response.OK(c, gin.H{"feedbacks": rows, "limit": limit})
}

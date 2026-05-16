package handlers

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"ft-backend/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminStreamAutoIterationEvents streams iteration events via SSE (super_admin JWT only).
func AdminStreamAutoIterationEvents(c *gin.Context) {
	id, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.JSON(400, gin.H{"code": 400, "msg": "无效任务 ID"})
		return
	}
	if _, err := services.GetAutoIteration(id); err != nil {
		c.JSON(404, gin.H{"code": 404, "msg": "任务不存在"})
		return
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	c.Stream(func(w io.Writer) bool {
		events, err := services.ListAutoIterationEvents(id, uuid.Nil, 100)
		if err != nil {
			return false
		}
		for _, ev := range events {
			payload, _ := json.Marshal(ev)
			_, _ = io.WriteString(w, "event: log\n")
			_, _ = io.WriteString(w, "data: ")
			_, _ = w.Write(payload)
			_, _ = io.WriteString(w, "\n\n")
		}
		_, _ = io.WriteString(w, ": keepalive\n\n")
		time.Sleep(2 * time.Second)
		return true
	})
}

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
// Query: after_id — only send events newer than this UUID (incremental).
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
	afterID := uuid.Nil
	if s := strings.TrimSpace(c.Query("after_id")); s != "" {
		if parsed, err := uuid.Parse(s); err == nil {
			afterID = parsed
		}
	}
	c.Writer.Header().Set("Content-Type", "text/event-stream")
	c.Writer.Header().Set("Cache-Control", "no-cache")
	c.Writer.Header().Set("Connection", "keep-alive")
	c.Writer.Header().Set("X-Accel-Buffering", "no")
	lastStatus := ""
	lastEventID := afterID
	c.Stream(func(w io.Writer) bool {
		row, err := services.GetAutoIteration(id)
		if err != nil {
			return false
		}
		if row.Status != lastStatus {
			payload, _ := json.Marshal(row)
			_, _ = io.WriteString(w, "event: status\n")
			_, _ = io.WriteString(w, "data: ")
			_, _ = w.Write(payload)
			_, _ = io.WriteString(w, "\n\n")
			lastStatus = row.Status
		}
		events, err := services.ListAutoIterationEvents(id, lastEventID, 50)
		if err != nil {
			return false
		}
		for _, ev := range events {
			payload, _ := json.Marshal(ev)
			_, _ = io.WriteString(w, "event: log\n")
			_, _ = io.WriteString(w, "data: ")
			_, _ = w.Write(payload)
			_, _ = io.WriteString(w, "\n\n")
			lastEventID = ev.ID
		}
		_, _ = io.WriteString(w, ": keepalive\n\n")
		time.Sleep(2 * time.Second)
		return true
	})
}

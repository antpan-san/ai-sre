package handlers

import (
	"net/http"
	"os"
	"strconv"
	"strings"

	"ft-backend/common/config"
	"ft-backend/database"
	"ft-backend/models"
	"ft-backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// UploadFile handles file uploads.
func UploadFile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}

	uid := models.UserIDFromContext(userID)

	cfg := c.MustGet("config").(*config.Config)

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, cfg.File.MaxFileSize)

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "获取文件失败", "error": err.Error()})
		return
	}
	defer file.Close()

	if !utils.ValidateFileExtension(header.Filename, cfg.File.AllowedFormats) {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "文件格式不允许"})
		return
	}

	fileHash, err := utils.CalculateFileHash(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "计算文件哈希失败"})
		return
	}

	uniqueFilename := utils.GenerateUniqueFilename(header.Filename)

	filePath, fileSize, err := utils.SaveUploadedFile(file, header, cfg.File.UploadDir, uniqueFilename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存文件失败", "error": err.Error()})
		return
	}

	newFile := models.File{
		UserID:       uid,
		Filename:     uniqueFilename,
		OriginalName: header.Filename,
		Size:         fileSize,
		Path:         filePath,
		MimeType:     header.Header.Get("Content-Type"),
		Extension:    utils.GetFileExtension(header.Filename),
		Hash:         fileHash,
		Status:       "available",
		Visibility:   "private",
	}

	if err := database.DB.Create(&newFile).Error; err != nil {
		utils.DeleteFile(filePath)
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "保存文件元数据失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code": 201,
		"msg":  "文件上传成功",
		"data": gin.H{"file": newFile},
	})
}

// DownloadFile handles file downloads (public access).
func DownloadFile(c *gin.Context) {
	fileID, err := uuid.Parse(c.Param("file_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的文件ID"})
		return
	}

	var file models.File
	if err := database.DB.Where("id = ? AND status = ?", fileID, "available").First(&file).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "文件不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "数据库错误"})
		}
		return
	}

	if _, err := os.Stat(file.Path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "服务器上文件不存在"})
		return
	}

	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Transfer-Encoding", "binary")
	c.Header("Content-Disposition", "attachment; filename="+file.OriginalName)
	c.Header("Content-Type", file.MimeType)
	c.Header("Content-Length", strconv.FormatInt(file.Size, 10))

	c.File(file.Path)

	go func() {
		database.DB.Model(&file).UpdateColumn("download_count", gorm.Expr("download_count + ?", 1))
	}()
}

// ListFiles returns a paginated list of the authenticated user's files.
func ListFiles(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}

	uid := models.UserIDFromContext(userID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	offset := (page - 1) * pageSize

	var files []models.File
	var total int64

	db := database.DB.Model(&models.File{}).Where("user_id = ? AND deleted_at IS NULL", uid)

	if status := c.Query("status"); status != "" {
		db = db.Where("status = ?", status)
	}
	if visibility := c.Query("visibility"); visibility != "" {
		db = db.Where("visibility = ?", visibility)
	}

	db.Count(&total)

	if err := db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取文件列表失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取文件列表成功",
		"data": gin.H{"list": files, "total": total},
	})
}

// GetFileInfo returns details of a single file owned by the authenticated user.
func GetFileInfo(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}

	uid := models.UserIDFromContext(userID)

	fileID, err := uuid.Parse(c.Param("file_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的文件ID"})
		return
	}

	var file models.File
	if err := database.DB.Where("id = ? AND user_id = ? AND deleted_at IS NULL", fileID, uid).First(&file).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "文件不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "数据库错误"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "获取文件信息成功", "data": file})
}

// DeleteFile soft-deletes a file owned by the authenticated user.
func DeleteFile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}

	uid := models.UserIDFromContext(userID)

	fileID, err := uuid.Parse(c.Param("file_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的文件ID"})
		return
	}

	var file models.File
	if err := database.DB.Where("id = ? AND user_id = ? AND deleted_at IS NULL", fileID, uid).First(&file).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "文件不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "数据库错误"})
		}
		return
	}

	if err := database.DB.Delete(&file).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "删除文件失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 200, "msg": "文件删除成功"})
}

// ShareFile creates a share link for a file.
func ShareFile(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}

	uid := models.UserIDFromContext(userID)

	fileID, err := uuid.Parse(c.Param("file_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 400, "msg": "无效的文件ID"})
		return
	}

	var file models.File
	if err := database.DB.Where("id = ? AND user_id = ? AND deleted_at IS NULL", fileID, uid).First(&file).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 404, "msg": "文件不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "数据库错误"})
		}
		return
	}

	shareKey := utils.GenerateUniqueFilename(file.Filename)

	share := models.Share{
		FileID:    file.ID,
		ShareKey:  shareKey,
		ExpiresAt: database.DB.NowFunc().AddDate(0, 0, 7),
	}

	if err := database.DB.Create(&share).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "创建分享链接失败", "error": err.Error()})
		return
	}

	shareURL := strings.Join([]string{c.Request.Host, "/api/files/download/", shareKey}, "")

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "文件分享成功",
		"data": gin.H{
			"share_key":  shareKey,
			"share_url":  shareURL,
			"expires_at": share.ExpiresAt,
		},
	})
}

// GetSharedFiles returns a paginated list of shared files for the authenticated user.
func GetSharedFiles(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "未授权"})
		return
	}

	uid := models.UserIDFromContext(userID)

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))
	offset := (page - 1) * pageSize

	var shares []models.Share
	var total int64

	db := database.DB.Model(&models.Share{}).Preload("File").
		Joins("JOIN files ON shares.file_id = files.id").
		Where("files.user_id = ? AND shares.expires_at > ?", uid, database.DB.NowFunc())

	db.Count(&total)

	if err := db.Order("shares.created_at DESC").Offset(offset).Limit(pageSize).Find(&shares).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 500, "msg": "获取共享文件列表失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "获取共享文件列表成功",
		"data": gin.H{"list": shares, "total": total},
	})
}

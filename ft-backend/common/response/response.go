package response

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// R is the unified API response structure.
// All endpoints MUST return this format for consistency.
type R struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// PageResult wraps a paginated list.
type PageResult struct {
	List  interface{} `json:"list"`
	Total int64       `json:"total"`
}

// ---- Success Responses ----

func OK(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, R{Code: 200, Msg: "success", Data: data})
}

func OKMsg(c *gin.Context, msg string) {
	c.JSON(http.StatusOK, R{Code: 200, Msg: msg})
}

func OKPage(c *gin.Context, list interface{}, total int64) {
	c.JSON(http.StatusOK, R{Code: 200, Msg: "success", Data: PageResult{List: list, Total: total}})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, R{Code: 201, Msg: "success", Data: data})
}

// ---- Error Responses ----

func BadRequest(c *gin.Context, msg string) {
	c.JSON(http.StatusBadRequest, R{Code: 400, Msg: msg})
}

func Unauthorized(c *gin.Context, msg string) {
	c.JSON(http.StatusUnauthorized, R{Code: 401, Msg: msg})
}

func Forbidden(c *gin.Context, msg string) {
	c.JSON(http.StatusForbidden, R{Code: 403, Msg: msg})
}

func NotFound(c *gin.Context, msg string) {
	c.JSON(http.StatusNotFound, R{Code: 404, Msg: msg})
}

func Conflict(c *gin.Context, msg string) {
	c.JSON(http.StatusConflict, R{Code: 409, Msg: msg})
}

func ServerError(c *gin.Context, msg string) {
	c.JSON(http.StatusInternalServerError, R{Code: 500, Msg: msg})
}

// HandleDBError maps a GORM error to the appropriate HTTP error response.
// Returns true if an error was handled (caller should return), false if no error.
func HandleDBError(c *gin.Context, err error, notFoundMsg string) bool {
	if err == nil {
		return false
	}
	if err == gorm.ErrRecordNotFound {
		NotFound(c, notFoundMsg)
	} else {
		ServerError(c, "数据库操作失败")
	}
	return true
}

// ---- Pagination Helper ----

// Pagination holds page/pageSize parsed from query params with safe defaults.
type Pagination struct {
	Page     int
	PageSize int
	Offset   int
}

// GetPagination extracts pagination from query parameters with safe bounds.
func GetPagination(c *gin.Context) Pagination {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "10"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 200 {
		pageSize = 200
	}

	return Pagination{
		Page:     page,
		PageSize: pageSize,
		Offset:   (page - 1) * pageSize,
	}
}

// Paginate applies LIMIT/OFFSET/ORDER to a GORM query.
func Paginate(db *gorm.DB, p Pagination, orderBy string) *gorm.DB {
	if orderBy == "" {
		orderBy = "created_at DESC"
	}
	return db.Limit(p.PageSize).Offset(p.Offset).Order(orderBy)
}

package controllers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"ai-multi-platform-release/backend-go/services"
)

type UploadsController struct {
	BaseController
}

// Upload POST /api/uploads
// multipart/form-data，字段名 file，保存到 uploads 目录并返回可访问 URL。
func (c *UploadsController) Upload() {
	if !c.CheckPermissionAny([]string{
		"inspection:create:write",
		"inspection:update:write",
		"inspection:template:create:write",
		"inspection:template:update:write",
		"inspection:material:create:write",
		"inspection:material:update:write",
		"material:create:write",
		"material:update:write",
		"content:create:write",
	}) {
		return
	}
	file, header, err := c.GetFile("file")
	if err != nil {
		c.WriteError(http.StatusBadRequest, "未接收到文件")
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !isAllowedImageExt(ext) {
		c.WriteError(http.StatusBadRequest, "仅支持 jpg / jpeg / png / webp 图片格式")
		return
	}
	if header.Size > 10*1024*1024 {
		c.WriteError(http.StatusBadRequest, "图片大小不能超过 10MB")
		return
	}

	uploadDir := services.GetUploadDir()
	if err := os.MkdirAll(uploadDir, 0o755); err != nil {
		c.WriteError(http.StatusInternalServerError, "创建上传目录失败")
		return
	}
	filename := newID() + ext
	dst, err := os.Create(filepath.Join(uploadDir, filename))
	if err != nil {
		c.WriteError(http.StatusInternalServerError, "保存文件失败")
		return
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		c.WriteError(http.StatusInternalServerError, "保存文件失败")
		return
	}
	c.Created(map[string]interface{}{
		"url":  fmt.Sprintf("/uploads/%s", filename),
		"name": header.Filename,
		"size": header.Size,
	})
}

func isAllowedImageExt(ext string) bool {
	switch ext {
	case ".jpg", ".jpeg", ".png", ".webp":
		return true
	}
	return false
}

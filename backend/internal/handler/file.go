package handler

import (
	"fmt"
	"log"

	"blog/pkg/filemanager"

	"github.com/gin-gonic/gin"
)

// FileRequest 文件操作请求结构
type FileRequest struct {
	Filename         string `json:"filename"`
	Content          string `json:"content"`
	OriginalFilename string `json:"originalFilename"` // 用于重命名检测
}

// FileHandler 文件管理处理器
type FileHandler struct{}

// NewFileHandler 创建文件处理器
func NewFileHandler() *FileHandler {
	return &FileHandler{}
}

// GetAllFiles 获取所有文件
func (h *FileHandler) GetAllFiles(c *gin.Context) {
	log.Printf("收到获取文件列表请求")
	files, err := filemanager.GetAllFiles()
	if err != nil {
		log.Printf("获取文件列表失败: %v", err)
		c.JSON(500, gin.H{"error": fmt.Sprintf("获取文件列表失败: %v", err)})
		return
	}
	log.Printf("成功获取文件列表，文件数量: %d", len(files))
	c.JSON(200, files)
}

// GetFileContent 获取单个文件内容
func (h *FileHandler) GetFileContent(c *gin.Context) {
	filename := c.Param("filename")
	// 移除路径参数前面的斜杠
	if len(filename) > 0 && filename[0] == '/' {
		filename = filename[1:]
	}
	log.Printf("收到获取文件内容请求: %s", filename)
	content, err := filemanager.GetFileContent(filename)
	if err != nil {
		log.Printf("获取文件内容失败: %v", err)
		c.JSON(404, gin.H{"error": fmt.Sprintf("文件不存在或无法读取: %v", err)})
		return
	}
	log.Printf("成功获取文件内容: %s", filename)
	c.String(200, content)
}

// SaveFile 保存文件
func (h *FileHandler) SaveFile(c *gin.Context) {
	var req FileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "请求参数错误"})
		return
	}

	// 检查是否需要重命名（删除旧文件）
	if req.OriginalFilename != "" && req.OriginalFilename != req.Filename {
		log.Printf("检测到重命名操作: %s -> %s", req.OriginalFilename, req.Filename)
		if err := filemanager.DeleteFile(req.OriginalFilename); err != nil {
			log.Printf("重命名时删除旧文件失败: %v", err)
		}
	}

	if err := filemanager.SaveFile(req.Filename, req.Content); err != nil {
		c.JSON(500, gin.H{"error": "保存文件失败"})
		return
	}

	// 更新侧边栏配置
	if err := filemanager.UpdateSidebarConfig(); err != nil {
		log.Printf("更新侧边栏配置失败: %v", err)
	}

	c.JSON(200, gin.H{"status": "success"})
}

// DeleteFile 删除文件
func (h *FileHandler) DeleteFile(c *gin.Context) {
	filename := c.Param("filename")
	log.Printf("收到删除文件请求，原始路径: %s", filename)

	// 移除路径参数前面的斜杠
	if len(filename) > 0 && filename[0] == '/' {
		filename = filename[1:]
		log.Printf("处理后的文件路径: %s", filename)
	}

	if err := filemanager.DeleteFile(filename); err != nil {
		log.Printf("删除文件失败: %v", err)
		c.JSON(500, gin.H{"error": fmt.Sprintf("删除失败: %v", err)})
		return
	}

	// 更新侧边栏配置
	if err := filemanager.UpdateSidebarConfig(); err != nil {
		log.Printf("更新侧边栏配置失败: %v", err)
	}

	log.Printf("文件删除成功: %s", filename)
	c.JSON(200, gin.H{"status": "success"})
}

// BuildSite 构建站点
func (h *FileHandler) BuildSite(c *gin.Context) {
	if err := filemanager.BuildSite(); err != nil {
		c.JSON(500, gin.H{"error": "构建失败"})
		return
	}
	c.JSON(200, gin.H{"status": "success"})
}

// UploadFiles 批量上传文件
func (h *FileHandler) UploadFiles(c *gin.Context) {
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(400, gin.H{"error": fmt.Sprintf("上传失败: %v", err)})
		return
	}

	files := form.File["files"]
	var successCount, failCount int
	var errorMsgs []string

	for _, file := range files {
		log.Printf("正在处理上传文件: %s", file.Filename)

		src, err := file.Open()
		if err != nil {
			failCount++
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s: 打开失败", file.Filename))
			continue
		}
		defer src.Close()

		// 限制文件大小 10MB
		if file.Size > 10*1024*1024 {
			failCount++
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s: 文件过大", file.Filename))
			continue
		}

		contentBytes := make([]byte, file.Size)
		_, err = src.Read(contentBytes)
		if err != nil {
			failCount++
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s: 读取失败", file.Filename))
			continue
		}

		if err := filemanager.SaveFile(file.Filename, string(contentBytes)); err != nil {
			failCount++
			errorMsgs = append(errorMsgs, fmt.Sprintf("%s: 保存失败 - %v", file.Filename, err))
		} else {
			successCount++
		}
	}

	// 更新侧边栏
	if successCount > 0 {
		if err := filemanager.UpdateSidebarConfig(); err != nil {
			log.Printf("批量上传后更新侧边栏失败: %v", err)
		}
	}

	c.JSON(200, gin.H{
		"success": successCount,
		"failed":  failCount,
		"errors":  errorMsgs,
		"total":   len(files),
	})
}

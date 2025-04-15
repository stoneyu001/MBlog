package filemanager

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// 文章文件的目录
const ArticlesDir = "/app/frontend/docs/articles"

// 初始化文件管理器，确保目录存在
func Init() error {
	if _, err := os.Stat(ArticlesDir); os.IsNotExist(err) {
		return os.MkdirAll(ArticlesDir, 0755)
	}
	return nil
}

// 获取所有文章文件
func GetAllFiles() ([]string, error) {
	files, err := ioutil.ReadDir(ArticlesDir)
	if err != nil {
		return nil, err
	}

	var filenames []string
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".md") || strings.HasSuffix(file.Name(), ".markdown")) {
			filenames = append(filenames, file.Name())
		}
	}
	return filenames, nil
}

// 获取文件内容
func GetFileContent(filename string) (string, error) {
	// 清理文件名，防止目录遍历攻击
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") {
		return "", fmt.Errorf("无效的文件名")
	}

	filePath := filepath.Join(ArticlesDir, filename)
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

// 保存文件
func SaveFile(filename string, content string) error {
	// 清理文件名，防止目录遍历攻击
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") {
		return fmt.Errorf("无效的文件名")
	}

	// 确保文件名以.md结尾
	if !strings.HasSuffix(filename, ".md") && !strings.HasSuffix(filename, ".markdown") {
		filename = filename + ".md"
	}

	filePath := filepath.Join(ArticlesDir, filename)
	return ioutil.WriteFile(filePath, []byte(content), 0644)
}

// 删除文件
func DeleteFile(filename string) error {
	// 清理文件名，防止目录遍历攻击
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") {
		return fmt.Errorf("无效的文件名")
	}

	filePath := filepath.Join(ArticlesDir, filename)
	return os.Remove(filePath)
}

// 更新侧边栏配置
func UpdateSidebarConfig() error {
	files, err := GetAllFiles()
	if err != nil {
		return err
	}

	// 构建配置片段
	var sidebarItems string
	for _, file := range files {
		title := strings.TrimSuffix(strings.TrimSuffix(file, ".md"), ".markdown")
		sidebarItems += fmt.Sprintf("          { text: '%s', link: '/articles/%s' },\n", title, file)
	}

	// 读取现有配置文件
	configPath := "/app/frontend/docs/.vitepress/config.mts"
	content, err := ioutil.ReadFile(configPath)
	if err != nil {
		return err
	}

	// 替换侧边栏配置部分
	strContent := string(content)
	startMarker := "      {\n        text: '目录',\n        items: ["
	endMarker := "        ]\n      }"

	startIndex := strings.Index(strContent, startMarker)
	if startIndex == -1 {
		return fmt.Errorf("无法找到侧边栏配置起始标记")
	}
	startIndex += len(startMarker)

	endIndex := strings.Index(strContent[startIndex:], endMarker)
	if endIndex == -1 {
		return fmt.Errorf("无法找到侧边栏配置结束标记")
	}
	endIndex += startIndex

	newContent := strContent[:startIndex] + "\n" + sidebarItems + "        " + strContent[endIndex:]
	return ioutil.WriteFile(configPath, []byte(newContent), 0644)
}

// 构建站点
func BuildSite() error {
	cmd := exec.Command("npm", "run", "build")
	cmd.Dir = "/app/frontend"
	return cmd.Run()
}

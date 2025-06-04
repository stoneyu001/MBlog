package filemanager

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// 文章文件的目录
var ArticlesDirs = []string{
	"/app/frontend/docs/tech",
	"/app/frontend/docs/life",
}

// 获取工作目录
func getWorkingDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		log.Printf("获取工作目录失败: %v", err)
		return "", err
	}
	log.Printf("当前工作目录: %s", dir)
	return dir, nil
}

// 初始化文件管理器，确保目录存在
func Init() error {
	for _, dir := range ArticlesDirs {
		log.Printf("检查目录是否存在: %s", dir)
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			log.Printf("创建目录: %s", dir)
			if err := os.MkdirAll(dir, 0755); err != nil {
				log.Printf("创建目录失败: %v", err)
				return err
			}
		}
	}
	return nil
}

// 获取所有文章文件
func GetAllFiles() ([]string, error) {
	var allFiles []string

	log.Printf("开始扫描文章文件...")
	for _, dir := range ArticlesDirs {
		log.Printf("扫描目录: %s", dir)

		// 检查目录是否存在
		if _, err := os.Stat(dir); os.IsNotExist(err) {
			log.Printf("目录不存在: %s", dir)
			continue
		}

		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				log.Printf("遍历文件失败: %v", err)
				return err
			}
			if !info.IsDir() && (strings.HasSuffix(info.Name(), ".md") || strings.HasSuffix(info.Name(), ".markdown")) {
				// 使用完整路径
				log.Printf("找到文章: %s", path)
				allFiles = append(allFiles, path)
			}
			return nil
		})
		if err != nil {
			log.Printf("扫描目录失败: %v", err)
			return nil, err
		}
	}

	log.Printf("找到文章总数: %d", len(allFiles))
	for i, file := range allFiles {
		log.Printf("文章[%d]: %s", i+1, file)
	}

	return allFiles, nil
}

// 获取文件内容
func GetFileContent(filename string) (string, error) {
	// 清理文件名，防止目录遍历攻击
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") {
		return "", fmt.Errorf("无效的文件名")
	}

	// 如果文件名以tech/或life/开头，直接在对应目录查找
	var searchDirs []string
	if strings.HasPrefix(filename, "tech/") {
		searchDirs = []string{ArticlesDirs[0]}
		filename = strings.TrimPrefix(filename, "tech/")
	} else if strings.HasPrefix(filename, "life/") {
		searchDirs = []string{ArticlesDirs[1]}
		filename = strings.TrimPrefix(filename, "life/")
	} else {
		searchDirs = ArticlesDirs
	}

	// 在指定目录中查找文件
	for _, dir := range searchDirs {
		fullPath := filepath.Join(dir, filename)
		log.Printf("尝试读取文件: %s", fullPath)

		content, err := ioutil.ReadFile(fullPath)
		if err == nil {
			return string(content), nil
		}
		log.Printf("在 %s 中读取文件失败: %v", fullPath, err)
	}

	return "", fmt.Errorf("文件不存在或无法读取")
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

	// 根据文件名前缀决定保存目录
	var targetDir string
	if strings.HasPrefix(filename, "tech/") {
		targetDir = ArticlesDirs[0] // tech目录
	} else if strings.HasPrefix(filename, "life/") {
		targetDir = ArticlesDirs[1] // life目录
	} else {
		targetDir = ArticlesDirs[0] // 默认保存到tech目录
	}

	fullPath := filepath.Join(targetDir, filename)
	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(fullPath, []byte(content), 0644)
}

// 删除文件
func DeleteFile(filename string) error {
	// 清理文件名，防止目录遍历攻击
	filename = filepath.Clean(filename)
	if strings.Contains(filename, "..") {
		log.Printf("检测到非法的文件路径: %s", filename)
		return fmt.Errorf("无效的文件名")
	}

	log.Printf("准备删除文件，清理后的路径: %s", filename)

	// 处理文件路径
	// 1. 移除可能存在的 /app/frontend/docs/ 前缀
	filename = strings.TrimPrefix(filename, "/app/frontend/docs/")

	// 2. 确定文件类别（tech 或 life）
	var targetDir string
	if strings.HasPrefix(filename, "tech/") {
		targetDir = ArticlesDirs[0]
		filename = strings.TrimPrefix(filename, "tech/")
	} else if strings.HasPrefix(filename, "life/") {
		targetDir = ArticlesDirs[1]
		filename = strings.TrimPrefix(filename, "life/")
	} else {
		// 如果没有前缀，默认为tech目录
		targetDir = ArticlesDirs[0]
	}

	// 构建完整的文件路径
	fullPath := filepath.Join(targetDir, filename)
	log.Printf("尝试删除文件: %s", fullPath)

	// 检查文件是否存在
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		log.Printf("文件不存在: %s", fullPath)
		return fmt.Errorf("文件不存在: %s", filename)
	}

	// 尝试删除文件
	if err := os.Remove(fullPath); err != nil {
		log.Printf("删除文件失败: %v", err)
		return fmt.Errorf("删除失败: %v", err)
	}

	log.Printf("文件删除成功: %s", fullPath)
	return nil
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
		// 获取相对路径，去掉前缀部分
		relPath := file
		for _, dir := range ArticlesDirs {
			if strings.HasPrefix(file, dir) {
				relPath = strings.TrimPrefix(file, dir+"/")
				break
			}
		}
		title := strings.TrimSuffix(strings.TrimSuffix(relPath, ".md"), ".markdown")
		sidebarItems += fmt.Sprintf("          { text: '%s', link: '/%s' },\n", title, relPath)
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
	// 创建一个新的命令，设置工作目录
	installCmd := exec.Command("npm", "install")
	installCmd.Dir = "/app/frontend"

	// 设置命令的输出
	var installOutput strings.Builder
	installCmd.Stdout = &installOutput
	installCmd.Stderr = &installOutput

	// 执行安装命令
	if err := installCmd.Run(); err != nil {
		log.Printf("npm install 失败: %v\n输出: %s", err, installOutput.String())
		return fmt.Errorf("安装依赖失败: %v", err)
	}

	// 创建构建命令
	buildCmd := exec.Command("npm", "run", "build")
	buildCmd.Dir = "/app/frontend"

	// 设置命令的输出
	var buildOutput strings.Builder
	buildCmd.Stdout = &buildOutput
	buildCmd.Stderr = &buildOutput

	// 执行构建命令
	if err := buildCmd.Run(); err != nil {
		log.Printf("npm run build 失败: %v\n输出: %s", err, buildOutput.String())
		return fmt.Errorf("构建失败: %v", err)
	}

	log.Printf("站点构建成功")
	return nil
}

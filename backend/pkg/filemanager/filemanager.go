package filemanager

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	ProjectRoot  string
	FrontendDir  string
	ArticlesDirs []string
)

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
	// Determine ProjectRoot
	// Check for Docker environment first
	if _, err := os.Stat("/app/frontend"); err == nil {
		ProjectRoot = "/app"
	} else {
		// Local development
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		// If running from backend directory
		if filepath.Base(wd) == "backend" {
			ProjectRoot = filepath.Dir(wd)
		} else {
			ProjectRoot = wd
		}
	}

	FrontendDir = filepath.Join(ProjectRoot, "frontend")
	ArticlesDirs = []string{
		filepath.Join(FrontendDir, "docs", "tech"),
		filepath.Join(FrontendDir, "docs", "life"),
	}

	log.Printf("Project Root: %s", ProjectRoot)
	log.Printf("Frontend Dir: %s", FrontendDir)

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
				// 获取相对路径
				relPath, err := filepath.Rel(dir, path)
				if err != nil {
					log.Printf("获取相对路径失败: %v", err)
					return nil
				}
				// 加上目录前缀 (tech/ or life/)
				category := filepath.Base(dir)
				finalPath := filepath.ToSlash(filepath.Join(category, relPath))

				log.Printf("找到文章: %s", finalPath)
				allFiles = append(allFiles, finalPath)
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
	filename = filepath.ToSlash(filepath.Clean(filename))
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
	log.Printf("SaveFile: 开始保存文件, 原始文件名: %s", filename)

	// 清理文件名，防止目录遍历攻击
	filename = filepath.ToSlash(filepath.Clean(filename))
	log.Printf("SaveFile: 清理后的文件名: %s", filename)

	if strings.Contains(filename, "..") {
		return fmt.Errorf("无效的文件名")
	}

	// 确保文件名以.md结尾
	if !strings.HasSuffix(filename, ".md") && !strings.HasSuffix(filename, ".markdown") {
		filename = filename + ".md"
	}

	// 从文件名生成标题（去掉目录前缀）
	baseFilename := filepath.Base(filename)
	title := strings.TrimSuffix(baseFilename, ".md")
	title = strings.TrimSuffix(title, ".markdown")

	// 检查内容是否已经包含frontmatter
	if !strings.HasPrefix(content, "---") {
		// 添加frontmatter，使用VitePress标准格式
		frontmatter := fmt.Sprintf(`---
title: %s
---

# %s

`, title, title)
		content = frontmatter + content
	}

	// 确定目标目录和文件名
	var targetDir string
	targetFilename := filename

	// 如果文件名已经包含tech/或life/前缀，直接使用对应目录
	if strings.HasPrefix(filename, "tech/") {
		targetDir = ArticlesDirs[0]
		targetFilename = strings.TrimPrefix(filename, "tech/")
		log.Printf("SaveFile: 匹配到 tech 目录")
	} else if strings.HasPrefix(filename, "life/") {
		targetDir = ArticlesDirs[1]
		targetFilename = strings.TrimPrefix(filename, "life/")
		log.Printf("SaveFile: 匹配到 life 目录")
	} else {
		// 如果没有前缀，默认保存到tech目录
		targetDir = ArticlesDirs[0]
		log.Printf("SaveFile: 未匹配到前缀，默认使用 tech 目录")
	}

	fullPath := filepath.Join(targetDir, targetFilename)
	log.Printf("SaveFile: 最终保存路径: %s", fullPath)

	// 确保目标目录存在
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		log.Printf("SaveFile: 创建目录失败: %v", err)
		return err
	}
	if err := ioutil.WriteFile(fullPath, []byte(content), 0644); err != nil {
		log.Printf("SaveFile: 写入文件失败: %v", err)
		return err
	}
	log.Printf("SaveFile: 文件保存成功")
	return nil
}

// 删除文件
func DeleteFile(filename string) error {
	// 清理文件名，防止目录遍历攻击
	filename = filepath.ToSlash(filepath.Clean(filename))
	if strings.Contains(filename, "..") {
		log.Printf("检测到非法的文件路径: %s", filename)
		return fmt.Errorf("无效的文件名")
	}

	log.Printf("准备删除文件，清理后的路径: %s", filename)

	// 处理文件路径
	// 1. 移除可能存在的 前缀
	docsPrefix := filepath.ToSlash(filepath.Join(FrontendDir, "docs")) + "/"
	filename = strings.TrimPrefix(filename, docsPrefix)
	// 兼容旧的硬编码路径
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
		var relPath string
		var category string

		for _, dir := range ArticlesDirs {
			// 将路径标准化为 Slash 分隔，以便比较
			normDir := filepath.ToSlash(dir)
			normFile := filepath.ToSlash(file)

			if strings.HasPrefix(normFile, normDir) {
				relPath = strings.TrimPrefix(normFile, normDir+"/")
				category = filepath.Base(normDir)
				break
			}
		}

		if relPath != "" {
			title := strings.TrimSuffix(strings.TrimSuffix(relPath, ".md"), ".markdown")
			sidebarItems += fmt.Sprintf("          { text: '%s', link: '/%s/%s' },\n", title, category, title)
		}
	}

	// 读取现有配置文件
	configPath := filepath.Join(FrontendDir, "docs", ".vitepress", "config.mts")
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
	npmName := "npm"
	if runtime.GOOS == "windows" {
		npmName = "npm.cmd"
	}

	// 创建一个新的命令，设置工作目录
	installCmd := exec.Command(npmName, "install")
	installCmd.Dir = FrontendDir

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
	buildCmd := exec.Command(npmName, "run", "build")
	buildCmd.Dir = FrontendDir

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

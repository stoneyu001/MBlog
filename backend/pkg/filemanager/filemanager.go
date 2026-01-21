package filemanager

import (
	"fmt"
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
	ArticlesDirs map[string]string
)

// Init 初始化文件管理器
func Init() error {
	// 确定 ProjectRoot
	if _, err := os.Stat("/app/frontend"); err == nil {
		ProjectRoot = "/app"
	} else {
		wd, err := os.Getwd()
		if err != nil {
			return err
		}
		if filepath.Base(wd) == "backend" {
			ProjectRoot = filepath.Dir(wd)
		} else {
			ProjectRoot = wd
		}
	}

	FrontendDir = filepath.Join(ProjectRoot, "frontend")
	ArticlesDirs = map[string]string{
		"tech": filepath.Join(FrontendDir, "docs", "tech"),
		"life": filepath.Join(FrontendDir, "docs", "life"),
	}

	log.Printf("Project Root: %s", ProjectRoot)

	for category, dir := range ArticlesDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("创建目录失败 [%s]: %v", category, err)
		}
	}
	return nil
}

// resolvePath 解析文件路径，返回绝对路径和分类
func resolvePath(filename string) (fullPath string, category string, err error) {
	filename = filepath.ToSlash(filepath.Clean(filename))
	if strings.Contains(filename, "..") {
		return "", "", fmt.Errorf("非法路径")
	}

	// 移除可能的绝对路径前缀
	docsDir := filepath.ToSlash(filepath.Join(FrontendDir, "docs"))
	if strings.HasPrefix(filename, docsDir) {
		filename = strings.TrimPrefix(filename, docsDir+"/")
	}

	// 确定分类
	category = "tech"
	targetDir := ArticlesDirs["tech"]
	subPath := filename

	if strings.HasPrefix(filename, "tech/") {
		category = "tech"
		targetDir = ArticlesDirs["tech"]
		subPath = strings.TrimPrefix(filename, "tech/")
	} else if strings.HasPrefix(filename, "life/") {
		category = "life"
		targetDir = ArticlesDirs["life"]
		subPath = strings.TrimPrefix(filename, "life/")
	}

	fullPath = filepath.Join(targetDir, subPath)
	return fullPath, category, nil
}

// GetAllFiles 获取所有文章文件
func GetAllFiles() ([]string, error) {
	var allFiles []string

	for category, dir := range ArticlesDirs {
		err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() && (strings.HasSuffix(info.Name(), ".md") || strings.HasSuffix(info.Name(), ".markdown")) {
				relPath, err := filepath.Rel(dir, path)
				if err != nil {
					return nil
				}
				finalPath := filepath.ToSlash(filepath.Join(category, relPath))
				allFiles = append(allFiles, finalPath)
			}
			return nil
		})
		if err != nil {
			log.Printf("扫描目录 %s 失败: %v", category, err)
		}
	}
	return allFiles, nil
}

// GetFileContent 获取文件内容
func GetFileContent(filename string) (string, error) {
	fullPath, _, err := resolvePath(filename)
	if err != nil {
		return "", err
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("文件不存在或无法读取")
	}
	return string(content), nil
}

// SaveFile 保存文件
func SaveFile(filename string, content string) error {
	if !strings.HasSuffix(filename, ".md") && !strings.HasSuffix(filename, ".markdown") {
		filename = filename + ".md"
	}

	fullPath, _, err := resolvePath(filename)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(content, "---") {
		baseName := filepath.Base(fullPath)
		title := strings.TrimSuffix(strings.TrimSuffix(baseName, ".md"), ".markdown")
		frontmatter := fmt.Sprintf("---\ntitle: %s\n---\n\n# %s\n\n", title, title)
		content = frontmatter + content
	}

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	log.Printf("保存文件: %s", fullPath)
	return os.WriteFile(fullPath, []byte(content), 0644)
}

// DeleteFile 删除文件
func DeleteFile(filename string) error {
	fullPath, _, err := resolvePath(filename)
	if err != nil {
		return err
	}

	// 安全检查：只允许删除 docs 目录下的文件
	if !strings.Contains(filepath.ToSlash(fullPath), "/docs/") {
		return fmt.Errorf("安全拒绝：只能删除 docs 目录下的文件")
	}

	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		return fmt.Errorf("文件不存在")
	}

	log.Printf("删除文件: %s", fullPath)
	return os.Remove(fullPath)
}

// UpdateSidebarConfig 更新侧边栏配置
func UpdateSidebarConfig() error {
	files, err := GetAllFiles()
	if err != nil {
		return err
	}

	var sidebarItems strings.Builder
	for _, file := range files {
		parts := strings.SplitN(file, "/", 2)
		if len(parts) != 2 {
			continue
		}
		category, relPath := parts[0], parts[1]
		title := strings.TrimSuffix(strings.TrimSuffix(relPath, ".md"), ".markdown")
		line := fmt.Sprintf("          { text: '%s', link: '/%s/%s' },\n", title, category, title)
		sidebarItems.WriteString(line)
	}

	configPath := filepath.Join(FrontendDir, "docs", ".vitepress", "config.mts")
	contentBytes, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	content := string(contentBytes)

	startMarker := "text: '目录',"
	if !strings.Contains(content, startMarker) {
		return fmt.Errorf("配置文件中未找到锚点")
	}

	sIdx := strings.Index(content, startMarker)
	subStr := content[sIdx:]
	itemsStart := strings.Index(subStr, "items: [")
	if itemsStart == -1 {
		return fmt.Errorf("锚点后未找到 items 定义")
	}

	realStart := sIdx + itemsStart + len("items: [")
	endIndex := strings.Index(content[realStart:], "]")
	if endIndex == -1 {
		return fmt.Errorf("未找到闭合的 ]")
	}
	endIndex += realStart

	newContent := content[:realStart] + "\n" + sidebarItems.String() + "        " + content[endIndex:]
	return os.WriteFile(configPath, []byte(newContent), 0644)
}

// BuildSite 构建站点
func BuildSite() error {
	npmName := "npm"
	if runtime.GOOS == "windows" {
		npmName = "npm.cmd"
	}

	log.Printf("开始构建站点...")
	log.Printf("前端目录: %s", FrontendDir)

	// 检查 npm 是否可用
	checkCmd := exec.Command(npmName, "--version")
	if out, err := checkCmd.CombinedOutput(); err != nil {
		log.Printf("npm 检查失败: %v, 输出: %s", err, string(out))
		return fmt.Errorf("npm 不可用: %v", err)
	} else {
		log.Printf("npm 版本: %s", strings.TrimSpace(string(out)))
	}

	// 执行 npm install
	log.Printf("执行 npm install...")
	cmd := exec.Command(npmName, "install")
	cmd.Dir = FrontendDir
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("npm install 失败: %v", err)
		log.Printf("npm install 输出: %s", string(out))
		return fmt.Errorf("npm install 失败: %v", err)
	}
	log.Printf("npm install 完成")

	// 修复 node_modules/.bin 下可执行文件的权限（解决 Docker 挂载目录权限问题）
	binDir := filepath.Join(FrontendDir, "node_modules", ".bin")
	if runtime.GOOS != "windows" {
		log.Printf("修复 %s 目录权限...", binDir)
		chmodCmd := exec.Command("chmod", "-R", "+x", binDir)
		if out, err := chmodCmd.CombinedOutput(); err != nil {
			log.Printf("chmod 失败 (非致命): %v, 输出: %s", err, string(out))
		}
	}

	// 执行 npm run build
	log.Printf("执行 npm run build...")
	cmd = exec.Command(npmName, "run", "build")
	cmd.Dir = FrontendDir
	if out, err := cmd.CombinedOutput(); err != nil {
		log.Printf("npm run build 失败: %v", err)
		log.Printf("npm run build 输出: %s", string(out))
		return fmt.Errorf("npm run build 失败: %v", err)
	}
	log.Printf("npm run build 完成")

	log.Println("站点构建成功")
	return nil
}

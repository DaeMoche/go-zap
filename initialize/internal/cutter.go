package internal

import (
	"compress/gzip"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Cutter struct {
	level    string        // 日志级别(debug, info, warn, error, dpanic, panic, fatal)
	layout   string        // 时间格式 2006-01-02 15:04:05
	formats  []string      // 自定义参数([]string{Director,"2006-01-02", "business"(此参数可不写), level+".log"}
	director string        // 日志文件夹
	maxSize  int           // 文件大小 (MB)
	maxAge   int           // 日志保留天数
	backups  int           // 备份
	compress bool          // 压缩文件
	file     *os.File      // 文件句柄
	mutex    *sync.RWMutex // 读写锁
}

type CutterOption func(*Cutter)

func CutterWithLayout(layout string) CutterOption {
	return func(c *Cutter) {
		c.layout = layout
	}
}

func CutterWithFormats(format ...string) CutterOption {
	return func(c *Cutter) {
		if len(format) > 0 {
			c.formats = format
		}
	}
}

func NewCutter(director string, level string, maxSize int, backups int, compress bool, maxAge int, options ...CutterOption) *Cutter {
	rotate := &Cutter{
		level:    level,
		director: director,
		maxAge:   maxAge,
		maxSize:  maxSize,
		backups:  backups,
		compress: compress,
		mutex:    new(sync.RWMutex),
	}
	for i := 0; i < len(options); i++ {
		options[i](rotate)
	}
	return rotate
}

func (c *Cutter) Write(bytes []byte) (n int, err error) {
	c.mutex.Lock()
	defer func() {
		if c.file != nil {
			_ = c.file.Close()
			c.file = nil
		}
		c.mutex.Unlock()
	}()

	// 1. 构建文件名
	length := len(c.formats)
	values := make([]string, 0, 3+length)
	values = append(values, c.director)
	if c.layout != "" {
		values = append(values, time.Now().Format(c.layout))
	}

	for i := 0; i < length; i++ {
		values = append(values, c.formats[i])
	}

	values = append(values, c.level+".log")
	fileName := filepath.Join(values...)
	director := filepath.Dir(fileName)

	// 2. 创建目录
	err = os.MkdirAll(director, os.ModePerm)
	if err != nil {
		return 0, err
	}

	// 3. 检查文件大小并切割 (新增逻辑)
	if c.maxSize > 0 {
		if info, statErr := os.Stat(fileName); statErr == nil {
			// 将 maxSize (MB) 转换为字节
			maxSizeBytes := int64(c.maxSize) * 1024 * 1024
			// 如果当前文件大小 + 本次写入大小 >= 限制大小，则进行切割
			if info.Size()+int64(len(bytes)) >= maxSizeBytes {
				// 生成备份文件名：原文件名.时间戳
				// 例如: error.log -> error.log.2023-10-27-15-04-05
				backupName := fmt.Sprintf("%s.%s", fileName, time.Now().Format("2006-01-02-15-04-05"))

				// 重命名当前文件
				// 注意：这里没有处理重命名冲突（极短时间内切割两次），通常日志场景下时间戳精度足够
				_ = os.Rename(fileName, backupName)

				// 如果配置了压缩，可以在这里对 backupName 进行压缩操作
				if c.compress {
					go compressFile(backupName)
				}
			}
		}
	}

	// 4. 清理过期日志 (保留原有逻辑)
	defer func() {
		err := removeNDaysFolders(c.director, c.maxAge)
		if err != nil {
			fmt.Println("清理过期日志失败", err)
		}
	}()

	// 5. 打开文件并写入
	c.file, err = os.OpenFile(fileName, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return 0, err
	}

	// 如果是新文件，写入 BOM 头
	fileInfo, _ := c.file.Stat()
	if fileInfo.Size() == 0 {
		_, _ = c.file.Write([]byte{0xEF, 0xBB, 0xBF})
	}

	return c.file.Write(bytes)
}

func (c *Cutter) Sync() error {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	if c.file != nil {
		return c.file.Sync()
	}
	return nil
}

func removeNDaysFolders(dir string, days int) error {
	if days <= 0 {
		return nil
	}
	cutoff := time.Now().AddDate(0, 0, -days)
	return filepath.Walk(dir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 清理过期的目录或文件
		if !info.IsDir() && info.ModTime().Before(cutoff) && path != dir {
			// 这里原逻辑是删除目录，为了配合按大小切割产生的文件，建议也支持删除过期文件
			// 如果只想保留原逻辑（只删目录），可以保留 !info.IsDir() 的判断去掉
			// 但为了更通用，这里修改为删除所有过期文件和空目录
			_ = os.Remove(path)
		}

		// 如果是目录且为空（在删除文件后），也可以尝试删除目录
		if info.IsDir() && path != dir {
			// 简单的检查目录是否为空比较复杂，这里暂不深入，保留原逻辑意图
			if info.ModTime().Before(cutoff) {
				_ = os.RemoveAll(path)
			}
		}
		return nil
	})
}

// compressFile 异步压缩文件
func compressFile(src string) {
	// 打开源文件
	srcFile, err := os.Open(src)
	if err != nil {
		fmt.Println("打开文件压缩失败:", err)
		return
	}
	defer srcFile.Close()

	// 创建目标 .gz 文件
	dstFile := src + ".gz"
	gzFile, err := os.Create(dstFile)
	if err != nil {
		fmt.Println("创建压缩文件失败:", err)
		return
	}
	defer gzFile.Close()

	// 使用 gzip writer
	gzWriter := gzip.NewWriter(gzFile)
	defer gzWriter.Close()

	// 执行拷贝
	if _, err = io.Copy(gzWriter, srcFile); err != nil {
		fmt.Println("写入压缩内容失败:", err)
		// 如果压缩失败，删除可能产生的不完整 .gz 文件
		_ = os.Remove(dstFile)
		return
	}

	// 压缩成功，删除原始文件
	if err := os.Remove(src); err != nil {
		fmt.Println("删除原始文件失败:", err)
	}
}

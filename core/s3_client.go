package core

import (
	"context"
	"fmt"
	"log"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"farshore.ai/fast-comfy-api/model"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type S3Client struct {
	Client *minio.Client
	Config model.S3Config
}

// NewS3Client 创建一个新的 S3 客户端实例
func NewS3Client(cfg model.S3Config) (*S3Client, error) {
	client, err := minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	})
	if err != nil {
		return nil, fmt.Errorf("连接 S3 失败: %w", err)
	}

	s3 := &S3Client{
		Client: client,
		Config: cfg,
	}

	// 确保桶存在并设置策略
	if err := s3.ensureBucket(); err != nil {
		return nil, err
	}

	return s3, nil
}

// ensureBucket 检查桶是否存在，不存在则创建并设置为公有
func (s *S3Client) ensureBucket() error {
	ctx := context.Background()
	found, err := s.Client.BucketExists(ctx, s.Config.Bucket)
	if err != nil {
		return fmt.Errorf("检查桶失败: %w", err)
	}

	if !found {
		if err := s.Client.MakeBucket(ctx, s.Config.Bucket, minio.MakeBucketOptions{Region: s.Config.Region}); err != nil {
			return fmt.Errorf("创建桶失败: %w", err)
		}
	}

	// 设置桶为公有（MinIO / S3 通用）
	policy := fmt.Sprintf(`{
	  "Version": "2012-10-17",
	  "Statement": [
	    {
	      "Effect": "Allow",
	      "Principal": {"AWS": ["*"]},
	      "Action": ["s3:GetObject"],
	      "Resource": ["arn:aws:s3:::%s/*"]
	    }
	  ]
	}`, s.Config.Bucket)

	if err := s.Client.SetBucketPolicy(ctx, s.Config.Bucket, policy); err != nil {
		// 某些 S3 兼容服务（如 Cloudflare R2）不支持 SetBucketPolicy
		fmt.Printf("⚠️ 设置桶策略失败（可忽略）: %v\n", err)
	}

	return nil
}

// UploadInputFile 上传到 input 目录（自动识别 content-type）
func (s *S3Client) UploadInputFile(ctx context.Context, id string, filePath string) (string, error) {
	return s.uploadWithPrefix(ctx, s.Config.InputPrefix, id, filePath)
}

// UploadOutputFile 上传到 output 目录（自动识别 content-type）
func (s *S3Client) UploadOutputFile(ctx context.Context, id string, filePath string) (string, error) {
	return s.uploadWithPrefix(ctx, s.Config.OutputPrefix, id, filePath)
}

// uploadWithPrefix 内部方法，1. 自动拼接前缀和本地路径 2.重命名文件名为 id.ext 3. 自动识别 content-type 4. 上传文件 5. 返回公有 URL
func (s *S3Client) uploadWithPrefix(ctx context.Context, prefix, id, filePath string) (string, error) {
	log.Printf("⏫ 正在向 S3 上传文件, id: %s, filePath: %s", id, filePath)
	// 1️⃣ 获取文件名
	//ext := filepath.Ext(filePath)
	filename := filepath.Base(filePath)

	// 2️⃣ 将相对路径转换为干净的相对路径（去掉开头的 "./" 或 ".\"）
	cleanPath := filepath.Clean(filePath)
	cleanPath = strings.TrimPrefix(cleanPath, "."+string(filepath.Separator))

	// 3️⃣ 提取目录路径（去掉文件名）
	dir := filepath.Dir(cleanPath)

	// 4️⃣ 替换 Windows 下的 "\" 为 "/"（S3 使用统一的 "/"）
	dir = strings.ReplaceAll(dir, "\\", "/")

	// 5️⃣ 拼接对象名，例如 uploads/tmp/image/234553456/xyz123.jpeg
	var objectName string
	if dir != "." {
		objectName = fmt.Sprintf("%s/%s/%s", prefix, dir, filename)
	} else {
		objectName = fmt.Sprintf("%s/%s%s", prefix, id, filename)
	}

	// 6️⃣ 自动识别 content-type
	contentType := detectContentType(filePath)

	// 7️⃣ 上传文件
	_, err := s.Client.FPutObject(ctx, s.Config.Bucket, objectName, filePath, minio.PutObjectOptions{
		ContentType: contentType,
	})
	if err != nil {
		return "", fmt.Errorf("上传文件失败: %w", err)
	}

	// 日志打印
	fmt.Printf("id: %s : ✅ 上传文件成功: %s\n", id, objectName)

	// 8️⃣ 返回公开 URL
	return s.BuildPublicURL(objectName), nil
}

// 辅助函数 detectContentType 自动识别文件类型
func detectContentType(filePath string) string {
	ext := filepath.Ext(filePath)
	if ext != "" {
		if mimeType := mime.TypeByExtension(ext); mimeType != "" {
			return mimeType
		}
	}

	// 如果扩展名无法识别，再尝试读取文件头
	f, err := os.Open(filePath)
	if err != nil {
		return "application/octet-stream"
	}
	defer f.Close()

	buffer := make([]byte, 512)
	n, _ := f.Read(buffer)
	return http.DetectContentType(buffer[:n])
}

// 辅助函数 BuildPublicURL 构建公有访问 URL
func (s *S3Client) BuildPublicURL(key string) string {
	endpoint := strings.TrimSuffix(s.Config.Endpoint, "/")

	if strings.Contains(endpoint, "amazonaws.com") {
		// AWS S3
		return fmt.Sprintf("https://%s.s3.amazonaws.com/%s", s.Config.Bucket, key)
	}

	// MinIO / 其他 S3 兼容存储
	scheme := "http"
	if s.Config.UseSSL {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s/%s/%s", scheme, endpoint, s.Config.Bucket, key)
}

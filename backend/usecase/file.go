package usecase

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/minio/minio-go/v7"
	"gorm.io/gorm"

	"github.com/chaitin/panda-wiki/config"
	"github.com/chaitin/panda-wiki/consts"
	"github.com/chaitin/panda-wiki/domain"
	"github.com/chaitin/panda-wiki/log"
	"github.com/chaitin/panda-wiki/repo/pg"
	"github.com/chaitin/panda-wiki/store/s3"
	"github.com/chaitin/panda-wiki/utils"
)

type FileUsecase struct {
	logger            *log.Logger
	s3Client          *s3.MinioClient
	config            *config.Config
	systemSettingRepo *pg.SystemSettingRepo
	httpClient        *http.Client
}

const remoteDownloadTimeout = 2 * time.Minute

func NewFileUsecase(logger *log.Logger, s3Client *s3.MinioClient, config *config.Config, systemSettingRepo *pg.SystemSettingRepo) *FileUsecase {
	return &FileUsecase{
		s3Client:          s3Client,
		logger:            logger.WithModule("usecase.file"),
		config:            config,
		systemSettingRepo: systemSettingRepo,
		httpClient: &http.Client{
			Transport: &http.Transport{},
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				// Prevent redirects to bypass SSRF checks
				return http.ErrUseLastResponse
			},
		},
	}
}

func (u *FileUsecase) UploadFileGetUrl(ctx context.Context, kbID string, file *multipart.FileHeader) (string, error) {
	key, err := u.UploadFile(ctx, kbID, file)
	if err != nil {
		return "", err
	}
	return u.s3Client.PublicObjectURL(key), nil
}

func (u *FileUsecase) UploadFile(ctx context.Context, kbID string, file *multipart.FileHeader) (string, error) {
	src, err := file.Open()
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	ext := strings.ToLower(filepath.Ext(file.Filename))

	// Check denied extensions
	if err := u.checkDeniedExtension(ctx, ext); err != nil {
		return "", err
	}

	filename := fmt.Sprintf("%s/%s%s", kbID, uuid.New().String(), ext)

	size := file.Size

	// 安全 MIME 判定：阻止伪装成图片等扩展名的 HTML/JS 载荷在桶域名触发存储型 XSS，
	// 同时保留正常 .svg / Office 文档等的精确 MIME 类型。
	headSrc := io.NewSectionReader(src, 0, 512)
	buf := make([]byte, 512)
	n, _ := io.ReadFull(headSrc, buf)
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("rewind upload stream failed: %w", err)
	}
	contentType := resolveSafeUploadContentType(ext, buf[:n])

	resp, err := u.s3Client.PutObject(
		ctx,
		u.s3Client.Bucket(),
		filename,
		src,
		size,
		minio.PutObjectOptions{
			ContentType: contentType,
			UserMetadata: map[string]string{
				"originalname": file.Filename,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}

	return resp.Key, nil
}

func (u *FileUsecase) UploadFileFromBytes(ctx context.Context, kbID string, filename string, fileBytes []byte) (string, error) {
	// Create a reader from the byte slice
	reader := bytes.NewReader(fileBytes)

	ext := strings.ToLower(filepath.Ext(filename))

	// Check denied extensions
	if err := u.checkDeniedExtension(ctx, ext); err != nil {
		return "", err
	}

	s3Filename := fmt.Sprintf("%s/%s%s", kbID, uuid.New().String(), ext)

	size := int64(len(fileBytes))

	headLen := min(len(fileBytes), 512)
	contentType := resolveSafeUploadContentType(ext, fileBytes[:headLen])

	resp, err := u.s3Client.PutObject(
		ctx,
		u.s3Client.Bucket(),
		s3Filename,
		reader,
		size,
		minio.PutObjectOptions{
			ContentType: contentType,
			UserMetadata: map[string]string{
				"originalname": filename,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}

	return resp.Key, nil
}

func (u *FileUsecase) UploadFileFromReader(
	ctx context.Context,
	kbID string,
	filename string,
	reader io.Reader,
	size int64, // 必须提供对象大小
) (string, error) {
	// 生成唯一文件名
	ext := strings.ToLower(filepath.Ext(filename))

	// Check denied extensions
	if err := u.checkDeniedExtension(ctx, ext); err != nil {
		return "", err
	}

	s3Filename := fmt.Sprintf("%s/%s%s", kbID, uuid.New().String(), ext)

	head := make([]byte, 512)
	n, err := io.ReadFull(reader, head)
	if err != nil && !errors.Is(err, io.EOF) && !errors.Is(err, io.ErrUnexpectedEOF) {
		return "", fmt.Errorf("read upload header failed: %w", err)
	}
	contentType := resolveSafeUploadContentType(ext, head[:n])
	uploadReader := io.MultiReader(bytes.NewReader(head[:n]), reader)

	// 上传到 S3
	_, err = u.s3Client.PutObject(
		ctx,
		u.s3Client.Bucket(),
		s3Filename,
		uploadReader,
		size, // 必须提供对象大小
		minio.PutObjectOptions{
			ContentType: contentType,
			UserMetadata: map[string]string{
				"originalname": filename,
			},
		},
	)
	if err != nil {
		return "", fmt.Errorf("S3 upload failed: %w", err)
	}

	return s3Filename, nil
}

func (u *FileUsecase) UploadFileByUrl(ctx context.Context, kbID string, fileURL string) (string, error) {
	downloadCtx, cancel := context.WithTimeout(ctx, remoteDownloadTimeout)
	defer cancel()

	// Validate URL to prevent SSRF attacks
	validatedIPs, err := utils.ResolveURLForSSRFWithContext(downloadCtx, fileURL)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(downloadCtx, http.MethodGet, fileURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	client := u.httpClientForIPs(validatedIPs)
	defer client.CloseIdleConnections()
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to download file: %w", err)
	}
	defer resp.Body.Close()

	// Reject redirects so every target must pass SSRF validation explicitly.
	if resp.StatusCode >= 300 && resp.StatusCode < 400 {
		return "", fmt.Errorf("redirects are not allowed for security reasons")
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to download file, status: %d", resp.StatusCode)
	}

	const maxRemoteFileSize = 50 * 1024 * 1024 // 50MB
	lr := io.LimitReader(resp.Body, maxRemoteFileSize+1)
	data, err := io.ReadAll(lr)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}
	if len(data) > maxRemoteFileSize {
		return "", fmt.Errorf("failed to read response body: file size exceeds limit of %d bytes", maxRemoteFileSize)
	}

	// Derive the stored type from the downloaded bytes. A remote URL path and
	// Content-Type can both be chosen by the attacker and must not determine
	// the extension that later controls browser handling.
	contentType, ext, err := safeRemoteContentType(data)
	if err != nil {
		return "", err
	}

	if err := u.checkDeniedExtension(ctx, ext); err != nil {
		return "", err
	}

	s3Filename := fmt.Sprintf("%s/%s%s", kbID, uuid.New().String(), ext)

	putResp, err := u.s3Client.PutObject(
		ctx,
		u.s3Client.Bucket(),
		s3Filename,
		bytes.NewReader(data),
		int64(len(data)),
		minio.PutObjectOptions{
			ContentType: contentType,
		},
	)
	if err != nil {
		return "", fmt.Errorf("upload failed: %w", err)
	}

	return putResp.Key, nil
}

func safeRemoteContentType(data []byte) (string, string, error) {
	detected := http.DetectContentType(data)
	mediaType, _, err := mime.ParseMediaType(detected)
	if err != nil {
		return "", "", fmt.Errorf("unsupported remote file type")
	}

	switch mediaType {
	case "image/jpeg":
		return mediaType, ".jpg", nil
	case "image/png":
		return mediaType, ".png", nil
	case "image/gif":
		return mediaType, ".gif", nil
	case "image/webp":
		return mediaType, ".webp", nil
	default:
		return "", "", fmt.Errorf("unsupported remote file type: %s", mediaType)
	}
}

func isActiveScriptOrHTMLMime(rawMime string) bool {
	if rawMime == "" {
		return false
	}
	mediaType, _, err := mime.ParseMediaType(rawMime)
	if err != nil {
		mediaType = strings.TrimSpace(strings.SplitN(rawMime, ";", 2)[0])
	}
	switch strings.ToLower(mediaType) {
	case "text/html",
		"application/xhtml+xml",
		"text/javascript",
		"application/javascript",
		"application/x-javascript":
		return true
	default:
		return false
	}
}

// resolveSafeUploadContentType 结合字节嗅探与扩展名 MIME 推断安全 Content-Type：
// 1. 若字节嗅探或扩展名命中 HTML/JS 等可执行高危类型，一律降级为 application/octet-stream；
// 2. 否则优先取扩展名 MIME（保证 .svg / Office 文档不退化），未知扩展名回退字节嗅探。
func resolveSafeUploadContentType(ext string, head []byte) string {
	sniffed := http.DetectContentType(head)
	extMime := mime.TypeByExtension(ext)
	if isActiveScriptOrHTMLMime(sniffed) || isActiveScriptOrHTMLMime(extMime) {
		return "application/octet-stream"
	}
	if extMime != "" {
		return extMime
	}
	if sniffed != "" {
		return sniffed
	}
	return "application/octet-stream"
}

func (u *FileUsecase) httpClientForIPs(ips []net.IP) *http.Client {
	baseTransport := u.httpClient.Transport.(*http.Transport)
	transport := baseTransport.Clone()
	transport.DisableKeepAlives = true
	transport.DialContext = dialApprovedIPs(ips)

	return &http.Client{
		Transport:     transport,
		CheckRedirect: u.httpClient.CheckRedirect,
	}
}

// dialApprovedIPs 已提升为公共工具 utils.DialApprovedIPs，供其他按 URL 拉取的场景复用
func dialApprovedIPs(ips []net.IP) func(context.Context, string, string) (net.Conn, error) {
	return utils.DialApprovedIPs(ips)
}

// checkDeniedExtension checks if the file extension is in the denied list
func (u *FileUsecase) checkDeniedExtension(ctx context.Context, ext string) error {
	// Remove leading dot from extension
	ext = strings.TrimPrefix(ext, ".")
	if ext == "" {
		return nil
	}

	// Get denied extensions from system settings
	setting, err := u.systemSettingRepo.GetSystemSetting(ctx, consts.SystemSettingUpload)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		// 设置读取失败时 fail-closed：管理员配置的黑名单不能因后端抖动被绕过
		u.logger.Error("failed to get upload denied extensions setting", "error", err)
		return fmt.Errorf("upload denied extensions setting unavailable, upload rejected for safety")
	}

	var deniedSetting domain.UploadDeniedExtensionsSetting
	if err := json.Unmarshal(setting.Value, &deniedSetting); err != nil {
		// 设置格式损坏同样 fail-closed（宁拒绝不绕过，修复设置即可恢复上传）
		u.logger.Error("failed to unmarshal denied extensions setting", "error", err)
		return fmt.Errorf("upload denied extensions setting malformed, upload rejected for safety")
	}

	// Check if extension is denied
	for _, deniedExt := range deniedSetting.DeniedExtensions {
		if strings.EqualFold(ext, deniedExt) {
			return fmt.Errorf("file extension '.%s' is not allowed for upload", ext)
		}
	}

	return nil
}

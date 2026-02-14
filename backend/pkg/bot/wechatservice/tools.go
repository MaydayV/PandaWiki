package wechatservice

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"
)

// 读取 cursor，以客服账号的消息作为key，返回对应的cursor值
func getCursor(openKfId string) string {
	cursorValue, _ := KfCursors.Load(openKfId)
	cursor, _ := cursorValue.(string)
	return cursor
}

// 存储 cursor
func setCursor(openKfId, cursor string) {
	KfCursors.Store(openKfId, cursor)
}

func CheckSessionState(token, extrenaluserid, kfId string) (int, error) {
	var statusrequest struct {
		OpenKfId       string `json:"open_kfid"`
		ExternalUserid string `json:"external_userid"`
	}
	statusrequest.OpenKfId = kfId
	statusrequest.ExternalUserid = extrenaluserid
	// 将请求体转换为JSON
	jsonBody, err := json.Marshal(statusrequest)
	if err != nil {
		return 0, err
	}
	// 获取状态信息
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/kf/service_state/get?access_token=%s", token)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return 0, fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("读取响应失败: %v", err)
	}

	var response Status

	if err := json.Unmarshal(body, &response); err != nil {
		return 0, fmt.Errorf("解析响应失败: %v", err)
	}
	// 得到用户的状态
	if response.ErrCode != 0 {
		return 0, fmt.Errorf("获取会话状态失败: %s", response.ErrMsg)
	}
	return response.ServiceState, nil
}

func ChangeState(token, extrenaluserId, kfId string, state int, serviceId string) error {
	var changestate struct {
		OpenKfId       string `json:"open_kfid"`
		ExternalUserid string `json:"external_userid"`
		ServiceState   int    `json:"service_state"`
		ServicerUserId string `json:"servicer_userid"`
	}
	changestate.OpenKfId = kfId
	changestate.ExternalUserid = extrenaluserId
	changestate.ServiceState = state
	changestate.ServicerUserId = serviceId
	jsonBody, err := json.Marshal(changestate)
	if err != nil {
		return err
	}
	// 发送请求
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/kf/service_state/trans?access_token=%s", token)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return fmt.Errorf("发送请求失败: %v", err)
	}
	defer resp.Body.Close()

	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %v", err)
	}
	// 解析响应
	var response struct {
		ErrCode int    `json:"errcode"`
		ErrMsg  string `json:"errmsg"`
		MsgCode string `json:"msg_code"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("解析响应失败: %v", err)
	}
	// 得到用户的状态
	if response.ErrCode != 0 {
		return fmt.Errorf("改变用户状态失败: %s", response.ErrMsg)
	}
	return nil
}

func GetUserInfo(userid string, accessToken string) (*Customer, error) {
	userInfoRequest := UerInfoRequest{
		UserID:         []string{userid},
		SessionContext: 0,
	}
	// 请求获取用户信息的url
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/kf/customer/batchget?access_token=%s", accessToken)

	jsonBody, err := json.Marshal(userInfoRequest)
	if err != nil {
		return nil, err
	}
	// post获取用户的消息信息
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userInfo WechatCustomerResponse
	if err := json.Unmarshal(body, &userInfo); err != nil {
		return nil, err
	}

	if userInfo.ErrCode != 0 {
		return nil, fmt.Errorf("获取用户信息失败: %d, %s", userInfo.ErrCode, userInfo.ErrMsg)
	}

	return &userInfo.CustomerList[0], nil
}

// get image id
func GetUserImageID(accessToken, filePath string) (string, error) {
	UImageCache.Mutex.Lock()
	defer UImageCache.Mutex.Unlock()

	if UImageCache.ImageID != "" && (UImageCache.ImagePath == filePath) && time.Now().Before(UImageCache.ImageExpire.Add(-5*time.Minute)) {
		return UImageCache.ImageID, nil
	}

	// URL
	mediaID, err := UploadMediaFromURL(accessToken, filePath)

	if err != nil {
		return "", err
	}

	UImageCache.ImagePath = filePath
	UImageCache.ImageID = mediaID
	UImageCache.ImageExpire = time.Now().Add(72 * time.Hour) // 3 days
	return UImageCache.ImageID, nil
}

// get image id
func GetDefaultImageID(accessToken, ImageBase64 string) (string, error) {
	DImageCache.Mutex.Lock()
	defer DImageCache.Mutex.Unlock()

	if DImageCache.ImageID != "" && time.Now().Before(DImageCache.ImageExpire.Add(-5*time.Minute)) {
		return DImageCache.ImageID, nil
	}

	// Base64编码
	mediaID, err := UploadMediaFromBase64(accessToken, ImageBase64)

	if err != nil {
		return "", err
	}

	DImageCache.ImageID = mediaID
	DImageCache.ImageExpire = time.Now().Add(72 * time.Hour) // 3 days
	return DImageCache.ImageID, nil
}

// upload media to wechat server from URL
func UploadMediaFromURL(accessToken, fileURL string) (string, error) {
	// 处理URL
	resp, err := http.Get(fileURL)
	if err != nil {
		return "", fmt.Errorf("下载图片失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载图片失败，状态码: %d", resp.StatusCode)
	}

	reader := resp.Body
	fileName := "image.png" // 默认文件名

	// 从URL中提取文件名
	if u, err := url.Parse(fileURL); err == nil && u.Path != "" {
		if path.Base(u.Path) != "/" && path.Base(u.Path) != "." {
			fileName = path.Base(u.Path)
		}
	}

	return uploadMediaToWechat(accessToken, reader, fileName)
}

// upload media to wechat server from Base64
func UploadMediaFromBase64(accessToken, base64Data string) (string, error) {
	// 处理Base64编码的图片
	parts := strings.SplitN(base64Data, ",", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("无效的Base64图片数据")
	}

	// 解码Base64数据
	decodedData, err := base64.StdEncoding.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("解码Base64图片数据失败: %w", err)
	}

	reader := bytes.NewReader(decodedData)
	fileName := "image.png" // const

	return uploadMediaToWechat(accessToken, reader, fileName)
}

// upload media to wechat server - common function
func uploadMediaToWechat(accessToken string, reader io.Reader, fileName string) (string, error) {
	// 上传文件 req
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	part, err := writer.CreateFormFile("media", fileName)
	if err != nil {
		return "", err
	}

	// 将图片数据复制到表单中
	_, err = io.Copy(part, reader)
	if err != nil {
		return "", fmt.Errorf("复制图片数据失败: %w", err)
	}

	if err := writer.Close(); err != nil {
		return "", err
	}

	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/media/upload?access_token=%s&type=image", accessToken)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	client := &http.Client{}
	httpResp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer httpResp.Body.Close()

	var result MediaUploadResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&result); err != nil {
		return "", err
	}

	if result.ErrCode != 0 {
		return "", fmt.Errorf("上传失败: [%d] %s", result.ErrCode, result.ErrMsg)
	}

	return result.MediaID, nil
}

func getMsgs(accessToken string, msg *WeixinUserAskMsg) (*MsgRet, error) {
	var msgRet MsgRet
	// 拉取消息的路由
	url := fmt.Sprintf("https://qyapi.weixin.qq.com/cgi-bin/kf/sync_msg?access_token=%s", accessToken)
	cursor := getCursor(msg.OpenKfId)

	msgBody := MsgRequest{
		OpenKfid:    msg.OpenKfId,
		Token:       msg.Token,
		Limit:       1000,
		VoiceFormat: 0,
		Cursor:      cursor,
	}

	jsonBody, _ := json.Marshal(msgBody)

	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonBody)) // 得到对应的回复
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// 反序列化之后
	if err := json.Unmarshal([]byte(string(body)), &msgRet); err != nil {
		return nil, err
	}
	return &msgRet, nil
}

// markdowntotext
func MarkdowntoText(md string) string {
	md = regexp.MustCompile(`(?m)^#+\s*(.*)$`).ReplaceAllString(md, "$1")
	md = regexp.MustCompile(`\*\*([^*]+)\*\*`).ReplaceAllString(md, "$1")
	md = regexp.MustCompile(`(?m)^>\s*(.*)$`).ReplaceAllString(md, "【引用】$1")
	md = regexp.MustCompile(`(?m)^-{3,}$`).ReplaceAllString(md, "─────────")
	md = regexp.MustCompile(`\n{3,}`).ReplaceAllString(md, "\n\n")
	md = regexp.MustCompile(`\[\[(\d+)\]\([^)]+\)\]`).ReplaceAllString(md, "[$1]")
	md = regexp.MustCompile(`\[(\d+)\]\.\s*\[([^\]]+)\]\([^)]+\)`).ReplaceAllString(md, "[$1]. $2")
	md = regexp.MustCompile(`(?m)^【引用】\[(\d+)\].\s*([^\n(]+)\s*\([^)]+\)`).ReplaceAllString(md, "【引用】[$1]. $2")
	return strings.TrimSpace(md)
}

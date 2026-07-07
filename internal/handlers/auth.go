package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/shieldflow/shieldflow/internal/config"
	"github.com/shieldflow/shieldflow/internal/middleware"
	"github.com/shieldflow/shieldflow/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// AuthHandler 认证相关 handler 集合
type AuthHandler struct {
	cfg  *config.Config
	rdb  *redis.Client
}

// NewAuthHandler 构造 AuthHandler
func NewAuthHandler(cfg *config.Config, rdb *redis.Client) *AuthHandler {
	return &AuthHandler{cfg: cfg, rdb: rdb}
}

const (
	captchaKeyPrefix = "zy:auth:captcha:"
	captchaTTL       = 5 * time.Minute
	emailCodePrefix  = "zy:auth:emailcode:"
	emailCodeTTL     = 10 * time.Minute
)

// Login godoc
// POST /api/v1/auth/login
func (h *AuthHandler) Login(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		Account  string `json:"account"`  // 用户名 / 邮箱 / 手机号
		Password string `json:"password"`
		CaptchaID string `json:"captcha_id"`
		CaptchaCode string `json:"captcha_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}

	req.Account = strings.TrimSpace(req.Account)
	if req.Account == "" || req.Password == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "账号或密码不能为空", "data": nil})
		return
	}

	// 验证图片验证码
	if req.CaptchaID == "" || req.CaptchaCode == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "验证码不能为空", "data": nil})
		return
	}
	stored, err := h.rdb.Get(context.Background(), captchaKeyPrefix+req.CaptchaID).Result()
	if err != nil || strings.ToLower(stored) != strings.ToLower(req.CaptchaCode) {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "验证码错误或已过期", "data": nil})
		return
	}
	// 删除已使用的验证码
	h.rdb.Del(context.Background(), captchaKeyPrefix+req.CaptchaID)

	var user models.User
	if err := db.Where("username = ? OR email = ? OR phone = ?", req.Account, req.Account, req.Account).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "账号或密码错误", "data": nil})
		return
	}

	if user.Status != "active" {
		c.JSON(http.StatusOK, gin.H{"code": 403, "message": "账号已被禁用，请联系管理员", "data": nil})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 401, "message": "账号或密码错误", "data": nil})
		return
	}

	token, err := middleware.GenerateToken(&h.cfg.JWT, user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "生成 token 失败: " + err.Error(), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
				"role":     user.Role,
				"verified": user.Verified,
			},
		},
	})
}

// Register godoc
// POST /api/v1/auth/register
func (h *AuthHandler) Register(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		Username     string `json:"username"`
		Email        string `json:"email"`
		Phone        string `json:"phone"`
		Password     string `json:"password"`
		CaptchaID    string `json:"captcha_id"`
		CaptchaCode  string `json:"captcha_code"`
		EmailCode    string `json:"email_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}

	req.Username = strings.TrimSpace(req.Username)
	req.Email = strings.TrimSpace(req.Email)
	if req.Username == "" || req.Email == "" || req.Password == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "用户名/邮箱/密码不能为空", "data": nil})
		return
	}
	if len(req.Password) < 6 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "密码长度至少 6 位", "data": nil})
		return
	}

	// 验证图片验证码
	if req.CaptchaID == "" || req.CaptchaCode == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "图片验证码不能为空", "data": nil})
		return
	}
	stored, err := h.rdb.Get(context.Background(), captchaKeyPrefix+req.CaptchaID).Result()
	if err != nil || strings.ToLower(stored) != strings.ToLower(req.CaptchaCode) {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "图片验证码错误或已过期", "data": nil})
		return
	}
	h.rdb.Del(context.Background(), captchaKeyPrefix+req.CaptchaID)

	// 校验邮箱验证码
	if req.EmailCode == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "邮箱验证码不能为空", "data": nil})
		return
	}
	emailCode, err := h.rdb.Get(context.Background(), emailCodePrefix+req.Email).Result()
	if err != nil || emailCode != req.EmailCode {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "邮箱验证码错误或已过期", "data": nil})
		return
	}
	h.rdb.Del(context.Background(), emailCodePrefix+req.Email)

	// 检查重复
	var count int64
	db.Model(&models.User{}).Where("username = ? OR email = ?", req.Username, req.Email).Count(&count)
	if count > 0 {
		c.JSON(http.StatusOK, gin.H{"code": 409, "message": "用户名或邮箱已存在", "data": nil})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "密码加密失败: " + err.Error(), "data": nil})
		return
	}

	user := models.User{
		Username:     req.Username,
		Email:        req.Email,
		Phone:        req.Phone,
		PasswordHash: string(hash),
		Role:         "user",
		Status:       "active",
	}
	if err := db.Create(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "创建用户失败: " + err.Error(), "data": nil})
		return
	}

	token, err := middleware.GenerateToken(&h.cfg.JWT, user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "生成 token 失败: " + err.Error(), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
				"role":     user.Role,
			},
		},
	})
}

// Logout godoc
// POST /api/v1/auth/logout
func (h *AuthHandler) Logout(c *gin.Context) {
	// 客户端侧丢弃 token 即可；服务端可选将 token 加入黑名单
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// Captcha godoc
// GET /api/v1/auth/captcha
func (h *AuthHandler) Captcha(c *gin.Context) {
	captchaID := fmt.Sprintf("%d", time.Now().UnixNano())
	// 生成 4 位字母数字验证码
	const charset = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := make([]byte, 4)
	for i := range code {
		code[i] = charset[r.Intn(len(charset))]
	}
	codeStr := string(code)

	// 保存到 redis
	h.rdb.Set(context.Background(), captchaKeyPrefix+captchaID, codeStr, captchaTTL)

	// 生成 PNG 图片（简易实现）
	img := generateCaptchaImage(codeStr)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "验证码图片编码失败: " + err.Error(), "data": nil})
		return
	}
	base64Str := base64.StdEncoding.EncodeToString(buf.Bytes())

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"captcha_id": captchaID,
			"image":      "data:image/png;base64," + base64Str,
		},
	})
}

// VerifyCode godoc
// POST /api/v1/auth/verify-code (邮箱验证码登录)
func (h *AuthHandler) VerifyCode(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)

	var req struct {
		Email     string `json:"email"`
		EmailCode string `json:"email_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	req.Email = strings.TrimSpace(req.Email)
	if req.Email == "" || req.EmailCode == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "邮箱和验证码不能为空", "data": nil})
		return
	}

	stored, err := h.rdb.Get(context.Background(), emailCodePrefix+req.Email).Result()
	if err != nil || stored != req.EmailCode {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "验证码错误或已过期", "data": nil})
		return
	}
	h.rdb.Del(context.Background(), emailCodePrefix+req.Email)

	var user models.User
	if err := db.Where("email = ?", req.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "用户不存在", "data": nil})
		return
	}
	if user.Status != "active" {
		c.JSON(http.StatusOK, gin.H{"code": 403, "message": "账号已被禁用", "data": nil})
		return
	}

	token, err := middleware.GenerateToken(&h.cfg.JWT, user.ID, user.Username, user.Role)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "生成 token 失败: " + err.Error(), "data": nil})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"token": token,
			"user": gin.H{
				"id":       user.ID,
				"username": user.Username,
				"email":    user.Email,
				"role":     user.Role,
			},
		},
	})
}

// GetProfile godoc
// GET /api/v1/auth/profile
func (h *AuthHandler) GetProfile(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "用户不存在", "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": user})
}

// UpdateProfile godoc
// PUT /api/v1/auth/profile
func (h *AuthHandler) UpdateProfile(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var req struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		Phone    string `json:"phone"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}

	updates := map[string]interface{}{}
	if req.Username != "" {
		// 检查重复
		var cnt int64
		db.Model(&models.User{}).Where("username = ? AND id <> ?", req.Username, userID).Count(&cnt)
		if cnt > 0 {
			c.JSON(http.StatusOK, gin.H{"code": 409, "message": "用户名已存在", "data": nil})
			return
		}
		updates["username"] = req.Username
	}
	if req.Email != "" {
		var cnt int64
		db.Model(&models.User{}).Where("email = ? AND id <> ?", req.Email, userID).Count(&cnt)
		if cnt > 0 {
			c.JSON(http.StatusOK, gin.H{"code": 409, "message": "邮箱已存在", "data": nil})
			return
		}
		updates["email"] = req.Email
	}
	if req.Phone != "" {
		updates["phone"] = req.Phone
	}

	if len(updates) == 0 {
		c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
		return
	}
	if err := db.Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "更新失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// ChangePassword godoc
// PUT /api/v1/auth/password
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var req struct {
		OldPassword string `json:"old_password"`
		NewPassword string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	if len(req.NewPassword) < 6 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "新密码长度至少 6 位", "data": nil})
		return
	}

	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 404, "message": "用户不存在", "data": nil})
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "原密码错误", "data": nil})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "密码加密失败: " + err.Error(), "data": nil})
		return
	}
	if err := db.Model(&user).Update("password_hash", string(hash)).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "更新密码失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// Realname godoc
// POST /api/v1/auth/realname (实名认证)
func (h *AuthHandler) Realname(c *gin.Context) {
	db := c.MustGet("db").(*gorm.DB)
	userID := c.MustGet("user_id").(uint)

	var req struct {
		RealName string `json:"real_name"`
		IDCard   string `json:"id_card"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "请求参数错误: " + err.Error(), "data": nil})
		return
	}
	req.RealName = strings.TrimSpace(req.RealName)
	req.IDCard = strings.TrimSpace(req.IDCard)
	if req.RealName == "" || req.IDCard == "" {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "真实姓名和身份证号不能为空", "data": nil})
		return
	}
	if len(req.IDCard) != 18 {
		c.JSON(http.StatusOK, gin.H{"code": 400, "message": "身份证号格式不正确", "data": nil})
		return
	}

	// TODO: 接入第三方实名认证 API 校验，此处仅做基本保存
	if err := db.Model(&models.User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"real_name": req.RealName,
		"id_card":   req.IDCard,
		"verified":  true,
	}).Error; err != nil {
		c.JSON(http.StatusOK, gin.H{"code": 500, "message": "实名认证失败: " + err.Error(), "data": nil})
		return
	}
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": nil})
}

// --- 邮箱验证码发送辅助函数 (供 auth 或其他 handler 复用) ---

// SendEmailCode 发送邮箱验证码 (内部辅助函数，可被其他 handler 调用)
func (h *AuthHandler) SendEmailCode(email string) error {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	code := fmt.Sprintf("%06d", r.Intn(1000000))
	h.rdb.Set(context.Background(), emailCodePrefix+email, code, emailCodeTTL)
	// TODO: 接入邮件发送 SDK 实际投递
	_ = code
	return nil
}

// --- 验证码图片生成 (无外部依赖) ---

func generateCaptchaImage(text string) image.Image {
	width := 120
	height := 40
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	bg := color.RGBA{R: 240, G: 240, B: 240, A: 255}
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, bg)
		}
	}
	// 绘制干扰线
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := 0; i < 5; i++ {
		x1 := r.Intn(width)
		y1 := r.Intn(height)
		x2 := r.Intn(width)
		y2 := r.Intn(height)
		lineColor := color.RGBA{
			R: uint8(r.Intn(200)),
			G: uint8(r.Intn(200)),
			B: uint8(r.Intn(200)),
			A: 255,
		}
		drawLine(img, x1, y1, x2, y2, lineColor)
	}
	// 绘制字符（简易位图字体）
	startX := 15
	for i, ch := range text {
		drawChar(img, startX+i*25, 8, string(ch))
	}
	return img
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, col color.Color) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx, sy := 1, 1
	if x1 > x2 {
		sx = -1
	}
	if y1 > y2 {
		sy = -1
	}
	err := dx - dy
	for {
		img.Set(x1, y1, col)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// drawChar 在指定位置绘制单个字符 (5x7 位图字体)
func drawChar(img *image.RGBA, x, y int, ch string) {
	glyph, ok := glyphs[ch]
	if !ok {
		return
	}
	fg := color.RGBA{R: 30, G: 30, B: 30, A: 255}
	for row := 0; row < len(glyph); row++ {
		for col := 0; col < len(glyph[row]); col++ {
			if glyph[row][col] == 1 {
				for dy := 0; dy < 3; dy++ {
					for dx := 0; dx < 3; dx++ {
						img.Set(x+col*3+dx, y+row*3+dy, fg)
					}
				}
			}
		}
	}
}

// 简易 5x7 位图字体 (仅 A-Z 0-9 部分常用字符)
var glyphs = map[string][][]int{
	"A": {{0,1,1,1,0},{1,0,0,0,1},{1,0,0,0,1},{1,1,1,1,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1}},
	"B": {{1,1,1,1,0},{1,0,0,0,1},{1,0,0,0,1},{1,1,1,1,0},{1,0,0,0,1},{1,0,0,0,1},{1,1,1,1,0}},
	"C": {{0,1,1,1,1},{1,0,0,0,0},{1,0,0,0,0},{1,0,0,0,0},{1,0,0,0,0},{1,0,0,0,0},{0,1,1,1,1}},
	"D": {{1,1,1,1,0},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{1,1,1,1,0}},
	"E": {{1,1,1,1,1},{1,0,0,0,0},{1,0,0,0,0},{1,1,1,1,0},{1,0,0,0,0},{1,0,0,0,0},{1,1,1,1,1}},
	"F": {{1,1,1,1,1},{1,0,0,0,0},{1,0,0,0,0},{1,1,1,1,0},{1,0,0,0,0},{1,0,0,0,0},{1,0,0,0,0}},
	"G": {{0,1,1,1,1},{1,0,0,0,0},{1,0,0,0,0},{1,0,0,1,1},{1,0,0,0,1},{1,0,0,0,1},{0,1,1,1,1}},
	"H": {{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{1,1,1,1,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1}},
	"J": {{0,0,0,0,1},{0,0,0,0,1},{0,0,0,0,1},{0,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{0,1,1,1,1}},
	"K": {{1,0,0,0,1},{1,0,0,1,0},{1,0,1,0,0},{1,1,0,0,0},{1,0,1,0,0},{1,0,0,1,0},{1,0,0,0,1}},
	"L": {{1,0,0,0,0},{1,0,0,0,0},{1,0,0,0,0},{1,0,0,0,0},{1,0,0,0,0},{1,0,0,0,0},{1,1,1,1,1}},
	"M": {{1,0,0,0,1},{1,1,0,1,1},{1,0,1,0,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1}},
	"N": {{1,0,0,0,1},{1,1,0,0,1},{1,0,1,0,1},{1,0,0,1,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1}},
	"P": {{1,1,1,1,0},{1,0,0,0,1},{1,0,0,0,1},{1,1,1,1,0},{1,0,0,0,0},{1,0,0,0,0},{1,0,0,0,0}},
	"Q": {{0,1,1,1,0},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,1,0,1},{1,0,0,1,0},{0,1,1,0,1}},
	"R": {{1,1,1,1,0},{1,0,0,0,1},{1,0,0,0,1},{1,1,1,1,0},{1,0,1,0,0},{1,0,0,1,0},{1,0,0,0,1}},
	"S": {{0,1,1,1,1},{1,0,0,0,0},{1,0,0,0,0},{0,1,1,1,0},{0,0,0,0,1},{0,0,0,0,1},{1,1,1,1,0}},
	"T": {{1,1,1,1,1},{0,0,1,0,0},{0,0,1,0,0},{0,0,1,0,0},{0,0,1,0,0},{0,0,1,0,0},{0,0,1,0,0}},
	"U": {{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{0,1,1,1,0}},
	"V": {{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{0,1,0,1,0},{0,0,1,0,0}},
	"W": {{1,0,0,0,1},{1,0,0,0,1},{1,0,0,0,1},{1,0,1,0,1},{1,0,1,0,1},{1,1,0,1,1},{1,0,0,0,1}},
	"X": {{1,0,0,0,1},{1,0,0,0,1},{0,1,0,1,0},{0,0,1,0,0},{0,1,0,1,0},{1,0,0,0,1},{1,0,0,0,1}},
	"Y": {{1,0,0,0,1},{1,0,0,0,1},{0,1,0,1,0},{0,0,1,0,0},{0,0,1,0,0},{0,0,1,0,0},{0,0,1,0,0}},
	"Z": {{1,1,1,1,1},{0,0,0,0,1},{0,0,0,1,0},{0,0,1,0,0},{0,1,0,0,0},{1,0,0,0,0},{1,1,1,1,1}},
	"2": {{0,1,1,1,0},{1,0,0,0,1},{0,0,0,0,1},{0,0,0,1,0},{0,0,1,0,0},{0,1,0,0,0},{1,1,1,1,1}},
	"3": {{0,1,1,1,0},{1,0,0,0,1},{0,0,0,0,1},{0,1,1,1,0},{0,0,0,0,1},{1,0,0,0,1},{0,1,1,1,0}},
	"4": {{0,0,0,1,0},{0,0,1,1,0},{0,1,0,1,0},{1,0,0,1,0},{1,1,1,1,1},{0,0,0,1,0},{0,0,0,1,0}},
	"5": {{1,1,1,1,1},{1,0,0,0,0},{1,1,1,1,0},{0,0,0,0,1},{0,0,0,0,1},{1,0,0,0,1},{0,1,1,1,0}},
	"6": {{0,0,1,1,0},{0,1,0,0,0},{1,0,0,0,0},{1,1,1,1,0},{1,0,0,0,1},{1,0,0,0,1},{0,1,1,1,0}},
	"7": {{1,1,1,1,1},{0,0,0,0,1},{0,0,0,1,0},{0,0,1,0,0},{0,1,0,0,0},{0,1,0,0,0},{0,1,0,0,0}},
	"8": {{0,1,1,1,0},{1,0,0,0,1},{1,0,0,0,1},{0,1,1,1,0},{1,0,0,0,1},{1,0,0,0,1},{0,1,1,1,0}},
	"9": {{0,1,1,1,0},{1,0,0,0,1},{1,0,0,0,1},{0,1,1,1,1},{0,0,0,0,1},{0,0,0,1,0},{0,1,1,0,0}},
}

// 为了避免 strconv 引用未使用
var _ = strconv.Itoa
var _ = json.Marshal

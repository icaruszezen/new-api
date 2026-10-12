package controller

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
)

const (
	// 工单表单的请求体上限（4 张原图 + 1 MiB 文本和表单开销），在解析前强制
	ticketMaxRequestBytes = model.TicketMaxImagesPerMessage*service.TicketImageMaxUploadBytes + 1<<20
	ticketMultipartMemory = 8 << 20
	ticketStatusBodyBytes = 4 << 10
)

var (
	errTicketInvalidForm     = errors.New("invalid ticket form")
	errTicketRequestTooLarge = errors.New("ticket request body too large")
)

// 附件只会以服务端重新编码得到的类型输出
var ticketAttachmentExtensions = map[string]string{
	"image/png":  "png",
	"image/jpeg": "jpg",
}

var ticketErrorMessages = []struct {
	err  error
	key  string
	args map[string]any
}{
	{model.ErrTicketNotFound, i18n.MsgTicketNotFound, nil},
	{model.ErrTicketClosed, i18n.MsgTicketClosed, nil},
	{model.ErrTicketInvalidStatus, i18n.MsgTicketInvalidStatus, nil},
	{model.ErrTicketTooManyOpen, i18n.MsgTicketTooManyOpen, map[string]any{"Max": model.TicketMaxActivePerUser}},
	{model.ErrTicketMessageLimit, i18n.MsgTicketMessageLimit, map[string]any{"Max": model.TicketMaxMessagesPerTicket}},
	{model.ErrTicketAttachmentLimit, i18n.MsgTicketAttachmentLimit, map[string]any{"Max": model.TicketMaxAttachmentsPerTicket}},
	{model.ErrTicketDailyImageLimit, i18n.MsgTicketDailyImageLimit, map[string]any{"Max": model.TicketMaxImagesPerUserPerDay}},
	{service.ErrTicketSubjectRequired, i18n.MsgTicketSubjectRequired, nil},
	{service.ErrTicketSubjectTooLong, i18n.MsgTicketSubjectTooLong, map[string]any{"Max": model.TicketSubjectMaxRunes}},
	{service.ErrTicketContentRequired, i18n.MsgTicketContentRequired, nil},
	{service.ErrTicketContentTooLong, i18n.MsgTicketContentTooLong, map[string]any{"Max": model.TicketContentMaxRunes}},
	{service.ErrTicketTooManyImages, i18n.MsgTicketTooManyImages, map[string]any{"Max": model.TicketMaxImagesPerMessage}},
	{service.ErrTicketImageTooLarge, i18n.MsgTicketImageTooLarge, map[string]any{"MaxMB": service.TicketImageMaxUploadBytes >> 20}},
	{service.ErrTicketImageType, i18n.MsgTicketImageInvalidType, nil},
	{service.ErrTicketImageDimensions, i18n.MsgTicketImageDimensionsTooLarge, map[string]any{
		"MaxSide":       service.TicketImageMaxSide,
		"MaxMegapixels": service.TicketImageMaxPixels / 1_000_000,
		"MaxPixelsWan":  service.TicketImageMaxPixels / 10_000,
	}},
	{service.ErrTicketImageInvalid, i18n.MsgTicketImageInvalid, nil},
	{service.ErrTicketImageTooComplex, i18n.MsgTicketImageTooComplex, nil},
	{service.ErrTicketImageBusy, i18n.MsgTicketImageBusy, nil},
	{service.ErrTicketRequestInProgress, i18n.MsgTicketRequestInProgress, nil},
	{errTicketInvalidForm, i18n.MsgTicketInvalidForm, nil},
	{errTicketRequestTooLarge, i18n.MsgTicketRequestTooLarge, nil},
}

// respondTicketError 把工单相关错误映射为本地化提示；未识别的错误只记日志，不把数据库错误原文返回给前端。
// 附件不存在返回 HTTP 404，工单不属于当前用户时与附件不存在完全一致。
func respondTicketError(c *gin.Context, err error) {
	if errors.Is(err, model.ErrTicketAttachmentNotFound) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": common.TranslateMessage(c, i18n.MsgTicketAttachmentNotFound),
		})
		return
	}
	for _, item := range ticketErrorMessages {
		if errors.Is(err, item.err) {
			common.ApiErrorI18n(c, item.key, item.args)
			return
		}
	}
	// 客户端在图片处理期间断开，不是服务端错误
	if errors.Is(err, context.Canceled) {
		c.Abort()
		return
	}
	logger.LogError(c, fmt.Sprintf("ticket request failed: %v", err))
	common.ApiErrorI18n(c, i18n.MsgDatabaseError)
}

// parseTicketId 解析路径中的正整数 id，非法时返回 0
func parseTicketId(raw string) int {
	id, err := strconv.Atoi(raw)
	if err != nil || id <= 0 {
		return 0
	}
	return id
}

// parseOptionalTicketStatus 解析可选的状态参数：空字符串表示未指定（返回 0），否则必须是 1..4
func parseOptionalTicketStatus(raw string) (int, error) {
	if raw == "" {
		return 0, nil
	}
	status, err := strconv.Atoi(raw)
	if err != nil || !model.IsValidTicketStatus(status) {
		return 0, model.ErrTicketInvalidStatus
	}
	return status, nil
}

// readTicketForm 解析工单的 multipart 表单并读出全部图片字节。请求体在解析前限制大小；
// 只读取字段名为 files 的文件内容，客户端提供的文件名和 Content-Type 一律忽略。
// 文本字段解析后可通过 c.Request.PostFormValue 读取。
func readTicketForm(c *gin.Context) ([][]byte, error) {
	mediaType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || mediaType != "multipart/form-data" {
		return nil, errTicketInvalidForm
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, ticketMaxRequestBytes)
	if err := c.Request.ParseMultipartForm(ticketMultipartMemory); err != nil {
		if common.IsRequestBodyTooLargeError(err) {
			return nil, errTicketRequestTooLarge
		}
		return nil, errTicketInvalidForm
	}
	form := c.Request.MultipartForm
	defer form.RemoveAll()

	headers := form.File["files"]
	if len(headers) > model.TicketMaxImagesPerMessage {
		return nil, service.ErrTicketTooManyImages
	}
	images := make([][]byte, 0, len(headers))
	for _, header := range headers {
		file, err := header.Open()
		if err != nil {
			return nil, errTicketInvalidForm
		}
		data, err := io.ReadAll(io.LimitReader(file, service.TicketImageMaxUploadBytes+1))
		_ = file.Close()
		if err != nil {
			return nil, errTicketInvalidForm
		}
		if len(data) > service.TicketImageMaxUploadBytes {
			return nil, service.ErrTicketImageTooLarge
		}
		images = append(images, data)
	}
	return images, nil
}

// writeTicketAttachment 输出附件图片：类型取自服务端白名单，禁止嗅探，并用沙箱 CSP
// 防止图片地址被当作页面打开时执行任何内容。
func writeTicketAttachment(c *gin.Context, ticketId int, attachment *model.TicketAttachment) {
	extension, ok := ticketAttachmentExtensions[attachment.MimeType]
	if !ok {
		respondTicketError(c, model.ErrTicketAttachmentNotFound)
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'; img-src 'self' data: blob:; sandbox")
	c.Header("Content-Disposition", fmt.Sprintf(`inline; filename="ticket-%d-%d.%s"`, ticketId, attachment.Id, extension))
	c.Header("Cache-Control", "private, no-store")
	c.Header("Cross-Origin-Resource-Policy", "same-origin")
	c.Data(http.StatusOK, attachment.MimeType, attachment.Data)
}

// GetSelfTickets GET /api/ticket/self
func GetSelfTickets(c *gin.Context) {
	status, err := parseOptionalTicketStatus(c.Query("status"))
	if err != nil {
		respondTicketError(c, err)
		return
	}
	pageInfo := common.GetPageQuery(c)
	tickets, total, err := model.GetUserTickets(c.GetInt("id"), status, pageInfo)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tickets)
	common.ApiSuccess(c, pageInfo)
}

// GetSelfTicketUnreadCount GET /api/ticket/self/unread
func GetSelfTicketUnreadCount(c *gin.Context) {
	count, err := model.CountUserUnreadTickets(c.GetInt("id"))
	if err != nil {
		respondTicketError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"count": count})
}

// CreateSelfTicket POST /api/ticket/self
func CreateSelfTicket(c *gin.Context) {
	// 读取请求体前占用名额：同一用户同时只处理一个写请求
	release, err := service.AcquireTicketUserWrite(c.GetInt("id"))
	if err != nil {
		respondTicketError(c, err)
		return
	}
	defer release()
	images, err := readTicketForm(c)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	ticket, err := service.CreateTicket(
		c.Request.Context(),
		c.GetInt("id"),
		c.Request.PostFormValue("subject"),
		c.Request.PostFormValue("content"),
		images,
	)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	common.ApiSuccess(c, ticket)
}

// GetSelfTicket GET /api/ticket/self/:id
func GetSelfTicket(c *gin.Context) {
	ticketId := parseTicketId(c.Param("id"))
	if ticketId == 0 {
		respondTicketError(c, model.ErrTicketNotFound)
		return
	}
	detail, err := model.GetUserTicketDetail(c.GetInt("id"), ticketId)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	common.ApiSuccess(c, detail)
}

// ReplySelfTicket POST /api/ticket/self/:id/reply
func ReplySelfTicket(c *gin.Context) {
	ticketId := parseTicketId(c.Param("id"))
	if ticketId == 0 {
		respondTicketError(c, model.ErrTicketNotFound)
		return
	}
	release, err := service.AcquireTicketUserWrite(c.GetInt("id"))
	if err != nil {
		respondTicketError(c, err)
		return
	}
	defer release()
	images, err := readTicketForm(c)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	message, err := service.ReplyTicketAsUser(c.Request.Context(), c.GetInt("id"), ticketId, c.Request.PostFormValue("content"), images)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	common.ApiSuccess(c, message)
}

// CloseSelfTicket POST /api/ticket/self/:id/close
func CloseSelfTicket(c *gin.Context) {
	ticketId := parseTicketId(c.Param("id"))
	if ticketId == 0 {
		respondTicketError(c, model.ErrTicketNotFound)
		return
	}
	ticket, err := model.CloseUserTicket(c.GetInt("id"), ticketId)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	common.ApiSuccess(c, ticket)
}

// GetSelfTicketAttachment GET /api/ticket/self/:id/attachments/:attachment_id
func GetSelfTicketAttachment(c *gin.Context) {
	ticketId := parseTicketId(c.Param("id"))
	attachmentId := parseTicketId(c.Param("attachment_id"))
	if ticketId == 0 || attachmentId == 0 {
		respondTicketError(c, model.ErrTicketAttachmentNotFound)
		return
	}
	attachment, err := model.GetUserTicketAttachment(c.GetInt("id"), ticketId, attachmentId)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	writeTicketAttachment(c, ticketId, attachment)
}

// AdminGetTickets GET /api/ticket/admin
func AdminGetTickets(c *gin.Context) {
	status, err := parseOptionalTicketStatus(c.Query("status"))
	if err != nil {
		respondTicketError(c, err)
		return
	}
	pageInfo := common.GetPageQuery(c)
	tickets, total, err := model.GetAdminTickets(status, c.Query("keyword"), pageInfo)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(tickets)
	common.ApiSuccess(c, pageInfo)
}

// AdminGetTicketStats GET /api/ticket/admin/stats
func AdminGetTicketStats(c *gin.Context) {
	stats, err := model.GetTicketStats()
	if err != nil {
		respondTicketError(c, err)
		return
	}
	common.ApiSuccess(c, stats)
}

// AdminGetTicket GET /api/ticket/admin/:id
func AdminGetTicket(c *gin.Context) {
	ticketId := parseTicketId(c.Param("id"))
	if ticketId == 0 {
		respondTicketError(c, model.ErrTicketNotFound)
		return
	}
	detail, err := model.GetAdminTicketDetail(ticketId)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	common.ApiSuccess(c, detail)
}

// AdminReplyTicket POST /api/ticket/admin/:id/reply
func AdminReplyTicket(c *gin.Context) {
	ticketId := parseTicketId(c.Param("id"))
	if ticketId == 0 {
		respondTicketError(c, model.ErrTicketNotFound)
		return
	}
	images, err := readTicketForm(c)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	status, err := parseOptionalTicketStatus(c.Request.PostFormValue("status"))
	if err != nil {
		respondTicketError(c, err)
		return
	}
	ticket, message, err := service.ReplyTicketAsAdmin(c.Request.Context(), c.GetInt("id"), ticketId, c.Request.PostFormValue("content"), images, status)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	message.AuthorName = c.GetString("username")
	recordManageAuditFor(c, ticket.UserId, "ticket.reply", map[string]interface{}{"id": ticket.Id})
	if status != 0 {
		recordManageAuditFor(c, ticket.UserId, "ticket.status", map[string]interface{}{"id": ticket.Id, "status": ticket.Status})
	}
	common.ApiSuccess(c, message)
}

// AdminUpdateTicketStatus PUT /api/ticket/admin/:id/status
func AdminUpdateTicketStatus(c *gin.Context) {
	ticketId := parseTicketId(c.Param("id"))
	if ticketId == 0 {
		respondTicketError(c, model.ErrTicketNotFound)
		return
	}
	var request struct {
		Status int `json:"status"`
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, ticketStatusBodyBytes)
	if err := common.DecodeJson(c.Request.Body, &request); err != nil {
		respondTicketError(c, model.ErrTicketInvalidStatus)
		return
	}
	ticket, err := model.SetTicketStatus(ticketId, request.Status)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	recordManageAuditFor(c, ticket.UserId, "ticket.status", map[string]interface{}{"id": ticket.Id, "status": ticket.Status})
	common.ApiSuccess(c, ticket)
}

// AdminGetTicketAttachment GET /api/ticket/admin/:id/attachments/:attachment_id
func AdminGetTicketAttachment(c *gin.Context) {
	ticketId := parseTicketId(c.Param("id"))
	attachmentId := parseTicketId(c.Param("attachment_id"))
	if ticketId == 0 || attachmentId == 0 {
		respondTicketError(c, model.ErrTicketAttachmentNotFound)
		return
	}
	attachment, err := model.GetTicketAttachment(ticketId, attachmentId)
	if err != nil {
		respondTicketError(c, err)
		return
	}
	writeTicketAttachment(c, ticketId, attachment)
}

package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
)

var (
	ErrTicketSubjectRequired = errors.New("ticket subject is required")
	ErrTicketSubjectTooLong  = errors.New("ticket subject is too long")
	ErrTicketContentRequired = errors.New("ticket content is required")
	ErrTicketContentTooLong  = errors.New("ticket content is too long")
	ErrTicketTooManyImages   = errors.New("too many ticket images")

	ErrTicketRequestInProgress = errors.New("ticket request already in progress")
)

// 图片解码尝试按用户在滚动 24 小时内计数（本节点内存）。存储额度只统计成功保存的图片，
// 而失败的请求（如最后一张图损坏）同样消耗了解码资源；预算是存储额度的 2 倍，给正常重试留余量。
const (
	ticketImageDecodeWindow = 24 * time.Hour
	ticketImageDecodeBudget = 2 * model.TicketMaxImagesPerUserPerDay
)

var ticketImageDecodeAttempts = &common.InMemoryRateLimiter{}

// 正在处理工单写请求的用户（本节点）。同一用户同时只处理一个创建/回复请求，单个账号无法在内存中
// 排队大量请求体和图片解码，同一用户的额度检查也因此串行执行。
var ticketUserWrites sync.Map

var ticketNewlineReplacer = strings.NewReplacer("\r\n", "\n", "\r", "\n")

// SanitizeTicketContent 规范化用户提交的工单正文：去掉非法 UTF-8，统一换行为 \n，
// 删除除 \n、\t 以外的 C0/C1 控制字符以及 Unicode 双向控制字符（防止显示欺骗），再去掉首尾空白。
// 结果按纯文本存储和展示。
func SanitizeTicketContent(raw string) string {
	text := ticketNewlineReplacer.Replace(strings.ToValidUTF8(raw, ""))
	text = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\t' {
			return r
		}
		if unicode.IsControl(r) || unicode.Is(unicode.Bidi_Control, r) {
			return -1
		}
		return r
	}, text)
	return strings.TrimSpace(text)
}

// SanitizeTicketSubject 在正文规则基础上把换行、制表符等连续空白合并为单个空格，标题始终是单行。
func SanitizeTicketSubject(raw string) string {
	return strings.Join(strings.Fields(SanitizeTicketContent(raw)), " ")
}

// AcquireTicketUserWrite 占用用户的工单写入名额，已有请求在处理时返回 ErrTicketRequestInProgress。
// 成功时必须在请求结束后调用返回的 release。
func AcquireTicketUserWrite(userId int) (release func(), err error) {
	if _, busy := ticketUserWrites.LoadOrStore(userId, struct{}{}); busy {
		return nil, ErrTicketRequestInProgress
	}
	return func() { ticketUserWrites.Delete(userId) }, nil
}

// CreateTicket 校验并创建工单。图片在开启数据库事务前完成解码和重新编码。
func CreateTicket(ctx context.Context, userId int, subject string, content string, images [][]byte) (*model.Ticket, error) {
	subject = SanitizeTicketSubject(subject)
	if subject == "" {
		return nil, ErrTicketSubjectRequired
	}
	if utf8.RuneCountInString(subject) > model.TicketSubjectMaxRunes {
		return nil, ErrTicketSubjectTooLong
	}
	content, err := checkTicketContent(content, len(images), true)
	if err != nil {
		return nil, err
	}
	if err := model.CheckTicketUserLimits(userId, true, len(images)); err != nil {
		return nil, err
	}
	if err := chargeTicketImageDecodes(userId, len(images)); err != nil {
		return nil, err
	}
	attachments, err := sanitizeTicketImages(ctx, images)
	if err != nil {
		return nil, err
	}
	return model.CreateTicket(userId, subject, content, attachments)
}

// ReplyTicketAsUser 用户回复自己的工单。工单归属、关闭状态、单个工单上限和图片额度先做一次廉价检查，
// 不满足时不再解码图片；写事务内会再次校验。
func ReplyTicketAsUser(ctx context.Context, userId int, ticketId int, content string, images [][]byte) (*model.TicketMessage, error) {
	content, err := checkTicketContent(content, len(images), false)
	if err != nil {
		return nil, err
	}
	ticket, err := model.GetUserTicket(userId, ticketId)
	if err != nil {
		return nil, err
	}
	if ticket.Status == model.TicketStatusClosed {
		return nil, model.ErrTicketClosed
	}
	if err := model.CheckTicketReplyLimits(ticket, len(images)); err != nil {
		return nil, err
	}
	if err := model.CheckTicketUserLimits(userId, false, len(images)); err != nil {
		return nil, err
	}
	if err := chargeTicketImageDecodes(userId, len(images)); err != nil {
		return nil, err
	}
	attachments, err := sanitizeTicketImages(ctx, images)
	if err != nil {
		return nil, err
	}
	return model.AddUserTicketReply(userId, ticketId, content, attachments)
}

// ReplyTicketAsAdmin 客服回复工单；status 为 0 表示按默认规则流转状态。
// 管理员上传的图片同样经过重新编码，但不受普通用户的 24 小时图片额度限制。
func ReplyTicketAsAdmin(ctx context.Context, adminId int, ticketId int, content string, images [][]byte, status int) (*model.Ticket, *model.TicketMessage, error) {
	content, err := checkTicketContent(content, len(images), false)
	if err != nil {
		return nil, nil, err
	}
	if status != 0 && !model.IsValidTicketStatus(status) {
		return nil, nil, model.ErrTicketInvalidStatus
	}
	ticket, err := model.GetTicket(ticketId)
	if err != nil {
		return nil, nil, err
	}
	if err := model.CheckTicketReplyLimits(ticket, len(images)); err != nil {
		return nil, nil, err
	}
	attachments, err := sanitizeTicketImages(ctx, images)
	if err != nil {
		return nil, nil, err
	}
	return model.AddAdminTicketReply(adminId, ticketId, content, attachments, status)
}

// checkTicketContent 清洗并校验消息正文；contentRequired 为 false 时，带图片的消息正文可以为空。
func checkTicketContent(raw string, imageCount int, contentRequired bool) (string, error) {
	if imageCount > model.TicketMaxImagesPerMessage {
		return "", ErrTicketTooManyImages
	}
	content := SanitizeTicketContent(raw)
	if content == "" && (contentRequired || imageCount == 0) {
		return "", ErrTicketContentRequired
	}
	if utf8.RuneCountInString(content) > model.TicketContentMaxRunes {
		return "", ErrTicketContentTooLong
	}
	return content, nil
}

// chargeTicketImageDecodes 在解码前为用户记入即将解码的图片数，超出滚动 24 小时的尝试预算时
// 返回 ErrTicketDailyImageLimit。
func chargeTicketImageDecodes(userId int, imageCount int) error {
	ticketImageDecodeAttempts.Init(ticketImageDecodeWindow)
	key := fmt.Sprintf("ticket-image-decode:%d", userId)
	for range imageCount {
		if !ticketImageDecodeAttempts.Request(key, ticketImageDecodeBudget, int64(ticketImageDecodeWindow/time.Second)) {
			return model.ErrTicketDailyImageLimit
		}
	}
	return nil
}

func sanitizeTicketImages(ctx context.Context, images [][]byte) ([]*model.TicketAttachment, error) {
	attachments := make([]*model.TicketAttachment, 0, len(images))
	for _, raw := range images {
		// 客户端已断开时不再处理剩余图片
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		sanitized, err := SanitizeTicketImage(ctx, raw)
		if err != nil {
			return nil, err
		}
		attachments = append(attachments, &model.TicketAttachment{
			MimeType: sanitized.MimeType,
			Size:     len(sanitized.Data),
			Width:    sanitized.Width,
			Height:   sanitized.Height,
			Sha256:   sanitized.Sha256,
			Data:     sanitized.Data,
		})
	}
	return attachments, nil
}

package model

import (
	"errors"
	"math/rand/v2"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"

	"gorm.io/gorm"
)

// 工单状态，数值与前端共用
const (
	TicketStatusPending    = 1 // 待处理：新建或用户追加回复，等待客服处理
	TicketStatusProcessing = 2 // 处理中：客服已回复或正在处理
	TicketStatusResolved   = 3 // 已解决
	TicketStatusClosed     = 4 // 已关闭：用户不能再回复，管理员仍可改为其他状态重新打开
)

const (
	TicketRoleUser  = "user"
	TicketRoleAdmin = "admin"
)

// 工单业务限制；图片解码/编码相关的限制见 service/ticket_image.go
const (
	TicketSubjectMaxRunes         = 100
	TicketContentMaxRunes         = 5000
	TicketMaxImagesPerMessage     = 4
	TicketMaxActivePerUser        = 5
	TicketMaxMessagesPerTicket    = 200
	TicketMaxAttachmentsPerTicket = 40
	TicketMaxImagesPerUserPerDay  = 40
	ticketImageWindowSeconds      = 24 * 60 * 60
	ticketTransactionAttempts     = 3
)

var (
	ErrTicketNotFound           = errors.New("ticket not found")
	ErrTicketClosed             = errors.New("ticket is closed")
	ErrTicketInvalidStatus      = errors.New("invalid ticket status")
	ErrTicketTooManyOpen        = errors.New("too many open tickets")
	ErrTicketMessageLimit       = errors.New("ticket message limit reached")
	ErrTicketAttachmentLimit    = errors.New("ticket attachment limit reached")
	ErrTicketDailyImageLimit    = errors.New("daily ticket image limit reached")
	ErrTicketAttachmentNotFound = errors.New("ticket attachment not found")
)

type Ticket struct {
	Id            int    `json:"id"`
	UserId        int    `json:"user_id" gorm:"index"`
	Subject       string `json:"subject" gorm:"type:varchar(255);not null"`
	Status        int    `json:"status" gorm:"type:int;default:1;index"`
	LastReplyRole string `json:"last_reply_role" gorm:"type:varchar(16);default:'user'"`
	MessageCount  int    `json:"message_count" gorm:"type:int;default:0"`
	// UserUnread 客服回复或修改状态后、用户尚未查看
	UserUnread bool `json:"user_unread"`
	// AdminUnread 用户新建或回复后、管理员尚未查看
	AdminUnread bool   `json:"admin_unread"`
	CreatedTime int64  `json:"created_time" gorm:"bigint"`
	UpdatedTime int64  `json:"updated_time" gorm:"bigint;index"`
	ClosedTime  int64  `json:"closed_time" gorm:"bigint;default:0"`
	Username    string `json:"username,omitempty" gorm:"-"` // 仅管理端填充
}

func (Ticket) TableName() string {
	return "tickets"
}

type TicketMessage struct {
	Id       int `json:"id"`
	TicketId int `json:"ticket_id" gorm:"index"`
	// UserId 是作者 id；作者可能是管理员，因此不对工单所有者暴露
	UserId      int                `json:"-" gorm:"index"`
	AuthorRole  string             `json:"author_role" gorm:"type:varchar(16)"`
	Content     string             `json:"content" gorm:"type:text"`
	CreatedTime int64              `json:"created_time" gorm:"bigint"`
	AuthorName  string             `json:"author_name,omitempty" gorm:"-"` // 仅管理端填充
	Attachments []TicketAttachment `json:"attachments" gorm:"-"`
}

func (TicketMessage) TableName() string {
	return "ticket_messages"
}

// TicketAttachment 只保存服务端重新编码后的图片字节，不保存原始文件名和客户端声明的类型
type TicketAttachment struct {
	Id          int    `json:"id"`
	TicketId    int    `json:"ticket_id" gorm:"index"`
	MessageId   int    `json:"message_id" gorm:"index"`
	UserId      int    `json:"-" gorm:"index:idx_ticket_attachment_user_time,priority:1"`
	MimeType    string `json:"mime_type" gorm:"type:varchar(32)"`
	Size        int    `json:"size"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Sha256      string `json:"-" gorm:"type:varchar(64)"`
	Data        []byte `json:"-"`
	CreatedTime int64  `json:"created_time" gorm:"bigint;index:idx_ticket_attachment_user_time,priority:2"`
}

func (TicketAttachment) TableName() string {
	return "ticket_attachments"
}

type TicketDetail struct {
	Ticket   *Ticket          `json:"ticket"`
	Messages []*TicketMessage `json:"messages"`
}

type TicketUser struct {
	Id          int    `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Group       string `json:"group"`
}

type AdminTicketDetail struct {
	TicketDetail
	User *TicketUser `json:"user"`
}

type TicketStats struct {
	Pending    int64 `json:"pending"`
	Processing int64 `json:"processing"`
	Resolved   int64 `json:"resolved"`
	Closed     int64 `json:"closed"`
	Unread     int64 `json:"unread"`
}

// 附件列表只取元数据列，图片字节只在单个附件下载时读取
var ticketAttachmentMetaColumns = []string{"id", "ticket_id", "message_id", "mime_type", "size", "width", "height", "created_time"}

var ticketLikeEscaper = strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")

func IsValidTicketStatus(status int) bool {
	return status >= TicketStatusPending && status <= TicketStatusClosed
}

func ticketLookupError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrTicketNotFound
	}
	return err
}

// CheckTicketUserLimits 检查用户的活跃工单数（opensTicket 时）和滚动 24 小时图片数。
// 写事务内会再次检查；这里供服务层在解码图片前提前拒绝，额度用尽时不再消耗解码资源。
func CheckTicketUserLimits(userId int, opensTicket bool, imageCount int) error {
	return checkTicketUserLimits(DB, userId, opensTicket, imageCount, common.GetTimestamp())
}

func checkTicketUserLimits(tx *gorm.DB, userId int, opensTicket bool, imageCount int, now int64) error {
	if opensTicket {
		var active int64
		err := tx.Model(&Ticket{}).
			Where("user_id = ? AND status IN ?", userId, []int{TicketStatusPending, TicketStatusProcessing}).
			Count(&active).Error
		if err != nil {
			return err
		}
		if active >= TicketMaxActivePerUser {
			return ErrTicketTooManyOpen
		}
	}
	if imageCount > 0 {
		var images int64
		err := tx.Model(&TicketAttachment{}).
			Where("user_id = ? AND created_time > ?", userId, now-ticketImageWindowSeconds).
			Count(&images).Error
		if err != nil {
			return err
		}
		if images+int64(imageCount) > TicketMaxImagesPerUserPerDay {
			return ErrTicketDailyImageLimit
		}
	}
	return nil
}

// CheckTicketReplyLimits 检查单个工单的消息数和附件总数上限。写事务内会再次检查；
// 这里供服务层在解码图片前提前拒绝。
func CheckTicketReplyLimits(ticket *Ticket, imageCount int) error {
	if ticket.MessageCount >= TicketMaxMessagesPerTicket {
		return ErrTicketMessageLimit
	}
	return checkTicketAttachmentLimit(DB, ticket.Id, imageCount)
}

func checkTicketAttachmentLimit(tx *gorm.DB, ticketId int, imageCount int) error {
	if imageCount == 0 {
		return nil
	}
	var existing int64
	if err := tx.Model(&TicketAttachment{}).Where("ticket_id = ?", ticketId).Count(&existing).Error; err != nil {
		return err
	}
	if existing+int64(imageCount) > TicketMaxAttachmentsPerTicket {
		return ErrTicketAttachmentLimit
	}
	return nil
}

// runTicketTransaction 执行工单写事务。工单事务都是先读后写：SQLite 上两个重叠的事务都持有读锁时，
// 后申请写锁的一方会立即得到 SQLITE_BUSY（busy_timeout 对这种锁升级冲突无效），此时整体重试。
// fn 可能被执行多次，不能依赖上一次执行留下的状态。
func runTicketTransaction(fn func(tx *gorm.DB) error) error {
	var err error
	for attempt := 1; attempt <= ticketTransactionAttempts; attempt++ {
		err = DB.Transaction(fn)
		if err == nil || attempt == ticketTransactionAttempts || !common.UsingMainDatabase(common.DatabaseTypeSQLite) {
			break
		}
		if message := err.Error(); !strings.Contains(message, "database is locked") && !strings.Contains(message, "database table is locked") {
			break
		}
		// 随机退避，避免同时失败的事务再次同时重试
		time.Sleep(rand.N(time.Duration(attempt) * 100 * time.Millisecond))
	}
	return err
}

// insertTicketMessage 写入一条消息及其附件，并检查单个工单的附件总数上限。
// 附件逐条插入，避免多张图片拼成一条超过 MySQL max_allowed_packet 的语句。
func insertTicketMessage(tx *gorm.DB, ticketId int, authorId int, role string, content string, attachments []*TicketAttachment, now int64) (*TicketMessage, error) {
	if err := checkTicketAttachmentLimit(tx, ticketId, len(attachments)); err != nil {
		return nil, err
	}
	message := &TicketMessage{
		TicketId:    ticketId,
		UserId:      authorId,
		AuthorRole:  role,
		Content:     content,
		CreatedTime: now,
		Attachments: make([]TicketAttachment, 0, len(attachments)),
	}
	if err := tx.Create(message).Error; err != nil {
		return nil, err
	}
	for _, attachment := range attachments {
		attachment.Id = 0 // 事务重试时清掉上一次回滚前分配的 id
		attachment.TicketId = ticketId
		attachment.MessageId = message.Id
		attachment.UserId = authorId
		attachment.CreatedTime = now
		if err := tx.Create(attachment).Error; err != nil {
			return nil, err
		}
		meta := *attachment
		meta.Data = nil
		message.Attachments = append(message.Attachments, meta)
	}
	return message, nil
}

// applyTicketStatus 修改状态并维护 closed_time：进入关闭状态时记录时间，离开关闭状态时清零。
func applyTicketStatus(ticket *Ticket, status int, now int64, updates map[string]any) {
	if status == TicketStatusClosed && ticket.Status != TicketStatusClosed {
		ticket.ClosedTime = now
	} else if status != TicketStatusClosed && ticket.Status == TicketStatusClosed {
		ticket.ClosedTime = 0
	}
	ticket.Status = status
	updates["status"] = status
	updates["closed_time"] = ticket.ClosedTime
}

// CreateTicket 创建工单和首条消息；attachments 必须是服务层已重新编码过的图片。
func CreateTicket(userId int, subject string, content string, attachments []*TicketAttachment) (*Ticket, error) {
	now := common.GetTimestamp()
	ticket := &Ticket{
		UserId:        userId,
		Subject:       subject,
		Status:        TicketStatusPending,
		LastReplyRole: TicketRoleUser,
		MessageCount:  1,
		AdminUnread:   true,
		CreatedTime:   now,
		UpdatedTime:   now,
	}
	err := runTicketTransaction(func(tx *gorm.DB) error {
		if err := checkTicketUserLimits(tx, userId, true, len(attachments), now); err != nil {
			return err
		}
		ticket.Id = 0 // 事务重试时清掉上一次回滚前分配的 id
		if err := tx.Create(ticket).Error; err != nil {
			return err
		}
		_, err := insertTicketMessage(tx, ticket.Id, userId, TicketRoleUser, content, attachments, now)
		return err
	})
	if err != nil {
		return nil, err
	}
	return ticket, nil
}

// AddUserTicketReply 用户追加回复：已关闭的工单不能回复，其余状态回到待处理。
func AddUserTicketReply(userId int, ticketId int, content string, attachments []*TicketAttachment) (*TicketMessage, error) {
	now := common.GetTimestamp()
	var message *TicketMessage
	err := runTicketTransaction(func(tx *gorm.DB) error {
		var ticket Ticket
		if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", ticketId, userId).Take(&ticket).Error; err != nil {
			return ticketLookupError(err)
		}
		if ticket.Status == TicketStatusClosed {
			return ErrTicketClosed
		}
		if ticket.MessageCount >= TicketMaxMessagesPerTicket {
			return ErrTicketMessageLimit
		}
		if err := checkTicketUserLimits(tx, userId, false, len(attachments), now); err != nil {
			return err
		}
		var err error
		message, err = insertTicketMessage(tx, ticket.Id, userId, TicketRoleUser, content, attachments, now)
		if err != nil {
			return err
		}
		return tx.Model(&Ticket{}).Where("id = ?", ticket.Id).Updates(map[string]any{
			"status":          TicketStatusPending,
			"last_reply_role": TicketRoleUser,
			"admin_unread":    true,
			"message_count":   gorm.Expr("message_count + ?", 1),
			"updated_time":    now,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	return message, nil
}

// AddAdminTicketReply 客服回复，任何状态都可回复。status 为 0 时自动处理：
// 待处理变为处理中，其余状态不变；否则改为指定状态。
func AddAdminTicketReply(adminId int, ticketId int, content string, attachments []*TicketAttachment, status int) (*Ticket, *TicketMessage, error) {
	if status != 0 && !IsValidTicketStatus(status) {
		return nil, nil, ErrTicketInvalidStatus
	}
	now := common.GetTimestamp()
	var ticket Ticket
	var message *TicketMessage
	err := runTicketTransaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("id = ?", ticketId).Take(&ticket).Error; err != nil {
			return ticketLookupError(err)
		}
		if ticket.MessageCount >= TicketMaxMessagesPerTicket {
			return ErrTicketMessageLimit
		}
		var err error
		message, err = insertTicketMessage(tx, ticket.Id, adminId, TicketRoleAdmin, content, attachments, now)
		if err != nil {
			return err
		}
		nextStatus := status
		if nextStatus == 0 {
			nextStatus = ticket.Status
			if ticket.Status == TicketStatusPending {
				nextStatus = TicketStatusProcessing
			}
		}
		updates := map[string]any{
			"last_reply_role": TicketRoleAdmin,
			"user_unread":     true,
			"admin_unread":    false,
			"message_count":   gorm.Expr("message_count + ?", 1),
			"updated_time":    now,
		}
		applyTicketStatus(&ticket, nextStatus, now, updates)
		return tx.Model(&Ticket{}).Where("id = ?", ticket.Id).Updates(updates).Error
	})
	if err != nil {
		return nil, nil, err
	}
	ticket.LastReplyRole = TicketRoleAdmin
	ticket.UserUnread = true
	ticket.AdminUnread = false
	ticket.MessageCount++
	ticket.UpdatedTime = now
	return &ticket, message, nil
}

// CloseUserTicket 用户关闭自己的工单；已关闭时直接返回成功。
func CloseUserTicket(userId int, ticketId int) (*Ticket, error) {
	now := common.GetTimestamp()
	var ticket Ticket
	err := runTicketTransaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("id = ? AND user_id = ?", ticketId, userId).Take(&ticket).Error; err != nil {
			return ticketLookupError(err)
		}
		if ticket.Status == TicketStatusClosed {
			return nil
		}
		updates := map[string]any{"updated_time": now}
		applyTicketStatus(&ticket, TicketStatusClosed, now, updates)
		ticket.UpdatedTime = now
		return tx.Model(&Ticket{}).Where("id = ?", ticket.Id).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

// SetTicketStatus 管理员修改工单状态；状态确有变化时标记用户未读，相同状态直接返回成功。
func SetTicketStatus(ticketId int, status int) (*Ticket, error) {
	if !IsValidTicketStatus(status) {
		return nil, ErrTicketInvalidStatus
	}
	now := common.GetTimestamp()
	var ticket Ticket
	err := runTicketTransaction(func(tx *gorm.DB) error {
		if err := lockForUpdate(tx).Where("id = ?", ticketId).Take(&ticket).Error; err != nil {
			return ticketLookupError(err)
		}
		if ticket.Status == status {
			return nil
		}
		updates := map[string]any{"user_unread": true, "updated_time": now}
		applyTicketStatus(&ticket, status, now, updates)
		ticket.UserUnread = true
		ticket.UpdatedTime = now
		return tx.Model(&Ticket{}).Where("id = ?", ticket.Id).Updates(updates).Error
	})
	if err != nil {
		return nil, err
	}
	return &ticket, nil
}

// GetTicket 按 id 读取工单，不修改未读标记。
func GetTicket(ticketId int) (*Ticket, error) {
	var ticket Ticket
	if err := DB.Where("id = ?", ticketId).Take(&ticket).Error; err != nil {
		return nil, ticketLookupError(err)
	}
	return &ticket, nil
}

func GetUserTicket(userId int, ticketId int) (*Ticket, error) {
	var ticket Ticket
	if err := DB.Where("id = ? AND user_id = ?", ticketId, userId).Take(&ticket).Error; err != nil {
		return nil, ticketLookupError(err)
	}
	return &ticket, nil
}

// GetUserTickets 列出用户自己的工单；status 为 0 表示不过滤。
func GetUserTickets(userId int, status int, pageInfo *common.PageInfo) ([]*Ticket, int64, error) {
	query := DB.Model(&Ticket{}).Where("user_id = ?", userId)
	if status != 0 {
		query = query.Where("status = ?", status)
	}
	return findTicketPage(query, pageInfo)
}

// GetAdminTickets 管理端工单列表。keyword 按字面匹配标题或用户名（% 和 _ 不作通配符），
// 纯数字（可带 # 前缀）时还匹配工单 ID。
func GetAdminTickets(status int, keyword string, pageInfo *common.PageInfo) ([]*Ticket, int64, error) {
	query := DB.Model(&Ticket{})
	if status != 0 {
		query = query.Where("status = ?", status)
	}
	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		pattern := "%" + ticketLikeEscaper.Replace(keyword) + "%"
		usernameMatches := DB.Unscoped().Model(&User{}).Select("id").Where("username LIKE ? ESCAPE '!'", pattern)
		// 界面上的工单号显示为 #123，带 # 也按 ID 匹配
		if id, err := strconv.Atoi(strings.TrimPrefix(keyword, "#")); err == nil {
			query = query.Where("(id = ? OR subject LIKE ? ESCAPE '!' OR user_id IN (?))", id, pattern, usernameMatches)
		} else {
			query = query.Where("(subject LIKE ? ESCAPE '!' OR user_id IN (?))", pattern, usernameMatches)
		}
	}
	tickets, total, err := findTicketPage(query, pageInfo)
	if err != nil {
		return nil, 0, err
	}
	userIds := make([]int, 0, len(tickets))
	for _, ticket := range tickets {
		userIds = append(userIds, ticket.UserId)
	}
	usernames, err := ticketUsernames(userIds)
	if err != nil {
		return nil, 0, err
	}
	for _, ticket := range tickets {
		ticket.Username = usernames[ticket.UserId]
	}
	return tickets, total, nil
}

// findTicketPage 按最近更新排序分页；页大小限制在 1..100（GetPageQuery 不拒绝负数）。
func findTicketPage(query *gorm.DB, pageInfo *common.PageInfo) ([]*Ticket, int64, error) {
	pageInfo.PageSize = min(max(pageInfo.GetPageSize(), 1), 100)
	pageInfo.Page = max(pageInfo.GetPage(), 1)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	tickets := make([]*Ticket, 0)
	err := query.Order("updated_time DESC").Order("id DESC").
		Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).
		Find(&tickets).Error
	if err != nil {
		return nil, 0, err
	}
	return tickets, total, nil
}

// ticketUsernames 批量查询用户名；包含已注销的用户，管理端仍能看到历史工单的提交人。
func ticketUsernames(userIds []int) (map[int]string, error) {
	usernames := make(map[int]string, len(userIds))
	if len(userIds) == 0 {
		return usernames, nil
	}
	var users []User
	if err := DB.Unscoped().Select("id", "username").Where("id IN ?", userIds).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, user := range users {
		usernames[user.Id] = user.Username
	}
	return usernames, nil
}

func CountUserUnreadTickets(userId int) (int64, error) {
	var count int64
	err := DB.Model(&Ticket{}).Where("user_id = ? AND user_unread = ?", userId, true).Count(&count).Error
	return count, err
}

func GetTicketStats() (*TicketStats, error) {
	var rows []struct {
		Status int
		Total  int64
	}
	if err := DB.Model(&Ticket{}).Select("status, COUNT(*) AS total").Group("status").Scan(&rows).Error; err != nil {
		return nil, err
	}
	stats := &TicketStats{}
	for _, row := range rows {
		switch row.Status {
		case TicketStatusPending:
			stats.Pending = row.Total
		case TicketStatusProcessing:
			stats.Processing = row.Total
		case TicketStatusResolved:
			stats.Resolved = row.Total
		case TicketStatusClosed:
			stats.Closed = row.Total
		}
	}
	if err := DB.Model(&Ticket{}).Where("admin_unread = ?", true).Count(&stats.Unread).Error; err != nil {
		return nil, err
	}
	return stats, nil
}

// GetUserTicketDetail 返回用户自己的工单详情，并清除用户未读标记（不更新 updated_time）。
func GetUserTicketDetail(userId int, ticketId int) (*TicketDetail, error) {
	ticket, err := GetUserTicket(userId, ticketId)
	if err != nil {
		return nil, err
	}
	// 先清标记再读消息：清除前提交的客服回复一定会出现在本次返回的消息里
	if ticket.UserUnread {
		if err := DB.Model(&Ticket{}).Where("id = ?", ticket.Id).Update("user_unread", false).Error; err != nil {
			return nil, err
		}
		ticket.UserUnread = false
	}
	messages, err := loadTicketMessages(ticket.Id, false)
	if err != nil {
		return nil, err
	}
	return &TicketDetail{Ticket: ticket, Messages: messages}, nil
}

// GetAdminTicketDetail 返回工单详情（含提交人信息和消息作者用户名），并清除管理员未读标记。
func GetAdminTicketDetail(ticketId int) (*AdminTicketDetail, error) {
	var ticket Ticket
	if err := DB.Where("id = ?", ticketId).Take(&ticket).Error; err != nil {
		return nil, ticketLookupError(err)
	}
	if ticket.AdminUnread {
		if err := DB.Model(&Ticket{}).Where("id = ?", ticket.Id).Update("admin_unread", false).Error; err != nil {
			return nil, err
		}
		ticket.AdminUnread = false
	}
	messages, err := loadTicketMessages(ticket.Id, true)
	if err != nil {
		return nil, err
	}
	user := &TicketUser{Id: ticket.UserId}
	err = DB.Unscoped().Model(&User{}).
		Select("id", "username", "display_name", "email", "group").
		Where("id = ?", ticket.UserId).
		Take(user).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	ticket.Username = user.Username
	return &AdminTicketDetail{
		TicketDetail: TicketDetail{Ticket: &ticket, Messages: messages},
		User:         user,
	}, nil
}

// loadTicketMessages 按时间正序返回消息及附件元数据；withAuthorNames 仅用于管理端。
func loadTicketMessages(ticketId int, withAuthorNames bool) ([]*TicketMessage, error) {
	messages := make([]*TicketMessage, 0)
	if err := DB.Where("ticket_id = ?", ticketId).Order("id ASC").Find(&messages).Error; err != nil {
		return nil, err
	}
	var attachments []TicketAttachment
	err := DB.Select(ticketAttachmentMetaColumns).Where("ticket_id = ?", ticketId).Order("id ASC").Find(&attachments).Error
	if err != nil {
		return nil, err
	}
	byMessage := make(map[int][]TicketAttachment, len(messages))
	for _, attachment := range attachments {
		byMessage[attachment.MessageId] = append(byMessage[attachment.MessageId], attachment)
	}
	authorIds := make([]int, 0, len(messages))
	for _, message := range messages {
		message.Attachments = byMessage[message.Id]
		if message.Attachments == nil {
			message.Attachments = []TicketAttachment{}
		}
		authorIds = append(authorIds, message.UserId)
	}
	if !withAuthorNames {
		return messages, nil
	}
	usernames, err := ticketUsernames(authorIds)
	if err != nil {
		return nil, err
	}
	for _, message := range messages {
		message.AuthorName = usernames[message.UserId]
	}
	return messages, nil
}

// GetUserTicketAttachment 读取用户自己工单下的附件（含图片字节）。工单不属于该用户与附件不存在
// 返回同一个错误，不暴露其他用户的工单是否存在。
func GetUserTicketAttachment(userId int, ticketId int, attachmentId int) (*TicketAttachment, error) {
	var owned int64
	if err := DB.Model(&Ticket{}).Where("id = ? AND user_id = ?", ticketId, userId).Count(&owned).Error; err != nil {
		return nil, err
	}
	if owned == 0 {
		return nil, ErrTicketAttachmentNotFound
	}
	return GetTicketAttachment(ticketId, attachmentId)
}

// GetTicketAttachment 读取工单下的单个附件（含图片字节），附件必须属于该工单。
func GetTicketAttachment(ticketId int, attachmentId int) (*TicketAttachment, error) {
	var attachment TicketAttachment
	err := DB.Where("id = ? AND ticket_id = ?", attachmentId, ticketId).Take(&attachment).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTicketAttachmentNotFound
	}
	if err != nil {
		return nil, err
	}
	return &attachment, nil
}

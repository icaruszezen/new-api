package service

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// useTicketServiceDB 给工单测试独立的库，并重置本测试会用到的图片解码尝试计数。
func useTicketServiceDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := model.DB, model.LOG_DB
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.Ticket{}, &model.TicketMessage{}, &model.TicketAttachment{}))
	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	previousAttempts := ticketImageDecodeAttempts
	ticketImageDecodeAttempts = &common.InMemoryRateLimiter{}
	model.DB, model.LOG_DB = db, db
	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		ticketImageDecodeAttempts = previousAttempts
		_ = sqlDB.Close()
	})
	return db
}

func TestSanitizeTicketContent(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "normalizes CRLF and CR to LF", raw: "line1\r\nline2\rline3", want: "line1\nline2\nline3"},
		{name: "keeps tabs and newlines", raw: "a\tb\n\nc", want: "a\tb\n\nc"},
		{name: "strips C0 controls and DEL", raw: "a\x00b\x07c\x1b[31md\x7fe", want: "abc[31mde"},
		{name: "strips C1 controls", raw: "a\u0085b\u009bc", want: "abc"},
		{name: "strips bidi overrides and isolates", raw: "invoice\u202Egpj.exe\u202C \u2066x\u2069 \u200Ey\u200F", want: "invoicegpj.exe x y"},
		{name: "drops invalid UTF-8", raw: "ok\xff\xfeok", want: "okok"},
		{name: "trims surrounding whitespace", raw: " \n\t hello \r\n ", want: "hello"},
		{name: "keeps markup as plain text", raw: "<img src=x onerror=alert(1)>", want: "<img src=x onerror=alert(1)>"},
		{name: "keeps emoji and CJK", raw: "支付失败 😢", want: "支付失败 😢"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SanitizeTicketContent(tt.raw))
		})
	}
}

func TestSanitizeTicketSubject(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "collapses newline runs to one space", raw: "Top up\r\n\r\nfailed", want: "Top up failed"},
		{name: "collapses tabs", raw: "a\t\tb", want: "a b"},
		{name: "collapses spaces around line breaks", raw: "  Refund \r\n request ", want: "Refund request"},
		{name: "control-only subject becomes empty", raw: "\u202E\x00\n\t", want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, SanitizeTicketSubject(tt.raw))
		})
	}
}

func TestCreateTicketRejectsInvalidSubjectBeforeStoring(t *testing.T) {
	_, err := CreateTicket(context.Background(), 1, "\u202E\n", "body", nil)
	assert.ErrorIs(t, err, ErrTicketSubjectRequired)

	_, err = CreateTicket(context.Background(), 1, strings.Repeat("工", model.TicketSubjectMaxRunes+1), "body", nil)
	assert.ErrorIs(t, err, ErrTicketSubjectTooLong)
}

func TestCheckTicketContentLimits(t *testing.T) {
	tests := []struct {
		name            string
		raw             string
		imageCount      int
		contentRequired bool
		want            string
		wantErr         error
	}{
		{name: "limit counts runes not bytes", raw: strings.Repeat("工", model.TicketContentMaxRunes), want: strings.Repeat("工", model.TicketContentMaxRunes)},
		{name: "over the rune limit", raw: strings.Repeat("工", model.TicketContentMaxRunes+1), wantErr: ErrTicketContentTooLong},
		{name: "length is checked after sanitising", raw: strings.Repeat("a", model.TicketContentMaxRunes) + strings.Repeat("\u202E", 10), want: strings.Repeat("a", model.TicketContentMaxRunes)},
		{name: "new ticket needs text even with images", raw: " \u200F ", imageCount: 1, contentRequired: true, wantErr: ErrTicketContentRequired},
		{name: "reply may be image only", raw: "", imageCount: 1, want: ""},
		{name: "reply needs text or an image", raw: "\r\n", wantErr: ErrTicketContentRequired},
		{name: "at most four images per message", raw: "hi", imageCount: model.TicketMaxImagesPerMessage + 1, wantErr: ErrTicketTooManyImages},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := checkTicketContent(tt.raw, tt.imageCount, tt.contentRequired)
			if tt.wantErr != nil {
				assert.ErrorIs(t, err, tt.wantErr)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAcquireTicketUserWriteAllowsOneRequestPerUser(t *testing.T) {
	release, err := AcquireTicketUserWrite(41)
	require.NoError(t, err)

	_, err = AcquireTicketUserWrite(41)
	require.ErrorIs(t, err, ErrTicketRequestInProgress)
	otherRelease, err := AcquireTicketUserWrite(42)
	require.NoError(t, err, "other users are not blocked")
	otherRelease()

	release()
	release, err = AcquireTicketUserWrite(41)
	require.NoError(t, err)
	release()
}

// 失败的请求不保存任何图片，存储额度不会减少；解码尝试必须单独计数，否则损坏的最后一张图可以让前面的图片被无限次解码。
func TestFailedTicketRepliesUseUpDailyImageDecodeBudget(t *testing.T) {
	db := useTicketServiceDB(t)
	const userId = 7
	ticket, err := model.CreateTicket(userId, "Screenshots", "Body", nil)
	require.NoError(t, err)
	// 头部合法、没有像素数据：通过尺寸检查后在完整解码时失败
	corrupt := ticketTestPNGHeader(16, 16)
	images := [][]byte{corrupt, corrupt, corrupt, corrupt}

	for i := 0; i < ticketImageDecodeBudget/len(images); i++ {
		_, err = ReplyTicketAsUser(context.Background(), userId, ticket.Id, "", images)
		require.ErrorIs(t, err, ErrTicketImageInvalid, "request %d", i)
	}
	_, err = ReplyTicketAsUser(context.Background(), userId, ticket.Id, "", images)
	require.ErrorIs(t, err, model.ErrTicketDailyImageLimit)
	_, err = CreateTicket(context.Background(), userId, "Another", "Body", [][]byte{corrupt})
	require.ErrorIs(t, err, model.ErrTicketDailyImageLimit)

	var attachments int64
	require.NoError(t, db.Model(&model.TicketAttachment{}).Count(&attachments).Error)
	assert.Zero(t, attachments)
	stored, err := model.GetTicket(ticket.Id)
	require.NoError(t, err)
	assert.Equal(t, 1, stored.MessageCount)

	// 其他用户和不带图片的回复不受影响
	_, err = ReplyTicketAsUser(context.Background(), userId, ticket.Id, "Text only", nil)
	require.NoError(t, err)
	_, err = CreateTicket(context.Background(), userId+1, "Mine", "Body", [][]byte{encodeTicketTestPNG(t, newTicketTestImage(8, 8))})
	require.NoError(t, err)
}

// 单个工单的消息数和附件数上限要在解码图片前检查：若先解码，这些请求会得到 ErrTicketImageInvalid。
func TestTicketRepliesCheckPerTicketLimitsBeforeDecoding(t *testing.T) {
	db := useTicketServiceDB(t)
	const ownerId, adminId = 7, 1
	full, err := model.CreateTicket(ownerId, "Long thread", "Body", nil)
	require.NoError(t, err)
	require.NoError(t, db.Model(&model.Ticket{}).Where("id = ?", full.Id).Update("message_count", model.TicketMaxMessagesPerTicket).Error)
	crowded, err := model.CreateTicket(ownerId, "Many images", "Body", nil)
	require.NoError(t, err)
	for i := 0; i < model.TicketMaxAttachmentsPerTicket; i++ {
		require.NoError(t, db.Create(&model.TicketAttachment{TicketId: crowded.Id, UserId: adminId, CreatedTime: 1, Data: []byte{1}}).Error)
	}
	corrupt := [][]byte{ticketTestPNGHeader(16, 16)}

	tests := []struct {
		name     string
		ticketId int
		asAdmin  bool
		wantErr  error
	}{
		{name: "user reply at message limit", ticketId: full.Id, wantErr: model.ErrTicketMessageLimit},
		{name: "user reply over attachment limit", ticketId: crowded.Id, wantErr: model.ErrTicketAttachmentLimit},
		{name: "admin reply at message limit", ticketId: full.Id, asAdmin: true, wantErr: model.ErrTicketMessageLimit},
		{name: "admin reply over attachment limit", ticketId: crowded.Id, asAdmin: true, wantErr: model.ErrTicketAttachmentLimit},
		{name: "admin reply to missing ticket", ticketId: crowded.Id + 100, asAdmin: true, wantErr: model.ErrTicketNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			if tt.asAdmin {
				_, _, err = ReplyTicketAsAdmin(context.Background(), adminId, tt.ticketId, "", corrupt, 0)
			} else {
				_, err = ReplyTicketAsUser(context.Background(), ownerId, tt.ticketId, "", corrupt)
			}
			require.ErrorIs(t, err, tt.wantErr)
		})
	}
}

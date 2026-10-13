package model

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// useTicketDB 给每个测试独立的库，避免和 TestMain 的共享库互相污染。
func useTicketDB(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB, previousLogDB := DB, LOG_DB
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&User{}, &Ticket{}, &TicketMessage{}, &TicketAttachment{}))

	sqlDB, err := db.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)

	DB, LOG_DB = db, db
	t.Cleanup(func() {
		DB, LOG_DB = previousDB, previousLogDB
		_ = sqlDB.Close()
	})
	return db
}

func createTicketTestUser(t *testing.T, db *gorm.DB, username string, role int) *User {
	t.Helper()
	user := &User{
		Username: username, Password: "password", DisplayName: username + " display", Email: username + "@example.com",
		Role: role, Status: common.UserStatusEnabled, Group: "vip", AffCode: "ticket-" + username,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

func reloadTicket(t *testing.T, db *gorm.DB, id int) Ticket {
	t.Helper()
	var ticket Ticket
	require.NoError(t, db.First(&ticket, id).Error)
	return ticket
}

func testTicketAttachments(count int) []*TicketAttachment {
	attachments := make([]*TicketAttachment, 0, count)
	for i := 0; i < count; i++ {
		attachments = append(attachments, &TicketAttachment{MimeType: "image/png", Size: 3, Width: 1, Height: 1, Data: []byte{1, 2, 3}})
	}
	return attachments
}

func TestTicketConversationStateMachine(t *testing.T) {
	db := useTicketDB(t)
	owner := createTicketTestUser(t, db, "ticket-owner", common.RoleCommonUser)
	support := createTicketTestUser(t, db, "ticket-support", common.RoleAdminUser)

	created, err := CreateTicket(owner.Id, "Cannot top up", "Payment fails", testTicketAttachments(1))
	require.NoError(t, err)
	stored := reloadTicket(t, db, created.Id)
	assert.Equal(t, TicketStatusPending, stored.Status)
	assert.Equal(t, TicketRoleUser, stored.LastReplyRole)
	assert.Equal(t, 1, stored.MessageCount)
	assert.True(t, stored.AdminUnread)
	assert.False(t, stored.UserUnread)
	assert.Equal(t, stored.CreatedTime, stored.UpdatedTime)
	assert.Zero(t, stored.ClosedTime)

	// 客服首次回复：待处理 -> 处理中，用户未读
	_, message, err := AddAdminTicketReply(support.Id, created.Id, "Looking into it", nil, 0)
	require.NoError(t, err)
	assert.Equal(t, TicketRoleAdmin, message.AuthorRole)
	stored = reloadTicket(t, db, created.Id)
	assert.Equal(t, TicketStatusProcessing, stored.Status)
	assert.Equal(t, TicketRoleAdmin, stored.LastReplyRole)
	assert.Equal(t, 2, stored.MessageCount)
	assert.True(t, stored.UserUnread)
	assert.False(t, stored.AdminUnread)

	// 用户追加回复：回到待处理，管理员未读
	_, err = AddUserTicketReply(owner.Id, created.Id, "Still failing", nil)
	require.NoError(t, err)
	stored = reloadTicket(t, db, created.Id)
	assert.Equal(t, TicketStatusPending, stored.Status)
	assert.Equal(t, TicketRoleUser, stored.LastReplyRole)
	assert.Equal(t, 3, stored.MessageCount)
	assert.True(t, stored.AdminUnread)

	// 已解决的工单被用户回复后重新打开
	_, err = SetTicketStatus(created.Id, TicketStatusResolved)
	require.NoError(t, err)
	_, err = AddUserTicketReply(owner.Id, created.Id, "It broke again", nil)
	require.NoError(t, err)
	stored = reloadTicket(t, db, created.Id)
	assert.Equal(t, TicketStatusPending, stored.Status)
	assert.Equal(t, 4, stored.MessageCount)

	// 客服回复时指定状态优先于自动流转
	_, _, err = AddAdminTicketReply(support.Id, created.Id, "Fixed", nil, TicketStatusResolved)
	require.NoError(t, err)
	assert.Equal(t, TicketStatusResolved, reloadTicket(t, db, created.Id).Status)

	// 处理中/已解决状态下的客服回复不改变状态
	_, _, err = AddAdminTicketReply(support.Id, created.Id, "Anything else?", nil, 0)
	require.NoError(t, err)
	assert.Equal(t, TicketStatusResolved, reloadTicket(t, db, created.Id).Status)
}

func TestTicketCloseBlocksUserRepliesAndAdminStatusReopens(t *testing.T) {
	db := useTicketDB(t)
	owner := createTicketTestUser(t, db, "ticket-closer", common.RoleCommonUser)
	support := createTicketTestUser(t, db, "ticket-agent", common.RoleAdminUser)
	created, err := CreateTicket(owner.Id, "Question", "Body", nil)
	require.NoError(t, err)

	closed, err := CloseUserTicket(owner.Id, created.Id)
	require.NoError(t, err)
	assert.Equal(t, TicketStatusClosed, closed.Status)
	stored := reloadTicket(t, db, created.Id)
	assert.Equal(t, TicketStatusClosed, stored.Status)
	assert.NotZero(t, stored.ClosedTime)

	_, err = AddUserTicketReply(owner.Id, created.Id, "One more thing", nil)
	assert.ErrorIs(t, err, ErrTicketClosed)
	assert.Equal(t, 1, reloadTicket(t, db, created.Id).MessageCount)

	// 重复关闭是幂等的成功操作，不改动关闭时间
	require.NoError(t, db.Model(&Ticket{}).Where("id = ?", created.Id).Update("closed_time", 123).Error)
	_, err = CloseUserTicket(owner.Id, created.Id)
	require.NoError(t, err)
	assert.EqualValues(t, 123, reloadTicket(t, db, created.Id).ClosedTime)

	// 管理员设置相同状态是空操作，不会标记用户未读
	_, err = SetTicketStatus(created.Id, TicketStatusClosed)
	require.NoError(t, err)
	assert.False(t, reloadTicket(t, db, created.Id).UserUnread)

	// 管理员仍可回复已关闭的工单，未指定状态时保持关闭
	_, _, err = AddAdminTicketReply(support.Id, created.Id, "Noted", nil, 0)
	require.NoError(t, err)
	stored = reloadTicket(t, db, created.Id)
	assert.Equal(t, TicketStatusClosed, stored.Status)
	assert.EqualValues(t, 123, stored.ClosedTime)

	// 改为其他状态即重新打开：清空关闭时间并标记用户未读
	require.NoError(t, db.Model(&Ticket{}).Where("id = ?", created.Id).Update("user_unread", false).Error)
	reopened, err := SetTicketStatus(created.Id, TicketStatusProcessing)
	require.NoError(t, err)
	assert.Equal(t, TicketStatusProcessing, reopened.Status)
	stored = reloadTicket(t, db, created.Id)
	assert.Equal(t, TicketStatusProcessing, stored.Status)
	assert.Zero(t, stored.ClosedTime)
	assert.True(t, stored.UserUnread)

	_, err = AddUserTicketReply(owner.Id, created.Id, "Thanks for reopening", nil)
	require.NoError(t, err)

	// 客服回复时直接关闭：记录关闭时间
	_, _, err = AddAdminTicketReply(support.Id, created.Id, "Closing", nil, TicketStatusClosed)
	require.NoError(t, err)
	stored = reloadTicket(t, db, created.Id)
	assert.Equal(t, TicketStatusClosed, stored.Status)
	assert.NotZero(t, stored.ClosedTime)

	_, err = SetTicketStatus(created.Id, 5)
	assert.ErrorIs(t, err, ErrTicketInvalidStatus)
	_, _, err = AddAdminTicketReply(support.Id, created.Id, "Bad", nil, 9)
	assert.ErrorIs(t, err, ErrTicketInvalidStatus)
}

func TestTicketDetailClearsOnlyTheViewersUnreadFlag(t *testing.T) {
	db := useTicketDB(t)
	owner := createTicketTestUser(t, db, "ticket-reader", common.RoleCommonUser)
	support := createTicketTestUser(t, db, "ticket-helper", common.RoleAdminUser)
	created, err := CreateTicket(owner.Id, "Subject", "First", testTicketAttachments(2))
	require.NoError(t, err)
	_, _, err = AddAdminTicketReply(support.Id, created.Id, "Reply", nil, 0)
	require.NoError(t, err)
	require.NoError(t, db.Model(&Ticket{}).Where("id = ?", created.Id).Updates(map[string]any{
		"admin_unread": true, "updated_time": 1000,
	}).Error)

	detail, err := GetUserTicketDetail(owner.Id, created.Id)
	require.NoError(t, err)
	assert.False(t, detail.Ticket.UserUnread)
	stored := reloadTicket(t, db, created.Id)
	assert.False(t, stored.UserUnread)
	assert.True(t, stored.AdminUnread, "opening as the user must not clear the admin flag")
	assert.EqualValues(t, 1000, stored.UpdatedTime, "opening a ticket is not an update")

	require.Len(t, detail.Messages, 2)
	assert.Equal(t, TicketRoleUser, detail.Messages[0].AuthorRole)
	assert.Equal(t, TicketRoleAdmin, detail.Messages[1].AuthorRole)
	assert.Empty(t, detail.Messages[1].AuthorName, "user views never carry the support agent's name")
	require.Len(t, detail.Messages[0].Attachments, 2)
	assert.Nil(t, detail.Messages[0].Attachments[0].Data, "listing attachments must not load image bytes")
	assert.Equal(t, "image/png", detail.Messages[0].Attachments[0].MimeType)
	assert.NotNil(t, detail.Messages[1].Attachments)
	assert.Empty(t, detail.Messages[1].Attachments)

	_, err = GetUserTicketDetail(support.Id, created.Id)
	assert.ErrorIs(t, err, ErrTicketNotFound, "tickets are only visible to their owner on the user side")

	require.NoError(t, db.Model(&Ticket{}).Where("id = ?", created.Id).Update("user_unread", true).Error)
	adminDetail, err := GetAdminTicketDetail(created.Id)
	require.NoError(t, err)
	stored = reloadTicket(t, db, created.Id)
	assert.False(t, stored.AdminUnread)
	assert.True(t, stored.UserUnread, "opening as admin must not clear the user flag")
	assert.Equal(t, owner.Username, adminDetail.Ticket.Username)
	assert.Equal(t, owner.Username, adminDetail.Messages[0].AuthorName)
	assert.Equal(t, support.Username, adminDetail.Messages[1].AuthorName)
	assert.Equal(t, TicketUser{
		Id: owner.Id, Username: owner.Username, DisplayName: owner.DisplayName, Email: owner.Email, Group: "vip",
	}, *adminDetail.User)

	unread, err := CountUserUnreadTickets(owner.Id)
	require.NoError(t, err)
	assert.EqualValues(t, 1, unread)
}

func TestTicketActiveTicketLimit(t *testing.T) {
	db := useTicketDB(t)
	owner := createTicketTestUser(t, db, "ticket-busy", common.RoleCommonUser)

	ids := make([]int, 0, TicketMaxActivePerUser)
	for i := 0; i < TicketMaxActivePerUser; i++ {
		created, err := CreateTicket(owner.Id, fmt.Sprintf("Ticket %d", i), "Body", nil)
		require.NoError(t, err)
		ids = append(ids, created.Id)
	}
	// 待处理和处理中都算活跃工单
	_, err := SetTicketStatus(ids[0], TicketStatusProcessing)
	require.NoError(t, err)

	_, err = CreateTicket(owner.Id, "One too many", "Body", nil)
	assert.ErrorIs(t, err, ErrTicketTooManyOpen)
	assert.ErrorIs(t, CheckTicketUserLimits(owner.Id, true, 0), ErrTicketTooManyOpen)

	// 已解决或已关闭的工单不占名额
	_, err = SetTicketStatus(ids[1], TicketStatusResolved)
	require.NoError(t, err)
	_, err = CreateTicket(owner.Id, "Now allowed", "Body", nil)
	require.NoError(t, err)

	// 其他用户不受影响
	other := createTicketTestUser(t, db, "ticket-other", common.RoleCommonUser)
	_, err = CreateTicket(other.Id, "Mine", "Body", nil)
	require.NoError(t, err)
}

func TestTicketDailyImageLimit(t *testing.T) {
	db := useTicketDB(t)
	owner := createTicketTestUser(t, db, "ticket-uploader", common.RoleCommonUser)
	target, err := CreateTicket(owner.Id, "Screenshots", "Body", nil)
	require.NoError(t, err)

	// 另一张工单上已有 39 张 24 小时内的图片，以及 1 张超出窗口的旧图片
	archive := &Ticket{UserId: owner.Id, Subject: "Archive", Status: TicketStatusClosed, MessageCount: 1}
	require.NoError(t, db.Create(archive).Error)
	now := common.GetTimestamp()
	for i := 0; i < TicketMaxImagesPerUserPerDay-1; i++ {
		require.NoError(t, db.Create(&TicketAttachment{TicketId: archive.Id, UserId: owner.Id, CreatedTime: now - 60, Data: []byte{1}}).Error)
	}
	require.NoError(t, db.Create(&TicketAttachment{TicketId: archive.Id, UserId: owner.Id, CreatedTime: now - ticketImageWindowSeconds - 60, Data: []byte{1}}).Error)

	_, err = AddUserTicketReply(owner.Id, target.Id, "Two more", testTicketAttachments(2))
	assert.ErrorIs(t, err, ErrTicketDailyImageLimit)
	assert.ErrorIs(t, CheckTicketUserLimits(owner.Id, false, 2), ErrTicketDailyImageLimit)
	assert.Equal(t, 1, reloadTicket(t, db, target.Id).MessageCount, "a rejected reply stores nothing")

	_, err = AddUserTicketReply(owner.Id, target.Id, "Exactly at the cap", testTicketAttachments(1))
	require.NoError(t, err)
	_, err = CreateTicket(owner.Id, "New ticket", "Body", testTicketAttachments(1))
	assert.ErrorIs(t, err, ErrTicketDailyImageLimit)

	// 不带图片的回复不受图片额度影响
	_, err = AddUserTicketReply(owner.Id, target.Id, "Text only", nil)
	require.NoError(t, err)
}

func TestTicketPerTicketLimits(t *testing.T) {
	db := useTicketDB(t)
	owner := createTicketTestUser(t, db, "ticket-chatty", common.RoleCommonUser)
	support := createTicketTestUser(t, db, "ticket-support2", common.RoleAdminUser)
	created, err := CreateTicket(owner.Id, "Long thread", "Body", nil)
	require.NoError(t, err)

	require.NoError(t, db.Model(&Ticket{}).Where("id = ?", created.Id).Update("message_count", TicketMaxMessagesPerTicket).Error)
	_, err = AddUserTicketReply(owner.Id, created.Id, "More", nil)
	assert.ErrorIs(t, err, ErrTicketMessageLimit)
	_, _, err = AddAdminTicketReply(support.Id, created.Id, "More", nil, 0)
	assert.ErrorIs(t, err, ErrTicketMessageLimit)

	require.NoError(t, db.Model(&Ticket{}).Where("id = ?", created.Id).Update("message_count", 1).Error)
	for i := 0; i < TicketMaxAttachmentsPerTicket-1; i++ {
		// 管理员上传的旧图片也计入单个工单的附件上限，同时不占用户的每日额度
		require.NoError(t, db.Create(&TicketAttachment{TicketId: created.Id, UserId: support.Id, CreatedTime: 1, Data: []byte{1}}).Error)
	}
	_, err = AddUserTicketReply(owner.Id, created.Id, "Two images", testTicketAttachments(2))
	assert.ErrorIs(t, err, ErrTicketAttachmentLimit)
	_, err = AddUserTicketReply(owner.Id, created.Id, "Last image", testTicketAttachments(1))
	require.NoError(t, err)
	_, _, err = AddAdminTicketReply(support.Id, created.Id, "One more", testTicketAttachments(1), 0)
	assert.ErrorIs(t, err, ErrTicketAttachmentLimit)
}

func TestTicketListingAndStats(t *testing.T) {
	db := useTicketDB(t)
	owner := createTicketTestUser(t, db, "ticket-lister", common.RoleCommonUser)
	other := createTicketTestUser(t, db, "ticket-stranger", common.RoleCommonUser)
	older, err := CreateTicket(owner.Id, "Older", "Body", nil)
	require.NoError(t, err)
	newer, err := CreateTicket(owner.Id, "Newer", "Body", nil)
	require.NoError(t, err)
	tied, err := CreateTicket(owner.Id, "Tied", "Body", nil)
	require.NoError(t, err)
	_, err = CreateTicket(other.Id, "Not yours", "Body", nil)
	require.NoError(t, err)
	require.NoError(t, db.Model(&Ticket{}).Where("id = ?", older.Id).Update("updated_time", 100).Error)
	require.NoError(t, db.Model(&Ticket{}).Where("id IN ?", []int{newer.Id, tied.Id}).Update("updated_time", 200).Error)
	_, err = SetTicketStatus(older.Id, TicketStatusResolved)
	require.NoError(t, err)
	require.NoError(t, db.Model(&Ticket{}).Where("id = ?", older.Id).Update("updated_time", 100).Error)

	tickets, total, err := GetUserTickets(owner.Id, 0, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	ids := make([]int, 0, len(tickets))
	for _, ticket := range tickets {
		ids = append(ids, ticket.Id)
	}
	assert.Equal(t, []int{tied.Id, newer.Id, older.Id}, ids, "updated_time desc, then id desc")

	tickets, total, err = GetUserTickets(owner.Id, TicketStatusResolved, &common.PageInfo{Page: 1, PageSize: 10})
	require.NoError(t, err)
	assert.EqualValues(t, 1, total)
	require.Len(t, tickets, 1)
	assert.Equal(t, older.Id, tickets[0].Id)

	// GetPageQuery 不拒绝负数页大小，模型层必须夹紧，不能退化为不分页
	pageInfo := &common.PageInfo{Page: -3, PageSize: -1}
	tickets, total, err = GetUserTickets(owner.Id, 0, pageInfo)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	assert.Len(t, tickets, 1)
	assert.Equal(t, 1, pageInfo.Page)
	assert.Equal(t, 1, pageInfo.PageSize)

	stats, err := GetTicketStats()
	require.NoError(t, err)
	assert.Equal(t, TicketStats{Pending: 3, Resolved: 1, Unread: 4}, *stats)
}

// SQLite 上两个重叠的工单写事务在升级写锁时会有一方立即得到 SQLITE_BUSY，整个事务需要重试；
// 其他错误和其他数据库不重试，失败的尝试不能留下任何数据。
func TestRunTicketTransactionRetriesSQLiteBusy(t *testing.T) {
	busy := errors.New("database is locked (5) (SQLITE_BUSY)")
	other := errors.New("UNIQUE constraint failed")
	tests := []struct {
		name      string
		database  common.DatabaseType
		failures  int
		failWith  error
		wantCalls int
		wantErr   error
	}{
		{name: "busy once then committed", database: common.DatabaseTypeSQLite, failures: 1, failWith: busy, wantCalls: 2},
		{name: "gives up after the last attempt", database: common.DatabaseTypeSQLite, failures: 10, failWith: busy, wantCalls: ticketTransactionAttempts, wantErr: busy},
		{name: "other errors are not retried", database: common.DatabaseTypeSQLite, failures: 10, failWith: other, wantCalls: 1, wantErr: other},
		{name: "not retried on mysql", database: common.DatabaseTypeMySQL, failures: 10, failWith: busy, wantCalls: 1, wantErr: busy},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := useTicketDB(t)
			previousMain, previousLog := common.MainDatabaseType(), common.LogDatabaseType()
			common.SetDatabaseTypes(tt.database, previousLog)
			t.Cleanup(func() { common.SetDatabaseTypes(previousMain, previousLog) })

			calls := 0
			err := runTicketTransaction(func(tx *gorm.DB) error {
				calls++
				if err := tx.Create(&Ticket{UserId: 1, Subject: "retry"}).Error; err != nil {
					return err
				}
				if calls <= tt.failures {
					return tt.failWith
				}
				return nil
			})
			assert.Equal(t, tt.wantCalls, calls)
			var stored int64
			require.NoError(t, db.Model(&Ticket{}).Count(&stored).Error)
			if tt.wantErr != nil {
				require.ErrorIs(t, err, tt.wantErr)
				assert.Zero(t, stored)
				return
			}
			require.NoError(t, err)
			assert.EqualValues(t, 1, stored)
		})
	}
}

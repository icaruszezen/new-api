package controller

import (
	"bytes"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupTicketControllerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	gin.SetMode(gin.TestMode)
	previousDB, previousLogDB := model.DB, model.LOG_DB
	previousRedisEnabled := common.RedisEnabled
	previousMainDatabaseType, previousLogDatabaseType := common.MainDatabaseType(), common.LogDatabaseType()
	common.RedisEnabled = false
	common.SetDatabaseTypes(common.DatabaseTypeSQLite, common.DatabaseTypeSQLite)

	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", strings.ReplaceAll(t.Name(), "/", "_"))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	model.DB, model.LOG_DB = db, db
	require.NoError(t, db.AutoMigrate(
		&model.User{}, &model.Log{}, &model.Ticket{}, &model.TicketMessage{}, &model.TicketAttachment{},
	))

	t.Cleanup(func() {
		model.DB, model.LOG_DB = previousDB, previousLogDB
		common.RedisEnabled = previousRedisEnabled
		common.SetDatabaseTypes(previousMainDatabaseType, previousLogDatabaseType)
		sqlDB, err := db.DB()
		if err == nil {
			_ = sqlDB.Close()
		}
	})
	return db
}

func createTicketControllerUser(t *testing.T, db *gorm.DB, username string, role int) *model.User {
	t.Helper()
	user := &model.User{
		Username: username, Password: "password", Role: role, Status: common.UserStatusEnabled,
		Group: "default", AffCode: "aff-" + username,
	}
	require.NoError(t, db.Create(user).Error)
	return user
}

type ticketTestRequest struct {
	method      string
	target      string
	body        io.Reader
	contentType string
	params      gin.Params
}

func serveTicketRequest(t *testing.T, handler gin.HandlerFunc, actor *model.User, request ticketTestRequest) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest(request.method, request.target, request.body)
	if request.contentType != "" {
		c.Request.Header.Set("Content-Type", request.contentType)
	}
	c.Params = request.params
	c.Set("id", actor.Id)
	c.Set("role", actor.Role)
	c.Set("username", actor.Username)
	handler(c)
	return recorder
}

type ticketAPIResponse struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func decodeTicketResponse(t *testing.T, recorder *httptest.ResponseRecorder) ticketAPIResponse {
	t.Helper()
	require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
	var response ticketAPIResponse
	require.NoError(t, common.Unmarshal(recorder.Body.Bytes(), &response))
	return response
}

type ticketUpload struct {
	filename    string
	contentType string
	data        []byte
}

func ticketMultipartBody(t *testing.T, fields map[string]string, uploads ...ticketUpload) (*bytes.Buffer, string) {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	for key, value := range fields {
		require.NoError(t, writer.WriteField(key, value))
	}
	for _, upload := range uploads {
		header := textproto.MIMEHeader{}
		header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="files"; filename=%q`, upload.filename))
		header.Set("Content-Type", upload.contentType)
		part, err := writer.CreatePart(header)
		require.NoError(t, err)
		_, err = part.Write(upload.data)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return &body, writer.FormDataContentType()
}

func ticketTestPNG(t *testing.T) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 8, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 8; x++ {
			img.Set(x, y, color.NRGBA{R: uint8(x * 30), G: uint8(y * 40), B: 200, A: 255})
		}
	}
	var buf bytes.Buffer
	require.NoError(t, png.Encode(&buf, img))
	return buf.Bytes()
}

func pngUpload(data []byte) ticketUpload {
	return ticketUpload{filename: "screenshot.png", contentType: "image/png", data: data}
}

func createTicketViaAPI(t *testing.T, user *model.User, subject string, uploads ...ticketUpload) model.Ticket {
	t.Helper()
	body, contentType := ticketMultipartBody(t, map[string]string{"subject": subject, "content": "details"}, uploads...)
	recorder := serveTicketRequest(t, CreateSelfTicket, user, ticketTestRequest{
		method: http.MethodPost, target: "/api/ticket/self", body: body, contentType: contentType,
	})
	response := decodeTicketResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	var ticket model.Ticket
	require.NoError(t, common.Unmarshal(response.Data, &ticket))
	return ticket
}

func ticketParams(ticketId int, attachmentId ...int) gin.Params {
	params := gin.Params{{Key: "id", Value: strconv.Itoa(ticketId)}}
	if len(attachmentId) > 0 {
		params = append(params, gin.Param{Key: "attachment_id", Value: strconv.Itoa(attachmentId[0])})
	}
	return params
}

func assertNoTicketRows(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, table := range []any{&model.Ticket{}, &model.TicketMessage{}, &model.TicketAttachment{}} {
		var count int64
		require.NoError(t, db.Model(table).Count(&count).Error)
		assert.Zero(t, count, "%T rows must not be stored for a rejected request", table)
	}
}

func TestCreateTicketStoresOnlyReencodedImage(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	user := createTicketControllerUser(t, db, "ticket-author", common.RoleCommonUser)
	payload := []byte("<html><script>alert(document.cookie)</script></html>")
	upload := append(ticketTestPNG(t), payload...)

	body, contentType := ticketMultipartBody(t,
		map[string]string{"subject": "  Refund \r\n request ", "content": "Charged twice\u202E\r\n"},
		ticketUpload{filename: "../../evil.html", contentType: "text/html", data: upload},
	)
	recorder := serveTicketRequest(t, CreateSelfTicket, user, ticketTestRequest{
		method: http.MethodPost, target: "/api/ticket/self", body: body, contentType: contentType,
	})
	response := decodeTicketResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	var ticket model.Ticket
	require.NoError(t, common.Unmarshal(response.Data, &ticket))
	assert.Equal(t, "Refund request", ticket.Subject)
	assert.Equal(t, model.TicketStatusPending, ticket.Status)
	assert.True(t, ticket.AdminUnread)

	var message model.TicketMessage
	require.NoError(t, db.Where("ticket_id = ?", ticket.Id).Take(&message).Error)
	assert.Equal(t, "Charged twice", message.Content)

	var attachments []model.TicketAttachment
	require.NoError(t, db.Where("ticket_id = ?", ticket.Id).Find(&attachments).Error)
	require.Len(t, attachments, 1)
	stored := attachments[0]
	assert.Equal(t, message.Id, stored.MessageId)
	assert.Equal(t, user.Id, stored.UserId)
	assert.Equal(t, "image/png", stored.MimeType, "the stored type comes from the encoder, not the client")
	assert.NotEqual(t, upload, stored.Data)
	assert.False(t, bytes.Contains(stored.Data, payload), "bytes appended after IEND must not be stored")
	assert.Equal(t, len(stored.Data), stored.Size)
	decoded, err := png.Decode(bytes.NewReader(stored.Data))
	require.NoError(t, err)
	assert.Equal(t, image.Rect(0, 0, 8, 6), decoded.Bounds())
	assert.Equal(t, 8, stored.Width)
	assert.Equal(t, 6, stored.Height)
}

func TestCreateTicketRejectsNonImageUploads(t *testing.T) {
	tests := []struct {
		name   string
		upload ticketUpload
	}{
		{name: "svg", upload: ticketUpload{filename: "logo.svg", contentType: "image/svg+xml",
			data: []byte(`<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"><rect width="1" height="1"/></svg>`)}},
		{name: "html disguised as png", upload: ticketUpload{filename: "photo.png", contentType: "image/png",
			data: []byte("<!DOCTYPE html><html><script>alert(1)</script></html>")}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := setupTicketControllerTestDB(t)
			user := createTicketControllerUser(t, db, "ticket-svg", common.RoleCommonUser)
			body, contentType := ticketMultipartBody(t, map[string]string{"subject": "Logo", "content": "See file"}, tt.upload)
			recorder := serveTicketRequest(t, CreateSelfTicket, user, ticketTestRequest{
				method: http.MethodPost, target: "/api/ticket/self", body: body, contentType: contentType,
			})
			response := decodeTicketResponse(t, recorder)
			assert.False(t, response.Success)
			assert.Equal(t, i18n.MsgTicketImageInvalidType, response.Message)
			assertNoTicketRows(t, db)
		})
	}
}

func TestCreateTicketRejectsMoreThanFourFiles(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	user := createTicketControllerUser(t, db, "ticket-many-files", common.RoleCommonUser)
	uploads := make([]ticketUpload, 0, model.TicketMaxImagesPerMessage+1)
	for i := 0; i <= model.TicketMaxImagesPerMessage; i++ {
		uploads = append(uploads, pngUpload(ticketTestPNG(t)))
	}
	body, contentType := ticketMultipartBody(t, map[string]string{"subject": "Images", "content": "Five of them"}, uploads...)
	recorder := serveTicketRequest(t, CreateSelfTicket, user, ticketTestRequest{
		method: http.MethodPost, target: "/api/ticket/self", body: body, contentType: contentType,
	})
	response := decodeTicketResponse(t, recorder)
	assert.False(t, response.Success)
	assert.Equal(t, i18n.MsgTicketTooManyImages, response.Message)
	assertNoTicketRows(t, db)
}

func TestCreateTicketRejectsOversizedOrNonMultipartBody(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	user := createTicketControllerUser(t, db, "ticket-big-body", common.RoleCommonUser)

	oversized, contentType := ticketMultipartBody(t, map[string]string{"subject": "Big", "content": "Body"},
		pngUpload(make([]byte, ticketMaxRequestBytes)))
	recorder := serveTicketRequest(t, CreateSelfTicket, user, ticketTestRequest{
		method: http.MethodPost, target: "/api/ticket/self", body: oversized, contentType: contentType,
	})
	response := decodeTicketResponse(t, recorder)
	assert.False(t, response.Success)
	assert.Equal(t, i18n.MsgTicketRequestTooLarge, response.Message)

	recorder = serveTicketRequest(t, CreateSelfTicket, user, ticketTestRequest{
		method: http.MethodPost, target: "/api/ticket/self",
		body: strings.NewReader(`{"subject":"Hi","content":"There"}`), contentType: "application/json",
	})
	response = decodeTicketResponse(t, recorder)
	assert.False(t, response.Success)
	assert.Equal(t, i18n.MsgTicketInvalidForm, response.Message)
	assertNoTicketRows(t, db)
}

// 同一用户已有工单写请求在处理时，新的创建/回复在读取请求体前就被拒绝；客服回复不受限制。
func TestTicketUserWritesAreRejectedWhileOneIsInProgress(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	owner := createTicketControllerUser(t, db, "ticket-impatient", common.RoleCommonUser)
	admin := createTicketControllerUser(t, db, "ticket-busy-admin", common.RoleAdminUser)
	ticket := createTicketViaAPI(t, owner, "First")

	release, err := service.AcquireTicketUserWrite(owner.Id)
	require.NoError(t, err)
	createBody, createType := ticketMultipartBody(t, map[string]string{"subject": "Second", "content": "details"}, pngUpload(ticketTestPNG(t)))
	replyBody, replyType := ticketMultipartBody(t, map[string]string{"content": "again"})
	for name, recorder := range map[string]*httptest.ResponseRecorder{
		"create": serveTicketRequest(t, CreateSelfTicket, owner, ticketTestRequest{
			method: http.MethodPost, target: "/api/ticket/self", body: createBody, contentType: createType,
		}),
		"reply": serveTicketRequest(t, ReplySelfTicket, owner, ticketTestRequest{
			method: http.MethodPost, target: "/api/ticket/self/reply", body: replyBody, contentType: replyType, params: ticketParams(ticket.Id),
		}),
	} {
		response := decodeTicketResponse(t, recorder)
		assert.False(t, response.Success, name)
		assert.Equal(t, i18n.MsgTicketRequestInProgress, response.Message, name)
	}

	adminRelease, err := service.AcquireTicketUserWrite(admin.Id)
	require.NoError(t, err)
	adminBody, adminType := ticketMultipartBody(t, map[string]string{"content": "On it"})
	recorder := serveTicketRequest(t, AdminReplyTicket, admin, ticketTestRequest{
		method: http.MethodPost, target: "/api/ticket/admin/reply", body: adminBody, contentType: adminType, params: ticketParams(ticket.Id),
	})
	require.True(t, decodeTicketResponse(t, recorder).Success)
	adminRelease()

	release()
	createTicketViaAPI(t, owner, "Second")
	var tickets int64
	require.NoError(t, db.Model(&model.Ticket{}).Count(&tickets).Error)
	assert.EqualValues(t, 2, tickets)
	var stored model.Ticket
	require.NoError(t, db.First(&stored, ticket.Id).Error)
	assert.Equal(t, 2, stored.MessageCount, "only the admin reply is stored")
}

func TestUserCannotAccessAnotherUsersTicket(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	owner := createTicketControllerUser(t, db, "ticket-owner-b", common.RoleCommonUser)
	intruder := createTicketControllerUser(t, db, "ticket-intruder-a", common.RoleCommonUser)
	ticket := createTicketViaAPI(t, owner, "Private", pngUpload(ticketTestPNG(t)))
	var attachment model.TicketAttachment
	require.NoError(t, db.Where("ticket_id = ?", ticket.Id).Take(&attachment).Error)
	missingTicketId := ticket.Id + 1000

	// 同一请求分别打到他人工单和不存在的工单，响应必须完全一致，不能泄露工单是否存在
	sameAsMissing := func(name string, handler gin.HandlerFunc, method string, buildBody func() (io.Reader, string), params func(ticketId int) gin.Params) {
		t.Run(name, func(t *testing.T) {
			var responses []*httptest.ResponseRecorder
			for _, ticketId := range []int{ticket.Id, missingTicketId} {
				body, contentType := buildBody()
				responses = append(responses, serveTicketRequest(t, handler, intruder, ticketTestRequest{
					method: method, target: "/api/ticket/self/" + strconv.Itoa(ticketId), body: body, contentType: contentType,
					params: params(ticketId),
				}))
			}
			assert.Equal(t, responses[1].Code, responses[0].Code)
			assert.Equal(t, responses[1].Body.String(), responses[0].Body.String())
			assert.Contains(t, responses[0].Body.String(), `"success":false`)
			assert.NotContains(t, responses[0].Body.String(), "Private")
		})
	}
	noBody := func() (io.Reader, string) { return nil, "" }
	replyBody := func() (io.Reader, string) {
		return ticketMultipartBody(t, map[string]string{"content": "let me in"})
	}
	byTicket := func(ticketId int) gin.Params { return ticketParams(ticketId) }
	byAttachment := func(ticketId int) gin.Params { return ticketParams(ticketId, attachment.Id) }

	sameAsMissing("detail", GetSelfTicket, http.MethodGet, noBody, byTicket)
	sameAsMissing("reply", ReplySelfTicket, http.MethodPost, replyBody, byTicket)
	sameAsMissing("close", CloseSelfTicket, http.MethodPost, noBody, byTicket)
	sameAsMissing("attachment", GetSelfTicketAttachment, http.MethodGet, noBody, byAttachment)

	attachmentRecorder := serveTicketRequest(t, GetSelfTicketAttachment, intruder, ticketTestRequest{
		method: http.MethodGet, target: "/api/ticket/self/attachment", params: ticketParams(ticket.Id, attachment.Id),
	})
	assert.Equal(t, http.StatusNotFound, attachmentRecorder.Code)
	assert.NotEqual(t, attachment.Data, attachmentRecorder.Body.Bytes())

	var stored model.Ticket
	require.NoError(t, db.First(&stored, ticket.Id).Error)
	assert.Equal(t, model.TicketStatusPending, stored.Status, "the intruder's close must not apply")
	assert.Equal(t, 1, stored.MessageCount, "the intruder's reply must not be stored")
}

func TestTicketAttachmentResponseHeaders(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	owner := createTicketControllerUser(t, db, "ticket-viewer", common.RoleCommonUser)
	admin := createTicketControllerUser(t, db, "ticket-admin-viewer", common.RoleAdminUser)
	ticket := createTicketViaAPI(t, owner, "Screenshot", pngUpload(ticketTestPNG(t)))
	var attachment model.TicketAttachment
	require.NoError(t, db.Where("ticket_id = ?", ticket.Id).Take(&attachment).Error)

	for name, request := range map[string]struct {
		handler gin.HandlerFunc
		actor   *model.User
	}{
		"owner": {handler: GetSelfTicketAttachment, actor: owner},
		"admin": {handler: AdminGetTicketAttachment, actor: admin},
	} {
		t.Run(name, func(t *testing.T) {
			recorder := serveTicketRequest(t, request.handler, request.actor, ticketTestRequest{
				method: http.MethodGet, target: "/api/ticket/attachment", params: ticketParams(ticket.Id, attachment.Id),
			})
			require.Equal(t, http.StatusOK, recorder.Code, recorder.Body.String())
			header := recorder.Header()
			assert.Equal(t, "image/png", header.Get("Content-Type"))
			assert.Equal(t, "nosniff", header.Get("X-Content-Type-Options"))
			assert.Equal(t, "default-src 'none'; img-src 'self' data: blob:; sandbox", header.Get("Content-Security-Policy"))
			assert.Equal(t, fmt.Sprintf(`inline; filename="ticket-%d-%d.png"`, ticket.Id, attachment.Id), header.Get("Content-Disposition"))
			assert.Equal(t, "private, no-store", header.Get("Cache-Control"))
			assert.Equal(t, "same-origin", header.Get("Cross-Origin-Resource-Policy"))
			assert.Equal(t, attachment.Data, recorder.Body.Bytes())
		})
	}

	// 附件必须属于路径中的工单
	other := createTicketViaAPI(t, owner, "Another")
	recorder := serveTicketRequest(t, AdminGetTicketAttachment, admin, ticketTestRequest{
		method: http.MethodGet, target: "/api/ticket/admin/attachment", params: ticketParams(other.Id, attachment.Id),
	})
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}

func TestUserTicketDetailHidesSupportIdentity(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	owner := createTicketControllerUser(t, db, "ticket-customer", common.RoleCommonUser)
	admin := createTicketControllerUser(t, db, "support-alice-secret", common.RoleAdminUser)
	ticket := createTicketViaAPI(t, owner, "Help")

	body, contentType := ticketMultipartBody(t, map[string]string{"content": "On it"})
	recorder := serveTicketRequest(t, AdminReplyTicket, admin, ticketTestRequest{
		method: http.MethodPost, target: "/api/ticket/admin/reply", body: body, contentType: contentType, params: ticketParams(ticket.Id),
	})
	require.True(t, decodeTicketResponse(t, recorder).Success)

	recorder = serveTicketRequest(t, GetSelfTicket, owner, ticketTestRequest{
		method: http.MethodGet, target: "/api/ticket/self/detail", params: ticketParams(ticket.Id),
	})
	response := decodeTicketResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	assert.NotContains(t, recorder.Body.String(), admin.Username)
	var detail struct {
		Ticket   map[string]any   `json:"ticket"`
		Messages []map[string]any `json:"messages"`
	}
	require.NoError(t, common.Unmarshal(response.Data, &detail))
	require.Len(t, detail.Messages, 2)
	supportMessage := detail.Messages[1]
	assert.Equal(t, model.TicketRoleAdmin, supportMessage["author_role"])
	assert.NotContains(t, supportMessage, "author_name")
	assert.NotContains(t, supportMessage, "user_id")
	assert.NotContains(t, detail.Ticket, "username")
	assert.Equal(t, false, detail.Ticket["user_unread"])

	// 管理端可以看到客服用户名
	recorder = serveTicketRequest(t, AdminGetTicket, admin, ticketTestRequest{
		method: http.MethodGet, target: "/api/ticket/admin/detail", params: ticketParams(ticket.Id),
	})
	assert.Contains(t, recorder.Body.String(), admin.Username)
}

func TestAdminReplyWithStatusUpdatesTicket(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	owner := createTicketControllerUser(t, db, "ticket-requester", common.RoleCommonUser)
	admin := createTicketControllerUser(t, db, "ticket-resolver", common.RoleAdminUser)
	ticket := createTicketViaAPI(t, owner, "Resolve me")

	body, contentType := ticketMultipartBody(t, map[string]string{"content": "Done", "status": "7"})
	recorder := serveTicketRequest(t, AdminReplyTicket, admin, ticketTestRequest{
		method: http.MethodPost, target: "/api/ticket/admin/reply", body: body, contentType: contentType, params: ticketParams(ticket.Id),
	})
	response := decodeTicketResponse(t, recorder)
	assert.False(t, response.Success)
	assert.Equal(t, i18n.MsgTicketInvalidStatus, response.Message)

	body, contentType = ticketMultipartBody(t, map[string]string{"content": "Done", "status": strconv.Itoa(model.TicketStatusResolved)})
	recorder = serveTicketRequest(t, AdminReplyTicket, admin, ticketTestRequest{
		method: http.MethodPost, target: "/api/ticket/admin/reply", body: body, contentType: contentType, params: ticketParams(ticket.Id),
	})
	response = decodeTicketResponse(t, recorder)
	require.True(t, response.Success, response.Message)
	var message model.TicketMessage
	require.NoError(t, common.Unmarshal(response.Data, &message))
	assert.Equal(t, "Done", message.Content)
	assert.Equal(t, admin.Username, message.AuthorName)

	var stored model.Ticket
	require.NoError(t, db.First(&stored, ticket.Id).Error)
	assert.Equal(t, model.TicketStatusResolved, stored.Status)
	assert.True(t, stored.UserUnread)
	assert.False(t, stored.AdminUnread)
	assert.Equal(t, 2, stored.MessageCount, "the rejected reply must not be stored")

	// 通过回复修改状态时，状态变更同样要留下审计记录
	var audits []model.Log
	require.NoError(t, db.Where("type = ?", model.LogTypeManage).Order("id ASC").Find(&audits).Error)
	require.Len(t, audits, 2)
	assert.Equal(t, admin.Id, audits[0].UserId)
	assert.Equal(t, fmt.Sprintf("Replied to ticket #%d", ticket.Id), audits[0].Content)
	assert.Equal(t, admin.Id, audits[1].UserId)
	assert.Equal(t, fmt.Sprintf("Changed status of ticket #%d to %d", ticket.Id, model.TicketStatusResolved), audits[1].Content)
}

func TestAdminTicketKeywordSearchIsLiteral(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	admin := createTicketControllerUser(t, db, "ticket-searcher", common.RoleAdminUser)
	underscoreUser := createTicketControllerUser(t, db, "bob_1", common.RoleCommonUser)
	lookalikeUser := createTicketControllerUser(t, db, "bobx1", common.RoleCommonUser)
	percent := createTicketViaAPI(t, underscoreUser, "Got 100% off")
	plain := createTicketViaAPI(t, lookalikeUser, "Got 100 off")

	search := func(keyword string) []model.Ticket {
		recorder := serveTicketRequest(t, AdminGetTickets, admin, ticketTestRequest{
			method: http.MethodGet, target: "/api/ticket/admin?keyword=" + url.QueryEscape(keyword),
		})
		response := decodeTicketResponse(t, recorder)
		require.True(t, response.Success, response.Message)
		var page struct {
			Total int            `json:"total"`
			Items []model.Ticket `json:"items"`
		}
		require.NoError(t, common.Unmarshal(response.Data, &page))
		require.Equal(t, len(page.Items), page.Total)
		return page.Items
	}

	matches := search("%")
	require.Len(t, matches, 1, "%% must match a literal percent sign only")
	assert.Equal(t, percent.Id, matches[0].Id)

	matches = search("bob_")
	require.Len(t, matches, 1, "_ must match a literal underscore only")
	assert.Equal(t, percent.Id, matches[0].Id)
	assert.Equal(t, underscoreUser.Username, matches[0].Username)

	matches = search("100 off")
	require.Len(t, matches, 1)
	assert.Equal(t, plain.Id, matches[0].Id)

	matches = search(strconv.Itoa(plain.Id))
	require.NotEmpty(t, matches)
	assert.Equal(t, plain.Id, matches[0].Id)

	// 界面显示的工单号带 #，复制过来也能按 ID 搜到
	matches = search("#" + strconv.Itoa(plain.Id))
	require.Len(t, matches, 1)
	assert.Equal(t, plain.Id, matches[0].Id)
}

func TestAdminUpdateTicketStatus(t *testing.T) {
	db := setupTicketControllerTestDB(t)
	owner := createTicketControllerUser(t, db, "ticket-status-owner", common.RoleCommonUser)
	admin := createTicketControllerUser(t, db, "ticket-status-admin", common.RoleAdminUser)
	ticket := createTicketViaAPI(t, owner, "Status")

	for _, body := range []string{`{"status":0}`, `{"status":5}`, `{"status":"4"}`, `not json`} {
		recorder := serveTicketRequest(t, AdminUpdateTicketStatus, admin, ticketTestRequest{
			method: http.MethodPut, target: "/api/ticket/admin/status", body: strings.NewReader(body),
			contentType: "application/json", params: ticketParams(ticket.Id),
		})
		response := decodeTicketResponse(t, recorder)
		assert.False(t, response.Success, body)
		assert.Equal(t, i18n.MsgTicketInvalidStatus, response.Message, body)
	}

	recorder := serveTicketRequest(t, AdminUpdateTicketStatus, admin, ticketTestRequest{
		method: http.MethodPut, target: "/api/ticket/admin/status", body: strings.NewReader(`{"status":4}`),
		contentType: "application/json", params: ticketParams(ticket.Id),
	})
	response := decodeTicketResponse(t, recorder)
	require.True(t, response.Success, response.Message)

	var stored model.Ticket
	require.NoError(t, db.First(&stored, ticket.Id).Error)
	assert.Equal(t, model.TicketStatusClosed, stored.Status)
	assert.NotZero(t, stored.ClosedTime)
	assert.True(t, stored.UserUnread)

	var audit model.Log
	require.NoError(t, db.Where("type = ?", model.LogTypeManage).Take(&audit).Error)
	assert.Equal(t, fmt.Sprintf("Changed status of ticket #%d to %d", ticket.Id, model.TicketStatusClosed), audit.Content)
	assert.Contains(t, audit.Other, `"target_user_id":`+strconv.Itoa(owner.Id))
}

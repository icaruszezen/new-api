package controller

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/i18n"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/oauth"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// legacyGitHubOAuthProvider follows the GitHub provider contract: the numeric
// account ID identifies the user, Extra["legacy_id"] carries the login name and
// the verified email list is served on demand. Lookups go through the real
// github_id model queries.
type legacyGitHubOAuthProvider struct {
	authFlowTestOAuthProvider
	providerUserID     string
	legacyID           string
	verifiedEmails     []string
	verifiedEmailsErr  error
	verifiedEmailCalls int
}

func (*legacyGitHubOAuthProvider) GetName() string { return "GitHub" }
func (provider *legacyGitHubOAuthProvider) GetUserInfo(context.Context, *oauth.OAuthToken) (*oauth.OAuthUser, error) {
	return &oauth.OAuthUser{ProviderUserID: provider.providerUserID, Username: provider.legacyID, Extra: map[string]any{"legacy_id": provider.legacyID}}, nil
}
func (provider *legacyGitHubOAuthProvider) GetVerifiedEmails(context.Context, *oauth.OAuthToken) ([]string, error) {
	provider.verifiedEmailCalls++
	return provider.verifiedEmails, provider.verifiedEmailsErr
}
func (*legacyGitHubOAuthProvider) IsUserIDTaken(providerUserID string) bool {
	return model.IsGitHubIdAlreadyTaken(providerUserID)
}
func (*legacyGitHubOAuthProvider) FillUserByProviderID(user *model.User, providerUserID string) error {
	user.GitHubId = providerUserID
	return user.FillUserByGitHubId()
}
func (*legacyGitHubOAuthProvider) SetProviderUserID(user *model.User, providerUserID string) {
	user.GitHubId = providerUserID
}
func (*legacyGitHubOAuthProvider) GetProviderPrefix() string    { return "github_" }
func (*legacyGitHubOAuthProvider) ProviderUserIDColumn() string { return "github_id" }

// A legacy GitHub binding stores a login name that can be renamed and
// re-registered by someone else, so it must never sign a user in on its own.
func TestOAuthLoginLegacyGitHubBindingRequiresVerifiedEmail(t *testing.T) {
	require.NoError(t, i18n.Init())
	const declined = "This GitHub account cannot be linked to an existing account automatically, please sign in or register another way and then link GitHub in account settings"
	tests := []struct {
		name              string
		existingGitHubID  string
		withoutEmail      bool
		legacyID          string
		verifiedEmails    []string
		verifiedEmailsErr error
		expectLogin       bool
		expectMigration   bool
		expectDeclined    bool
		expectEmailCalls  int
	}{
		{
			name: "verified email match migrates", existingGitHubID: "octocat-legacy", legacyID: "octocat-legacy",
			verifiedEmails: []string{"other@example.com", " Legacy-GitHub@Example.com "}, expectLogin: true, expectMigration: true, expectEmailCalls: 1,
		},
		{
			name: "no matching verified email declines", existingGitHubID: "octocat-legacy", legacyID: "octocat-legacy",
			verifiedEmails: []string{"other@example.com"}, expectDeclined: true, expectEmailCalls: 1,
		},
		{
			name: "verified emails failure declines", existingGitHubID: "octocat-legacy", legacyID: "octocat-legacy",
			verifiedEmailsErr: errors.New("emails unavailable"), expectDeclined: true, expectEmailCalls: 1,
		},
		{
			name: "account without email declines before asking the provider", existingGitHubID: "octocat-legacy", withoutEmail: true,
			legacyID: "octocat-legacy", verifiedEmails: []string{"legacy-github@example.com"}, expectDeclined: true,
		},
		{
			// An all-digit login equal to another user's numeric GitHub ID must not
			// reach that account; with registration disabled the login just fails.
			name: "numeric login never matches a numeric binding", existingGitHubID: "424242", legacyID: "424242",
			verifiedEmails: []string{"legacy-github@example.com"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := setupLegacyGitHubOAuthTest(t)
			existing := &model.User{
				Username: "legacy-github", Password: "password-placeholder", Email: "legacy-github@example.com",
				Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default",
				AffCode: "legacy-github", AuthVersion: 1, GitHubId: test.existingGitHubID,
			}
			if test.withoutEmail {
				existing.Email = ""
			}
			require.NoError(t, db.Create(existing).Error)
			provider := &legacyGitHubOAuthProvider{providerUserID: "900001", legacyID: test.legacyID, verifiedEmails: test.verifiedEmails, verifiedEmailsErr: test.verifiedEmailsErr}

			response := legacyGitHubOAuthLogin(t, provider)

			var result struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
				Data    struct {
					User struct {
						Id int `json:"id"`
					} `json:"user"`
				} `json:"data"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
			require.Equal(t, test.expectLogin, result.Success, response.Body.String())
			assert.Equal(t, test.expectEmailCalls, provider.verifiedEmailCalls)

			var reloaded model.User
			require.NoError(t, db.First(&reloaded, existing.Id).Error)
			if test.expectMigration {
				assert.Equal(t, existing.Id, result.Data.User.Id)
				assert.Equal(t, "900001", reloaded.GitHubId)
				return
			}
			assert.Equal(t, test.existingGitHubID, reloaded.GitHubId, "the existing binding must stay untouched")
			assert.Empty(t, response.Header().Values("Set-Cookie"))
			var sessions int64
			require.NoError(t, db.Model(&model.UserSession{}).Where("user_id = ?", existing.Id).Count(&sessions).Error)
			assert.Zero(t, sessions)
			if test.expectDeclined {
				assert.Equal(t, declined, result.Message)
			}
		})
	}
}

func setupLegacyGitHubOAuthTest(t *testing.T) *gorm.DB {
	t.Helper()
	previousDB := model.DB
	previousLogDB := model.LOG_DB
	previousType := common.MainDatabaseType()
	previousRedis := common.RedisEnabled
	previousSecret := common.SessionSecret
	previousRegister := common.RegisterEnabled
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&model.User{}, &model.UserSession{}, &model.AuthFlow{}, &model.Log{}))
	model.DB = db
	model.LOG_DB = db
	common.SetMainDatabaseType(common.DatabaseTypeSQLite)
	common.RedisEnabled = false
	common.SessionSecret = "legacy-github-oauth-test-secret"
	common.RegisterEnabled = false
	t.Cleanup(func() {
		model.DB = previousDB
		model.LOG_DB = previousLogDB
		common.SetMainDatabaseType(previousType)
		common.RedisEnabled = previousRedis
		common.SessionSecret = previousSecret
		common.RegisterEnabled = previousRegister
	})
	return db
}

// legacyGitHubOAuthLogin registers provider for one test and completes an OAuth
// login callback with it.
func legacyGitHubOAuthLogin(t *testing.T, provider *legacyGitHubOAuthProvider) *httptest.ResponseRecorder {
	t.Helper()
	const slug = "github-legacy-login-test"
	oauth.Register(slug, provider)
	t.Cleanup(func() { oauth.Unregister(slug) })
	token, _, err := model.CreateAuthFlow(model.AuthFlowCreate{Purpose: model.AuthFlowPurposeOAuth, Provider: slug, Intent: model.AuthFlowIntentLogin, Payload: `{}`, ExpiresAt: time.Now().Add(time.Minute)})
	require.NoError(t, err)
	router := gin.New()
	router.GET("/api/oauth/:provider", HandleOAuth)
	request := httptest.NewRequest(http.MethodGet, "/api/oauth/"+slug+"?state="+token+"&code=provider-code", nil)
	request.Header.Set("Accept-Language", "en")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

// Declined legacy users are told to sign in another way and relink GitHub in
// account settings, so the bind flow must compare only numeric account IDs.
func TestOAuthBindIgnoresLegacyGitHubUsernames(t *testing.T) {
	tests := []struct {
		name          string
		ownGitHubID   string
		otherGitHubID string
		legacyID      string
		expectBound   bool
	}{
		{name: "another account's legacy username does not block binding", otherGitHubID: "octocat-legacy", legacyID: "octocat-legacy", expectBound: true},
		{name: "own legacy username is replaced", ownGitHubID: "octocat-legacy", otherGitHubID: "unrelated-legacy", legacyID: "octocat-legacy", expectBound: true},
		{name: "numeric legacy value does not collide", otherGitHubID: "424242", legacyID: "424242", expectBound: true},
		{name: "numeric ID held by another account is rejected", otherGitHubID: "900001", legacyID: "octocat-legacy"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.NoError(t, i18n.Init())
			db := setupLegacyGitHubOAuthTest(t)
			user := &model.User{Username: "binding-user", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "binding-user", AuthVersion: 1, GitHubId: test.ownGitHubID}
			require.NoError(t, db.Create(user).Error)
			other := &model.User{Username: "other-github", Role: common.RoleCommonUser, Status: common.UserStatusEnabled, Group: "default", AffCode: "other-github", AuthVersion: 1, GitHubId: test.otherGitHubID}
			require.NoError(t, db.Create(other).Error)

			const slug = "github-legacy-bind-test"
			oauth.Register(slug, &legacyGitHubOAuthProvider{providerUserID: "900001", legacyID: test.legacyID})
			t.Cleanup(func() { oauth.Unregister(slug) })
			flowToken, _, err := model.CreateAuthFlow(model.AuthFlowCreate{
				Purpose: model.AuthFlowPurposeOAuth, Provider: slug, Intent: model.AuthFlowIntentBind,
				UserId: user.Id, SessionId: "binding-session", Payload: `{}`, ExpiresAt: time.Now().Add(time.Minute),
			})
			require.NoError(t, err)
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set("id", user.Id)
				c.Set("session_id", "binding-session")
				c.Set("auth_version", int64(1))
				c.Set("session_version", int64(1))
				c.Next()
			})
			router.GET("/api/oauth/:provider", HandleOAuth)
			request := httptest.NewRequest(http.MethodGet, "/api/oauth/"+slug+"?state="+flowToken+"&code=provider-code", nil)
			request.Header.Set("Accept-Language", "en")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			var result struct {
				Success bool   `json:"success"`
				Message string `json:"message"`
			}
			require.NoError(t, common.Unmarshal(response.Body.Bytes(), &result))
			assert.Equal(t, test.expectBound, result.Success, response.Body.String())

			var bound, untouched model.User
			require.NoError(t, db.First(&bound, user.Id).Error)
			require.NoError(t, db.First(&untouched, other.Id).Error)
			assert.Equal(t, test.otherGitHubID, untouched.GitHubId)
			if test.expectBound {
				assert.Equal(t, "900001", bound.GitHubId)
				return
			}
			assert.Equal(t, test.ownGitHubID, bound.GitHubId)
			assert.Equal(t, "This GitHub account has already been bound", result.Message)
		})
	}
}

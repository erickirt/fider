package handlers_test

import (
	"context"
	"testing"

	"github.com/getfider/fider/app/models/dto"
	"github.com/getfider/fider/app/models/entity"
	"github.com/getfider/fider/app/models/enum"

	"github.com/getfider/fider/app/services/oauth"

	"github.com/getfider/fider/app"

	"net/http"
	"net/url"

	"github.com/getfider/fider/app/handlers"
	"github.com/getfider/fider/app/middlewares"
	"github.com/getfider/fider/app/models/cmd"
	"github.com/getfider/fider/app/models/query"
	. "github.com/getfider/fider/app/pkg/assert"
	"github.com/getfider/fider/app/pkg/bus"
	"github.com/getfider/fider/app/pkg/jwt"
	"github.com/getfider/fider/app/pkg/mock"
	"github.com/getfider/fider/app/pkg/web"
)

func TestSignOutHandler(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	code, response := server.
		WithURL("http://demo.test.fider.io/signout").
		AddCookie(web.CookieAuthName, "some-value").
		Execute(handlers.SignOut())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("/")
	Expect(response.Header().Get("Set-Cookie")).ContainsSubstring(web.CookieAuthName + "=; Path=/; Expires=")
	Expect(response.Header().Get("Set-Cookie")).ContainsSubstring("Max-Age=0; HttpOnly")
}

func TestSignInByOAuthHandler_RootRedirect(t *testing.T) {
	RegisterT(t)
	bus.Init(&oauth.Service{})

	server := mock.NewServer()
	code, _ := server.
		AddParam("provider", app.FacebookProvider).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		WithURL("http://avengers.test.fider.io/oauth/facebook?redirect=http://avengers.test.fider.io").
		Use(middlewares.Session()).
		Execute(handlers.SignInByOAuth())

	Expect(code).Equals(http.StatusTemporaryRedirect)
}

func TestSignInByOAuthHandler_PathRedirect(t *testing.T) {
	RegisterT(t)
	bus.Init(&oauth.Service{})

	server := mock.NewServer()
	code, _ := server.
		AddParam("provider", app.FacebookProvider).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		WithURL("http://avengers.test.fider.io/oauth/facebook?redirect=http://avengers.test.fider.io/something").
		Use(middlewares.Session()).
		Execute(handlers.SignInByOAuth())

	Expect(code).Equals(http.StatusTemporaryRedirect)
}

func TestSignInByOAuthHandler_EvilRedirect(t *testing.T) {
	RegisterT(t)
	bus.Init(&oauth.Service{})

	server := mock.NewServer()
	code, _ := server.
		AddParam("provider", app.FacebookProvider).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		WithURL("http://avengers.test.fider.io/oauth/facebook?redirect=http://evil.com").
		Use(middlewares.Session()).
		Execute(handlers.SignInByOAuth())

	Expect(code).Equals(http.StatusForbidden)
}

func TestSignInByOAuthHandler_EvilRedirect2(t *testing.T) {
	RegisterT(t)
	bus.Init(&oauth.Service{})

	server := mock.NewServer()
	code, _ := server.
		AddParam("provider", app.FacebookProvider).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		WithURL("http://avengers.test.fider.io/oauth/facebook?redirect=http://avengers.test.fider.io.evil.com").
		Use(middlewares.Session()).
		Execute(handlers.SignInByOAuth())

	Expect(code).Equals(http.StatusForbidden)
}

func TestSignInByOAuthHandler_InvalidURL(t *testing.T) {
	RegisterT(t)
	bus.Init(&oauth.Service{})

	state, _ := jwt.Encode(jwt.OAuthStateClaims{
		Redirect:   "http://avengers.test.fider.io",
		Identifier: "MY_SESSION_ID",
	})

	server := mock.NewServer()
	code, response := server.
		AddParam("provider", app.FacebookProvider).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		WithURL("http://avengers.test.fider.io/oauth/facebook").
		Use(middlewares.Session()).
		Execute(handlers.SignInByOAuth())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("https://www.facebook.com/v3.2/dialog/oauth?client_id=FB_CL_ID&redirect_uri=http%3A%2F%2Flogin.test.fider.io%2Foauth%2Ffacebook%2Fcallback&response_type=code&scope=public_profile+email&state=" + state)
}

func TestSignInByOAuthHandler_AuthenticatedUser(t *testing.T) {
	RegisterT(t)
	bus.Init(&oauth.Service{})

	server := mock.NewServer()
	code, response := server.
		AsUser(mock.JonSnow).
		AddParam("provider", app.FacebookProvider).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		WithURL("http://avengers.test.fider.io/oauth/facebook?redirect=http://avengers.test.fider.io").
		Use(middlewares.Session()).
		Execute(handlers.SignInByOAuth())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("http://avengers.test.fider.io")
}

func TestSignInByOAuthHandler_AuthenticatedUser_UsingEcho(t *testing.T) {
	RegisterT(t)
	bus.Init(&oauth.Service{})

	server := mock.NewServer()
	code, response := server.
		AsUser(mock.JonSnow).
		AddParam("provider", app.FacebookProvider).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		WithURL("http://avengers.test.fider.io/oauth/facebook?redirect=http://avengers.test.fider.io/oauth/facebook/echo").
		Use(middlewares.Session()).
		Execute(handlers.SignInByOAuth())

	state, _ := jwt.Encode(jwt.OAuthStateClaims{
		Redirect:   "http://avengers.test.fider.io/oauth/facebook/echo",
		Identifier: "MY_SESSION_ID",
	})

	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("https://www.facebook.com/v3.2/dialog/oauth?client_id=FB_CL_ID&redirect_uri=http%3A%2F%2Flogin.test.fider.io%2Foauth%2Ffacebook%2Fcallback&response_type=code&scope=public_profile+email&state=" + state)
}

func TestCallbackHandler_InvalidState(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	code, _ := server.
		WithURL("http://login.test.fider.io/oauth/callback?state=abc").
		AddParam("provider", app.FacebookProvider).
		Execute(handlers.OAuthCallback())

	Expect(code).Equals(http.StatusForbidden)
}

func TestCallbackHandler_InvalidCode(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	state, _ := jwt.Encode(jwt.OAuthStateClaims{
		Redirect:   "http://avengers.test.fider.io",
		Identifier: "",
	})

	code, response := server.
		WithURL("http://login.test.fider.io/oauth/callback?state="+state).
		AddParam("provider", app.FacebookProvider).
		Execute(handlers.OAuthCallback())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("http://avengers.test.fider.io")
}

func TestCallbackHandler_SignIn(t *testing.T) {
	RegisterT(t)

	state, _ := jwt.Encode(jwt.OAuthStateClaims{
		Redirect:   "http://avengers.test.fider.io",
		Identifier: "888",
	})

	server := mock.NewServer()
	code, response := server.
		WithURL("http://login.test.fider.io/oauth/callback?state="+state+"&code=123").
		AddParam("provider", app.FacebookProvider).
		Execute(handlers.OAuthCallback())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("http://avengers.test.fider.io/oauth/facebook/token?code=123&identifier=888&redirect=%2F")
}

func TestCallbackHandler_SignIn_WithPath(t *testing.T) {
	RegisterT(t)
	server := mock.NewServer()

	state, _ := jwt.Encode(jwt.OAuthStateClaims{
		Redirect:   "http://avengers.test.fider.io/some-page",
		Identifier: "888",
	})

	code, response := server.
		WithURL("http://login.test.fider.io/oauth/callback?state="+state+"&code=123").
		AddParam("provider", app.FacebookProvider).
		Execute(handlers.OAuthCallback())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("http://avengers.test.fider.io/oauth/facebook/token?code=123&identifier=888&redirect=%2Fsome-page")
}

func TestCallbackHandler_SignUp(t *testing.T) {
	RegisterT(t)

	oauthUser := &dto.OAuthUserProfile{
		ID:    "FB123",
		Name:  "Jon Snow",
		Email: "jon.snow@got.com",
	}

	bus.AddHandler(func(ctx context.Context, q *query.GetOAuthProfile) error {
		if q.Provider == app.FacebookProvider && q.Code == "123" {
			q.Result = oauthUser
			return nil
		}
		return app.ErrNotFound
	})

	state, _ := jwt.Encode(jwt.OAuthStateClaims{
		Redirect:   "http://demo.test.fider.io/signup",
		Identifier: "",
	})

	server := mock.NewServer()
	code, response := server.
		WithURL("http://login.test.fider.io/oauth/callback?state="+state+"&code=123").
		AddParam("provider", app.FacebookProvider).
		Execute(handlers.OAuthCallback())
	Expect(code).Equals(http.StatusTemporaryRedirect)

	location, _ := url.Parse(response.Header().Get("Location"))
	Expect(location.Host).Equals("demo.test.fider.io")
	Expect(location.Scheme).Equals("http")
	Expect(location.Path).Equals("/signup")
	ExpectOAuthToken(location.Query().Get("token"), &jwt.OAuthClaims{
		OAuthProvider: "facebook",
		OAuthID:       oauthUser.ID,
		OAuthName:     oauthUser.Name,
		OAuthEmail:    oauthUser.Email,
	})
}

func TestOAuthTokenHandler_ExistingUserAndProvider(t *testing.T) {
	RegisterT(t)

	oauthUser := &dto.OAuthUserProfile{
		ID:    "FB123",
		Name:  "Jon Snow",
		Email: "jon.snow@got.com",
	}

	bus.AddHandler(func(ctx context.Context, q *query.GetOAuthProfile) error {
		if q.Provider == app.FacebookProvider && q.Code == "123" {
			q.Result = oauthUser
			return nil
		}
		return app.ErrNotFound
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetUserByProvider) error {
		if q.Provider == app.FacebookProvider && q.UID == oauthUser.ID {
			q.Result = mock.JonSnow
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	code, response := server.
		WithURL("http://demo.test.fider.io/oauth/facebook/token?code=123&identifier=MY_SESSION_ID&redirect=/hello").
		OnTenant(mock.DemoTenant).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		AddParam("provider", app.FacebookProvider).
		Use(middlewares.Session()).
		Execute(handlers.OAuthToken())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("/hello")
	ExpectFiderAuthCookie(response, mock.JonSnow)
}

func TestOAuthTokenHandler_NewUser(t *testing.T) {
	RegisterT(t)

	var registeredUser *entity.User
	bus.AddHandler(func(ctx context.Context, c *cmd.RegisterUser) error {
		registeredUser = c.User
		return nil
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetUserByProvider) error {
		Expect(q.Provider).Equals(app.FacebookProvider)
		Expect(q.UID).Equals("FB456")
		return app.ErrNotFound
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetUserByEmail) error {
		Expect(q.Email).Equals("some.guy@facebook.com")
		return app.ErrNotFound
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetCustomOAuthConfigByProvider) error {
		Expect(q.Provider).Equals(app.FacebookProvider)
		return app.ErrNotFound
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetOAuthProfile) error {
		Expect(q.Provider).Equals(app.FacebookProvider)
		Expect(q.Code).Equals("456")

		q.Result = &dto.OAuthUserProfile{
			ID:    "FB456",
			Name:  "Some Facebook Guy",
			Email: "some.guy@facebook.com",
		}
		return nil
	})

	server := mock.NewServer()
	code, response := server.
		WithURL("http://demo.test.fider.io/oauth/facebook/token?code=456&identifier=MY_SESSION_ID&redirect=/hello").
		OnTenant(mock.DemoTenant).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		AddParam("provider", app.FacebookProvider).
		Use(middlewares.Session()).
		Execute(handlers.OAuthToken())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("/hello")

	Expect(registeredUser.Name).Equals("Some Facebook Guy")

	ExpectFiderAuthCookie(response, registeredUser)
}

func TestOAuthTokenHandler_NewUserWithoutEmail(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	var newUser *entity.User
	bus.AddHandler(func(ctx context.Context, c *cmd.RegisterUser) error {
		c.User.ID = 1
		newUser = c.User
		return nil
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetUserByProvider) error {
		return app.ErrNotFound
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetCustomOAuthConfigByProvider) error {
		return app.ErrNotFound
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetOAuthProfile) error {
		if q.Provider == app.FacebookProvider && q.Code == "798" {
			q.Result = &dto.OAuthUserProfile{
				ID:    "FB798",
				Name:  "Mark",
				Email: "",
			}
			return nil
		}
		return app.ErrNotFound
	})

	code, response := server.
		WithURL("http://demo.test.fider.io/oauth/facebook/token?code=798&identifier=MY_SESSION_ID&redirect=/").
		OnTenant(mock.DemoTenant).
		AddParam("provider", app.FacebookProvider).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		Use(middlewares.Session()).
		Execute(handlers.OAuthToken())

	Expect(newUser.ID).Equals(1)
	Expect(newUser.Name).Equals("Mark")
	Expect(newUser.Providers).HasLen(1)

	Expect(code).Equals(http.StatusTemporaryRedirect)

	Expect(response.Header().Get("Location")).Equals("/")
	ExpectFiderAuthCookie(response, &entity.User{
		ID:   1,
		Name: "Mark",
	})
}

func TestOAuthTokenHandler_ExistingUser_WithoutEmail(t *testing.T) {
	RegisterT(t)

	user := &entity.User{
		ID:     3,
		Name:   "Some Facebook Guy",
		Email:  "",
		Tenant: mock.DemoTenant,
		Providers: []*entity.UserProvider{
			{UID: "FB456", Name: app.FacebookProvider},
		},
	}

	bus.AddHandler(func(ctx context.Context, q *query.GetOAuthProfile) error {
		if q.Provider == app.FacebookProvider && q.Code == "456" {
			q.Result = &dto.OAuthUserProfile{
				ID:    "FB456",
				Name:  "Some Facebook Guy",
				Email: "some.guy@facebook.com",
			}
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()

	bus.AddHandler(func(ctx context.Context, q *query.GetUserByProvider) error {
		if q.Provider == "facebook" && q.UID == "FB456" {
			q.Result = user
			return nil
		}
		return app.ErrNotFound
	})

	code, response := server.
		WithURL("http://demo.test.fider.io/oauth/facebook/token?code=456&identifier=MY_SESSION_ID&redirect=/").
		OnTenant(mock.DemoTenant).
		AddParam("provider", app.FacebookProvider).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		Use(middlewares.Session()).
		Execute(handlers.OAuthToken())

	Expect(code).Equals(http.StatusTemporaryRedirect)

	Expect(response.Header().Get("Location")).Equals("/")
	ExpectFiderAuthCookie(response, &entity.User{
		ID:    3,
		Name:  "Some Facebook Guy",
		Email: "",
	})
}

func TestOAuthTokenHandler_ExistingUser_NewProvider(t *testing.T) {
	RegisterT(t)

	var newProvider *entity.UserProvider
	bus.AddHandler(func(ctx context.Context, c *cmd.RegisterUserProvider) error {
		newProvider = &entity.UserProvider{
			Name: c.ProviderName,
			UID:  c.ProviderUID,
		}
		return nil
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetOAuthProfile) error {
		if q.Provider == app.GoogleProvider && q.Code == "123" {
			q.Result = &dto.OAuthUserProfile{
				ID:    "GO123",
				Name:  "Jon Snow",
				Email: "jon.snow@got.com",
			}
			return nil
		}
		return app.ErrNotFound
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetUserByProvider) error {
		if q.Provider == app.GoogleProvider && q.UID == "GO123" {
			q.Result = mock.JonSnow
			return nil
		}
		return app.ErrNotFound
	})

	server := mock.NewServer()
	code, response := server.
		WithURL("http://demo.test.fider.io/oauth/google/token?code=123&identifier=MY_SESSION_ID&redirect=/").
		OnTenant(mock.DemoTenant).
		AddParam("provider", app.GoogleProvider).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		Use(middlewares.Session()).
		Execute(handlers.OAuthToken())

	Expect(code).Equals(http.StatusTemporaryRedirect)

	Expect(newProvider.Name).Equals("google")
	Expect(newProvider.UID).Equals("GO123")

	Expect(response.Header().Get("Location")).Equals("/")
	ExpectFiderAuthCookie(response, mock.JonSnow)
}

func TestOAuthTokenHandler_NewUser_PrivateSite(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	mock.AvengersTenant.IsPrivate = true

	bus.AddHandler(func(ctx context.Context, q *query.GetUserByEmail) error {
		return app.ErrNotFound
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetUserByProvider) error {
		return app.ErrNotFound
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetCustomOAuthConfigByProvider) error {
		Expect(q.Provider).Equals(app.FacebookProvider)
		return app.ErrNotFound
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetOAuthProfile) error {
		Expect(q.Provider).Equals(app.FacebookProvider)
		Expect(q.Code).Equals("456")
		q.Result = &dto.OAuthUserProfile{
			ID:    "FB456",
			Name:  "Some Facebook Guy",
			Email: "some.guy@facebook.com",
		}
		return nil
	})

	code, response := server.
		WithURL("http://feedback.theavengers.com/oauth/facebook/token?code=456&identifier=MY_SESSION_ID&redirect=/").
		OnTenant(mock.AvengersTenant).
		AddParam("provider", app.FacebookProvider).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		Use(middlewares.Session()).
		Execute(handlers.OAuthToken())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("/not-invited")
	ExpectFiderAuthCookie(response, nil)
}

func TestOAuthTokenHandler_NewUser_PrivateSite_UsingTrustedProvider(t *testing.T) {
	RegisterT(t)

	server := mock.NewServer()
	mock.AvengersTenant.IsPrivate = true

	providerCode := "_jd72hfjv"

	bus.AddHandler(func(ctx context.Context, c *cmd.RegisterUser) error {
		Expect(c.User.Name).Equals("Mark Doe")
		Expect(c.User.Email).Equals("mark.doe@microsoft.com")
		Expect(c.User.Providers).HasLen(1)
		Expect(c.User.Providers[0].UID).Equals("1234-5678")
		Expect(c.User.Providers[0].Name).Equals(providerCode)
		Expect(c.User.Role).Equals(enum.RoleVisitor)

		c.User.ID = 999
		return nil
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetUserByEmail) error {
		return app.ErrNotFound
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetUserByProvider) error {
		return app.ErrNotFound
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetCustomOAuthConfigByProvider) error {
		Expect(q.Provider).Equals(providerCode)
		q.Result = &entity.OAuthConfig{
			Provider:    providerCode,
			DisplayName: "Microsoft AD",
			IsTrusted:   true,
		}
		return nil
	})

	bus.AddHandler(func(ctx context.Context, q *query.GetOAuthProfile) error {
		Expect(q.Provider).Equals(providerCode)
		Expect(q.Code).Equals("000111")
		q.Result = &dto.OAuthUserProfile{
			ID:    "1234-5678",
			Name:  "Mark Doe",
			Email: "mark.doe@microsoft.com",
		}
		return nil
	})

	code, response := server.
		WithURL("http://feedback.theavengers.com/oauth/"+providerCode+"/token?code=000111&identifier=MY_SESSION_ID&redirect=/").
		OnTenant(mock.AvengersTenant).
		AddParam("provider", providerCode).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		Use(middlewares.Session()).
		Execute(handlers.OAuthToken())

	Expect(code).Equals(http.StatusTemporaryRedirect)

	Expect(response.Header().Get("Location")).Equals("/")
	ExpectFiderAuthCookie(response, &entity.User{
		ID:    999,
		Name:  "Mark Doe",
		Email: "mark.doe@microsoft.com",
	})
}

func TestOAuthTokenHandler_InvalidIdentifier(t *testing.T) {
	RegisterT(t)
	server := mock.NewServer()
	mock.AvengersTenant.IsPrivate = true

	code, response := server.
		WithURL("http://feedback.theavengers.com/oauth/facebook/token?code=456&identifier=SOME_OTHER_ID&redirect=/").
		OnTenant(mock.AvengersTenant).
		AddParam("provider", app.FacebookProvider).
		AddCookie(web.CookieSessionName, "MY_SESSION_ID").
		Use(middlewares.Session()).
		Execute(handlers.OAuthToken())

	Expect(code).Equals(http.StatusTemporaryRedirect)
	Expect(response.Header().Get("Location")).Equals("/")
	ExpectFiderAuthCookie(response, nil)
}

func ExpectOAuthToken(token string, expected *jwt.OAuthClaims) {
	user, err := jwt.DecodeOAuthClaims(token)
	Expect(err).IsNil()
	Expect(user.OAuthID).Equals(expected.OAuthID)
	Expect(user.OAuthName).Equals(expected.OAuthName)
	Expect(user.OAuthEmail).Equals(expected.OAuthEmail)
	Expect(user.OAuthProvider).Equals(expected.OAuthProvider)
}

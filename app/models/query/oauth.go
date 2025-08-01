package query

import (
	"github.com/getfider/fider/app/models/dto"
	"github.com/getfider/fider/app/models/entity"
)

type GetCustomOAuthConfigByProvider struct {
	Provider string

	Result *entity.OAuthConfig
}

type ListCustomOAuthConfig struct {
	Result []*entity.OAuthConfig
}

type GetOAuthAuthorizationURL struct {
	Provider   string
	Redirect   string
	Identifier string
	// Code is optional, used to store code of the draft post in the state that is passed through the OAuth flow
	Code string

	Result string
}

type GetOAuthProfile struct {
	Provider string
	Code     string

	Result *dto.OAuthUserProfile
}

type GetOAuthRawProfile struct {
	Provider string
	Code     string

	Result string
}

type ListActiveOAuthProviders struct {
	Result []*dto.OAuthProviderOption
}

type ListAllOAuthProviders struct {
	Result []*dto.OAuthProviderOption
}

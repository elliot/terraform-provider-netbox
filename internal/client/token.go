package client

import (
	"errors"
	"fmt"
	"strings"
	"unicode"
)

// TokenVersion identifies the NetBox API token format.
type TokenVersion int

const (
	// TokenV1 is a legacy 40-character token sent as "Authorization: Token
	// <token>". Deprecated in NetBox 4.6 and removed in 5.0.
	TokenV1 TokenVersion = 1
	// TokenV2 is a "nbt_<key>.<secret>" token (NetBox 4.5+) sent as
	// "Authorization: Bearer <token>".
	TokenV2 TokenVersion = 2
)

// V2TokenPrefix starts every NetBox v2 token.
const V2TokenPrefix = "nbt_"

// v1TokenLength is the fixed length NetBox enforces for v1 tokens.
const v1TokenLength = 40

// ParseTokenVersion reports which NetBox token format tok uses, or an error
// describing why it is neither. A value starting with V2TokenPrefix must be a
// complete v2 token; anything else must be a 40-character v1 token.
func ParseTokenVersion(tok string) (TokenVersion, error) {
	if strings.ContainsFunc(tok, unicode.IsSpace) {
		return 0, errors.New("the token contains whitespace; set only the token itself, without an " +
			"\"Authorization\" scheme such as \"Bearer \" or \"Token \"")
	}
	if rest, ok := strings.CutPrefix(tok, V2TokenPrefix); ok {
		key, secret, ok := strings.Cut(rest, ".")
		if !ok || key == "" || secret == "" {
			return 0, fmt.Errorf("a v2 token must look like %s<key>.<secret>", V2TokenPrefix)
		}
		return TokenV2, nil
	}
	if len(tok) != v1TokenLength {
		return 0, fmt.Errorf("expected a v2 token (%s<key>.<secret>) or a %d-character v1 token, got %d characters",
			V2TokenPrefix, v1TokenLength, len(tok))
	}
	return TokenV1, nil
}

// AuthorizationHeader returns the Authorization header value NetBox expects
// for tok: "Bearer <tok>" for v2 tokens and "Token <tok>" for v1 tokens.
func AuthorizationHeader(tok string) string {
	if strings.HasPrefix(tok, V2TokenPrefix) {
		return "Bearer " + tok
	}
	return "Token " + tok
}

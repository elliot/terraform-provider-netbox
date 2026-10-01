package conv

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

var (
	macRe = regexp.MustCompile(`^[0-9A-Fa-f]{2}(:[0-9A-Fa-f]{2}){5}$`)
	wwnRe = regexp.MustCompile(`^[0-9A-Fa-f]{2}(:[0-9A-Fa-f]{2}){7}$`)
)

// MACAddress validates an EUI-48 address in colon notation. NetBox also
// accepts dashes, dots or bare hex digits but stores colon notation, so any
// other spelling would come back different from the configuration; letter
// case is fine (reads keep the configured case).
func MACAddress() validator.String {
	return stringvalidator.RegexMatches(macRe, "must be a MAC address in colon notation, e.g. aa:bb:cc:dd:ee:ff")
}

// WWN validates an EUI-64 World Wide Name in colon notation, for the same
// reason as MACAddress.
func WWN() validator.String {
	return stringvalidator.RegexMatches(wwnRe, "must be a WWN in colon notation, e.g. 50:01:43:80:12:34:56:78")
}

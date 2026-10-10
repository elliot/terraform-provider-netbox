package conv

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

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

// MaxDecimals validates that a number has at most n digits after the decimal
// point. NetBox stores coordinates as decimals with a fixed scale (xx.yyyyyy)
// and rounds anything finer, so a longer value would come back different from
// the configuration.
func MaxDecimals(n int) validator.Float64 {
	return maxDecimals{n: n}
}

type maxDecimals struct{ n int }

func (v maxDecimals) Description(context.Context) string {
	return fmt.Sprintf("value must have at most %d decimal places", v.n)
}

func (v maxDecimals) MarkdownDescription(ctx context.Context) string { return v.Description(ctx) }

func (v maxDecimals) ValidateFloat64(ctx context.Context, req validator.Float64Request, resp *validator.Float64Response) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	// Shortest representation that round-trips, so 52.520008 counts as six
	// decimals rather than whatever the binary expansion looks like.
	repr := strconv.FormatFloat(req.ConfigValue.ValueFloat64(), 'f', -1, 64)
	if _, frac, ok := strings.Cut(repr, "."); ok && len(frac) > v.n {
		resp.Diagnostics.AddAttributeError(req.Path, "Too many decimal places",
			fmt.Sprintf("%s, got %s.", v.Description(ctx), repr))
	}
}

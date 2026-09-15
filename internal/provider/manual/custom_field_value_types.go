package manual

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework/attr"
)

type (
	attrType  = attr.Type
	attrValue = attr.Value
)

var objectTypeRe = regexp.MustCompile(`^[a-z0-9_]+\.[a-z0-9_]+$`)

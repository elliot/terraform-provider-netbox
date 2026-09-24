# A module type profile defines a JSON schema for the attributes of module
# types that use it (NetBox 4.3+).
resource "netbox_module_type_profile" "line_card" {
  name        = "Line card"
  description = "Attributes common to all line cards"
  schema = jsonencode({
    type = "object"
    properties = {
      ports = { type = "integer", title = "Port count", minimum = 1 }
      speed = { type = "string", title = "Port speed", enum = ["1G", "10G", "25G", "100G"] }
    }
    required = ["ports"]
  })
}

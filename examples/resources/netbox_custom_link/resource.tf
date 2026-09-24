# Custom links render on the object's detail page; text and URL are Jinja2.
resource "netbox_custom_link" "weather" {
  name         = "weather"
  object_types = ["dcim.site"]
  link_text    = "Weather for {{ object.name }}"
  link_url     = "https://weather.example.com/?q={{ object.physical_address | urlencode }}"
  group_name   = "External"
  button_class = "blue"
  new_window   = true
  weight       = 100
}

# Only render when a condition holds (empty link_text hides the link).
resource "netbox_custom_link" "ticket" {
  name         = "ticket"
  object_types = ["dcim.device", "virtualization.virtualmachine"]
  link_text    = "{% if object.cf.ticket %}Ticket {{ object.cf.ticket }}{% endif %}"
  link_url     = "https://tickets.example.com/{{ object.cf.ticket }}"
  button_class = "outline-dark"
}

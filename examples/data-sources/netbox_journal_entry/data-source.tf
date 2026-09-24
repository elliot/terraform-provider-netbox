data "netbox_journal_entry" "example" {
  id = 123
}

output "journal_comment" {
  value = data.netbox_journal_entry.example.comments
}

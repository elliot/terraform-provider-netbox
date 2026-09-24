resource "netbox_ipsec_proposal" "esp_aes256_sha256" {
  name                     = "ESP-AES256-SHA256"
  encryption_algorithm     = "aes-256-cbc"
  authentication_algorithm = "hmac-sha256"
}

resource "netbox_ipsec_proposal" "esp_aes128_sha1" {
  name                     = "ESP-AES128-SHA1"
  encryption_algorithm     = "aes-128-cbc"
  authentication_algorithm = "hmac-sha1"
}

resource "netbox_ipsec_policy" "branch" {
  name        = "IPsec-branch"
  description = "Phase 2 policy with PFS (DH group 14)"
  proposal_ids = [
    netbox_ipsec_proposal.esp_aes256_sha256.id,
    netbox_ipsec_proposal.esp_aes128_sha1.id,
  ]
  pfs_group = 14
}

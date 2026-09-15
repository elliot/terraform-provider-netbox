# IPsec (phase 2) proposal. At least one of encryption_algorithm and
# authentication_algorithm must be set.
resource "netbox_ipsec_proposal" "esp_aes256_sha256" {
  name                     = "ESP-AES256-SHA256"
  description              = "Phase 2 proposal for site-to-site tunnels"
  encryption_algorithm     = "aes-256-cbc"
  authentication_algorithm = "hmac-sha256"
  sa_lifetime_seconds      = 3600
  sa_lifetime_data         = 4608000 # kilobytes
}

# AEAD proposal: the cipher provides integrity, so no authentication algorithm.
resource "netbox_ipsec_proposal" "esp_aes256_gcm" {
  name                 = "ESP-AES256-GCM"
  encryption_algorithm = "aes-256-gcm"
  sa_lifetime_seconds  = 3600
}

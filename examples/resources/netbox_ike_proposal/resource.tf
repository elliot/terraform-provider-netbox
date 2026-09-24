# IKE (phase 1) proposal with a CBC cipher, which needs an explicit
# integrity algorithm. AEAD ciphers (*-gcm) carry their own integrity
# check and must omit authentication_algorithm.
resource "netbox_ike_proposal" "aes256_sha256" {
  name                     = "IKE-AES256-SHA256-DH14"
  description              = "Phase 1 proposal for site-to-site tunnels"
  authentication_method    = "preshared-keys"
  encryption_algorithm     = "aes-256-cbc"
  authentication_algorithm = "hmac-sha256"
  group                    = 14
  sa_lifetime              = 86400 # seconds
}

# Certificate-authenticated proposal using an AEAD cipher.
resource "netbox_ike_proposal" "aes256_gcm" {
  name                  = "IKE-AES256-GCM-DH19"
  authentication_method = "certificates"
  encryption_algorithm  = "aes-256-gcm"
  group                 = 19
  sa_lifetime           = 28800
}

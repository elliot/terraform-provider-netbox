resource "netbox_ike_proposal" "aes256_sha256" {
  name                     = "IKE-AES256-SHA256-DH14"
  authentication_method    = "preshared-keys"
  encryption_algorithm     = "aes-256-cbc"
  authentication_algorithm = "hmac-sha256"
  group                    = 14
}

# IKEv2 policy: `mode` must be omitted (NetBox rejects it for IKEv2).
resource "netbox_ike_policy" "ikev2" {
  name          = "IKEv2-PSK"
  description   = "IKEv2 with pre-shared keys"
  version       = 2
  proposal_ids  = [netbox_ike_proposal.aes256_sha256.id]
  preshared_key = var.ike_psk
}

# IKEv1 policy: `mode` (main or aggressive) is mandatory.
resource "netbox_ike_policy" "ikev1" {
  name         = "IKEv1-main"
  version      = 1
  mode         = "main"
  proposal_ids = [netbox_ike_proposal.aes256_sha256.id]
}

variable "ike_psk" {
  type      = string
  sensitive = true
}

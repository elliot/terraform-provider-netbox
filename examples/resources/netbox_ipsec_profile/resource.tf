resource "netbox_ike_proposal" "p1" {
  name                     = "IKE-AES256-SHA256-DH14"
  authentication_method    = "preshared-keys"
  encryption_algorithm     = "aes-256-cbc"
  authentication_algorithm = "hmac-sha256"
  group                    = 14
}

resource "netbox_ike_policy" "p1" {
  name         = "IKEv2-PSK"
  version      = 2
  proposal_ids = [netbox_ike_proposal.p1.id]
}

resource "netbox_ipsec_proposal" "p2" {
  name                     = "ESP-AES256-SHA256"
  encryption_algorithm     = "aes-256-cbc"
  authentication_algorithm = "hmac-sha256"
}

resource "netbox_ipsec_policy" "p2" {
  name         = "IPsec-branch"
  proposal_ids = [netbox_ipsec_proposal.p2.id]
  pfs_group    = 14
}

# An IPsec profile ties an IKE policy and an IPsec policy together and is
# referenced by netbox_tunnel.ipsec_profile_id.
resource "netbox_ipsec_profile" "branch" {
  name            = "branch-site-to-site"
  description     = "ESP tunnel-mode profile for branch offices"
  mode            = "esp" # or "ah"
  ike_policy_id   = netbox_ike_policy.p1.id
  ipsec_policy_id = netbox_ipsec_policy.p2.id
}

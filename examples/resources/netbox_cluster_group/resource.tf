# Cluster groups organise clusters, for example by region or environment.
resource "netbox_cluster_group" "emea" {
  name        = "EMEA"
  slug        = "emea"
  description = "Clusters hosted in European data centres"
}

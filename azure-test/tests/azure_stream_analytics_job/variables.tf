variable "resource_name" {
  type        = string
  default     = "steampipe-test"
  description = "Name of the resource used throughout the test."
}

variable "azure_environment" {
  type        = string
  default     = "public"
  description = "Azure environment used for the test."
}

variable "azure_subscription" {
  type        = string
  default     = "cdffd708-7da0-4cea-abeb-0a4c334d7f64"
  description = "Azure subscription used for the test."
}

provider "azurerm" {
  # Cannot be passed as a variable
  version         = "=1.36.0"
  environment     = var.azure_environment
  subscription_id = var.azure_subscription
}

data "azurerm_client_config" "current" {}

data "null_data_source" "resource" {
  inputs = {
    scope = "azure:///subscriptions/${data.azurerm_client_config.current.subscription_id}"
  }
}

resource "azurerm_resource_group" "named_test_resource" {
  name     = var.resource_name
  location = "East US"
}

resource "azurerm_stream_analytics_job" "named_test_resource" {
  name                                     = var.resource_name
  resource_group_name                      = azurerm_resource_group.named_test_resource.name
  location                                 = azurerm_resource_group.named_test_resource.location
  compatibility_level                      = "1.1"
  data_locale                              = "en-GB"
  events_late_arrival_max_delay_in_seconds = 60
  events_out_of_order_max_delay_in_seconds = 50
  events_out_of_order_policy               = "Adjust"
  output_error_policy                      = "Drop"
  streaming_units                          = 3

  transformation_query = <<QUERY
    SELECT *
    INTO [YourOutputAlias]
    FROM [YourInputAlias]
QUERY

}

output "resource_aka" {
  value = "azure://${azurerm_stream_analytics_job.named_test_resource.id}"
}

output "resource_aka_lower" {
  value = "azure://${lower(azurerm_stream_analytics_job.named_test_resource.id)}"
}

output "resource_name" {
  value = var.resource_name
}

output "resource_id" {
  value = azurerm_stream_analytics_job.named_test_resource.id
}

output "location" {
  value = azurerm_resource_group.named_test_resource.location
}

output "subscription_id" {
  value = var.azure_subscription
}

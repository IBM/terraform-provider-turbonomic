
variable "hostname" {
  type        = string
  description = "Turbonomic hostname"
}
variable "username" {
  type        = string
  description = "Turbonomic API username"
}
variable "password" {
  type        = string
  description = "Turbonomic API password"
}
variable "client_id" {
  type        = string
  description = "Turbonomic API client id for oAuth"
}
variable "client_secret" {
  type        = string
  description = "Turbonomic API client secret for oAuth"
}
variable "role" {
  type        = string
  description = "Turbonomic role for oAuth"
}
variable "skipverify" {
  type        = bool
  description = "Whether to validate the hostname certificate"
  default     = false
}

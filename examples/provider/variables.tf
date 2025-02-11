variable "username" {
  type        = string
  description = "Turbonomic API username"
}

variable "password" {
  type        = string
  description = "Turbonomic API password"
}

variable "hostname" {
  type        = string
  description = "Turbonomic hostname"
}

variable "skipverify" {
  type        = bool
  description = "Whether to validate the hostname certificate"
  default     = false
}

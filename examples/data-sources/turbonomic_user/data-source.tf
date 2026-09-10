# Return all administrators
data "turbonomic_user" "admins" {
  role = "ADMINISTRATOR"
}

output "admin_usernames" {
  value = [for u in data.turbonomic_user.admins.users : u.username]
}

# Lookup a single user by username
data "turbonomic_user" "jdoe" {
  username = "jdoe"
}

output "jdoe_uuid" {
  value = data.turbonomic_user.jdoe.users[0].uuid
}

# Return all LDAP users
data "turbonomic_user" "ldap_users" {
  login_provider = "LDAP"
}

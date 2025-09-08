locals {
  throughput_value = (
    tonumber(
      try(
    regex("\\bto\\s+(\\d+)", data.turbonomic_entity_actions.example.actions.0.details)[0], 400))
  )
}

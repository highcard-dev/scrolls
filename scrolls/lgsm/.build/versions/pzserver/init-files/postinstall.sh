#!/bin/bash

# Define the file path
file="lgsm/config-default/config-game/server.ini"

set_config_value() {
  local file="$1"
  local key="$2"
  local value="$3"

  # Escape key and value
  local escaped_key
  local escaped_value
  escaped_key=$(printf '%s\n' "$key" | sed 's/[]\/$*.^[]/\\&/g')
  escaped_value=$(printf '%s\n' "$value" | sed 's/[&/]/\\&/g')

  # Choose sed -i syntax based on OS
  if [[ "$OSTYPE" == "darwin"* ]]; then
    # macOS
    if grep -q "^$escaped_key=" "$file"; then
      sed -i '' "s|^$escaped_key=.*|$escaped_key=$escaped_value|" "$file"
    else
      echo "$key=$value" >> "$file"
    fi
  else
    # Linux
    if grep -q "^$escaped_key=" "$file"; then
      sed -i "s|^$escaped_key=.*|$escaped_key=$escaped_value|" "$file"
    else
      echo "\n" >> "$file"
      echo "$key=$value" >> "$file"
    fi
  fi
}

set_config_value "$file" "DefaultPort" "$DRUID_PORT_MAIN_1"
set_config_value "$file" "UDPPort" "$DRUID_PORT_MAIN2_1"

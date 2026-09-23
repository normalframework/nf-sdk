#!/usr/bin/env bash
# Generates registry-secrets.json for `balena push --registry-secrets`.
#
# The output holds live registry credentials. It is gitignored; keep it that
# way, and delete it when you are done pushing.
set -euo pipefail

out="$(dirname "$0")/registry-secrets.json"

# credentials_for prints "username<TAB>password" for a registry, or nothing.
credentials_for() {
    local registry="$1"

    # The admin user is the simplest source, where it is enabled.
    if [ "$(az acr show -n "$registry" --query adminUserEnabled -o tsv 2>/dev/null)" = "true" ]; then
        az acr credential show -n "$registry" \
            --query "[username, passwords[0].value]" -o tsv 2>/dev/null | paste -s
        return
    fi

    # Otherwise a scoped pull token is needed. Creating one is a change to the
    # registry, so it is left to a human rather than done here:
    #
    #   az acr token create --registry <registry> --name balena-pull \
    #       --scope-map _repositories_pull --output json
    #
    # then use the token name as the username and one of its passwords.
    return 1
}

declare -A creds
missing=()
for registry in normalframework nfdev; do
    host="${registry}.azurecr.io"
    if line=$(credentials_for "$registry"); then
        creds["$host"]="$line"
        echo "$host: got admin credentials" >&2
    else
        missing+=("$host")
        echo "$host: admin user is disabled — a scoped pull token is required" >&2
    fi
done

{
    echo "{"
    first=1
    for host in "${!creds[@]}"; do
        IFS=$'\t' read -r user pass <<< "${creds[$host]}"
        [ $first -eq 1 ] || echo ","
        first=0
        printf '  "%s": {\n    "username": "%s",\n    "password": "%s"\n  }' "$host" "$user" "$pass"
    done
    for host in "${missing[@]}"; do
        [ $first -eq 1 ] || echo ","
        first=0
        printf '  "%s": {\n    "username": "REPLACE_ME",\n    "password": "REPLACE_ME"\n  }' "$host"
    done
    echo
    echo "}"
} > "$out"

chmod 600 "$out"
echo "wrote $out (mode 600)" >&2
if [ ${#missing[@]} -gt 0 ]; then
    echo "fill in by hand: ${missing[*]}" >&2
fi

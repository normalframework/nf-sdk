# dev0 deployment

NF 3.10.8 plus the network configuration console, for the `normal-dev` fleet.

```sh
balena push normal-dev --source .
```

Both registries are private, so the builder needs credentials:

```sh
./make-registry-secrets.sh          # writes registry-secrets.json, mode 600
balena push normal-dev --source . --registry-secrets registry-secrets.json
```

`registry-secrets.json` holds live credentials and is gitignored. Delete it when
you are done.

The script fills in what it can. `normalframework` has its admin user enabled, so
that one is automatic. **`nfdev` does not**, so it needs a scoped pull token,
created once:

```sh
az acr token create --registry nfdev --name balena-pull \
    --scope-map _repositories_pull --output json
```

Use the token name as the username and one of the generated passwords, and put
them in `registry-secrets.json`.

Registry secrets can also be stored per-fleet in balenaCloud under
**Fleet → Settings**, which avoids passing the file on every push.

## After pushing

The console is at `http://<device-ip>:8081`, signing in with `admin` / `normal`.
It forces a password change before anything else is reachable.

Ports on the host network: nf takes 80, the console takes 8081.

## Note before you push

`balena push` applies to the whole fleet, not one device. If `normal-dev` has
devices other than dev0, either move dev0 to its own fleet first or pin the
others to their current release.

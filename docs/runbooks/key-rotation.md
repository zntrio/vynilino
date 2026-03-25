# PASETO Key Rotation Runbook

**Threat:** T-009 — PASETO key loss or uncontrolled rotation
**DREAD total:** 28 (Tier 2)

## Overview

Vynilino uses a single PASETO v4 symmetric key (`VYNILINO_TOKEN_KEY`) to encrypt and decrypt
access tokens (15-minute TTL). Replacing the key naively invalidates all active sessions
immediately. This runbook describes a **zero-disruption rotation** using a 15-minute bridge
period during which both the old and new keys are accepted.

---

## Prerequisites

- Shell access to the host running vynilino
- Ability to update environment variables and restart the process (or update Docker secrets /
  HashiCorp Vault)
- The new key must **not** be committed to git

---

## Step-by-step procedure

### Step 1 — Generate the new key

```bash
NEW_KEY=$(openssl rand -hex 32)
echo "New key: $NEW_KEY"
```

Store `$NEW_KEY` in your secrets manager (see [Key storage](#key-storage) below).

### Step 2 — Enter the bridge period

Set **both** keys and restart. The old key becomes the _primary_ encryption key; the new key
acts as the _secondary_ (fallback) decryption key so existing tokens remain valid.

```bash
# Current state: VYNILINO_TOKEN_KEY=<old_key>
# Add:
export VYNILINO_TOKEN_KEY_NEW=<new_key>
# Restart vynilino
```

During this phase:
- New access tokens are minted with `TOKEN_KEY` (old key — will be rotated away next)
- Existing tokens encrypted with the old key continue to validate

> Wait at least **15 minutes** — one full access-token TTL — before proceeding.

### Step 3 — Promote the new key

After the bridge period, swap the keys and remove the secondary:

```bash
export VYNILINO_TOKEN_KEY=<new_key>
unset VYNILINO_TOKEN_KEY_NEW
# Restart vynilino
```

From this point:
- New tokens are minted with the new key
- Old tokens (encrypted with the old key) can no longer be decrypted — users with tokens
  older than 15 minutes will be asked to log in again (expected behaviour)

### Step 4 — Revoke all refresh tokens (optional, for emergency rotation)

If the old key was **compromised**, immediately force all users to re-authenticate by revoking
all refresh tokens. This can be done directly in SQLite:

```bash
sqlite3 ./data/vynilino.db \
  "UPDATE refresh_tokens SET revoked = 1 WHERE revoked = 0;"
```

All active sessions will expire within 15 minutes (next access-token validation).

---

## Key storage

Keys must **never** be committed to git. Acceptable storage options:

| Method | How to apply |
|--------|-------------|
| Environment variable in a restricted `.env` file | `chmod 600 .env` |
| Docker secrets | `docker secret create vynilino_token_key <(echo -n "$NEW_KEY")` |
| HashiCorp Vault | `vault kv put secret/vynilino token_key="$NEW_KEY"` |
| systemd `EnvironmentFile` | `chmod 600 /etc/vynilino/env`; reference with `EnvironmentFile=` |

---

## Rollback

To abort the rotation and revert to the old key during the bridge period:

```bash
unset VYNILINO_TOKEN_KEY_NEW
# Restart vynilino (TOKEN_KEY unchanged)
```

---

## Validation (EXP-009)

Run the following checks in staging before executing in production:

1. **Bridge period**: Login, obtain token T1. Set `VYNILINO_TOKEN_KEY_NEW` and restart.
   Verify T1 still validates within 15 minutes.
2. **New token issuance**: Login again after restart, obtain T2. Verify T2 validates.
3. **Cutover**: Promote new key (Step 3). Verify T2 still validates; verify T1 is rejected.
4. **Forced revocation** (if applicable): Run the SQL revocation above and verify all
   refresh tokens are invalidated within one TTL cycle.

See `docs/validation/T-009-paseto-key-rotation-validation.md` for the full test matrix.

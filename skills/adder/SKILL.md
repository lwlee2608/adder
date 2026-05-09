---
name: adder
description: Use when adding or editing config structs/YAML for projects that use lwlee2608/adder. Prevents silent zero-value binding from snake_case keys, env var mismatches, and missing defaults.
user-invocable: true
---

# Working with lwlee2608/adder

Adder reads YAML into Go structs and overlays env vars. It does **case-insensitive** key matching only — it does **not** fold snake_case ↔ CamelCase. Every rule below exists to avoid fields silently binding to their zero value.

## Rules

1. **Prefer YAML keys that already match the lowercased Go field name.** Adder looks up `strings.ToLower(field.Name)`, so a field `SessionTtl` is resolved against the YAML key `sessionttl`. Both lowercase and camelCase keys work because adder lowercases both sides — camelCase is just easier to read.

   ```yaml
   # good — matches field name when lowercased
   auth:
     username: admin
     sessionTtl: 24h
   ```

   ```go
   type AuthConfig struct {
       Username   string
       SessionTtl time.Duration  // no tag required
   }
   ```

   **Idiomatic Go vs. tag-free binding is a tension.** Go style says initialisms should be all-caps (`SessionTTL`). But `strings.ToLower("SessionTTL")` is `"sessionttl"`, which forces either an ugly YAML key (`sessionttl: 24h`) or a `mapstructure` tag. Pick your poison: idiomatic Go + tag, or non-idiomatic field name + no tag. This skill leans toward the latter to minimize tags, but either is defensible — be consistent within a project.

   This applies recursively at every nesting level — `auth.session.ttl` resolves each segment the same way.

2. **Avoid snake_case in YAML.** A snake_case key like `session_ttl` will **never** bind to `SessionTTL` automatically — adder compares `sessionttl` to `session_ttl` and finds no match, so the field silently stays at its zero value (e.g. `time.Duration(0)`, which means "expires immediately" for TTLs). If you must use snake_case for readability, every such field requires an explicit `mapstructure` tag:

   ```go
   // only if you insist on snake_case YAML
   SessionTTL time.Duration `mapstructure:"session_ttl"`
   ```

   Prefer rule 1 over this — fewer tags, fewer ways to forget one.

3. **Put defaults in `application.yml`, not in code.** `application.yml` is the canonical source of default values. Code should not paper over missing config with hard-coded fallbacks (e.g. `if cfg.SessionTTL == 0 { cfg.SessionTTL = 24*time.Hour }`) — that hides genuine misconfiguration. If a value is required, validate at startup and panic with a clear message.

   ```yaml
   # application.yml — defaults live here
   auth:
     sessionTtl: 24h
   ```

4. **Rely on `AutomaticEnv()` for the standard env var pattern.** A *keyPath* is the dotted path adder uses internally — struct field `Auth.SessionTtl` becomes keyPath `auth.sessionttl`. With `SetEnvKeyReplacer(strings.NewReplacer(".", "_"))` and `AutomaticEnv()`, adder derives the env var as `strings.ToUpper(keyPath)` with dots replaced by underscores. Don't add `BindEnv` calls for env vars that already follow this pattern — they're noise.

   ```
   keyPath: openrouter.apikey  →  env: OPENROUTER_APIKEY   (auto, no BindEnv needed)
   keyPath: db.url             →  env: DB_URL              (auto)
   ```

5. **Use `BindEnv` only when the env var name diverges from the auto pattern.** Common reason: an established external env name (e.g. `OPENROUTER_API_KEY` with an underscore between `API` and `KEY`) doesn't match the auto-derived `OPENROUTER_APIKEY`.

   ```go
   // Only because the external name is OPENROUTER_API_KEY, not OPENROUTER_APIKEY
   _ = adder.BindEnv("openrouter.apikey", "OPENROUTER_API_KEY")

   // Don't write this — DB_URL is already automatic from keyPath db.url
   // _ = adder.BindEnv("db.url", "DB_URL")
   ```

   If you can rename the env var to match the auto pattern, do that instead and drop the `BindEnv`.

## Verification procedure

After adding or changing a config field (these checks apply at every nesting level — verify the deepest field, not just the top-level struct):

1. **Lowercase-match check** — does `strings.ToLower(fieldName)` equal the YAML key exactly? If not, you need a `mapstructure` tag, or you should rename the YAML key.
2. **Default check** — is the default value in `application.yml`? Run the binary with no env overrides and confirm the field has the expected value. Don't rely on memory — log it, e.g. `log.Printf("%+v", cfg)`.
3. **Env override check** — if the field is meant to be env-overridable, set the auto-derived env var (`UPPER_CASE_WITH_UNDERSCORES`) and confirm it overrides the YAML default. Same rule: log the loaded value, don't assume.
4. **Zero-value trap** — for `time.Duration`, `int`, or `bool` fields, a binding miss looks identical to "configured as 0/false". Always test a non-zero default actually loads.

## Common mistakes to watch for

- **Adding a multi-word field without a `mapstructure` tag and using snake_case YAML.** This is the #1 footgun: the field silently binds to zero. `SessionTTL` + `session_ttl:` → `0`, not `24h`.
- **Adding `BindEnv` for env vars that already follow the auto pattern.** Redundant and obscures which bindings are actually needed.
- **Hiding misconfiguration with code defaults.** A `if cfg.X == 0 { cfg.X = default }` masks the real bug (the YAML key didn't bind). Fix the binding instead.
- **Assuming case-insensitive means snake-aware.** It doesn't — `caseInsensitiveLookup` only normalizes letter case, not separators.
- **Forgetting that `time.Duration` is `int64`.** A missing duration looks like `0s`, which most code treats as "no timeout" or "expired" — almost always wrong.

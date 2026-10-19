-- totp_secret holds a secret as soon as setup starts, confirmed or not;
-- totp_enabled_at is the real "is 2FA on" flag - it only gets set once
-- the user has proven, by submitting a valid code, that the secret
-- actually made it into their authenticator app. That ordering is what
-- makes it impossible to lock yourself out by enabling 2FA: login never
-- starts depending on it until a real code has already been verified.
ALTER TABLE users ADD COLUMN totp_secret TEXT;
ALTER TABLE users ADD COLUMN totp_enabled_at TIMESTAMPTZ;

-- Single-use backup codes for a lost authenticator device. Hashed like
-- passwords, never stored or shown again after the one time they're
-- generated at confirmation time.
CREATE TABLE totp_recovery_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code_hash TEXT NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_totp_recovery_codes_user ON totp_recovery_codes(user_id);

-- A password-verified-but-not-yet-fully-authenticated login, while the
-- 6-digit step is pending - deliberately its own short-lived row rather
-- than a half-authenticated session, so there's no path where a
-- password alone ever sets a usable session cookie for a 2FA account.
CREATE TABLE pending_totp_logins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    failed_attempts INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ip_address TEXT,
    user_agent TEXT
);

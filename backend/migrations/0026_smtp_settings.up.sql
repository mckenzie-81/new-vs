-- smtp_password_encrypted holds ciphertext (AES-256-GCM, see
-- internal/smtp) - never plaintext, since unlike everything else this
-- app stores, an SMTP credential is a genuine two-way secret that has to
-- be read back to actually send mail, not a one-way hash target.
ALTER TABLE system_settings ADD COLUMN smtp_host TEXT;
ALTER TABLE system_settings ADD COLUMN smtp_port INT;
ALTER TABLE system_settings ADD COLUMN smtp_username TEXT;
ALTER TABLE system_settings ADD COLUMN smtp_password_encrypted TEXT;
ALTER TABLE system_settings ADD COLUMN smtp_from_name TEXT;
ALTER TABLE system_settings ADD COLUMN smtp_from_email TEXT;
-- Defaults to true (safe) so configuring SMTP is never, by itself, the
-- moment real sending starts - see internal/smtp's package doc.
ALTER TABLE system_settings ADD COLUMN smtp_test_mode BOOLEAN NOT NULL DEFAULT true;

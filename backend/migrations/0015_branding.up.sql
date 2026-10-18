CREATE TABLE system_settings (
    id BOOLEAN PRIMARY KEY DEFAULT true,
    accent_color TEXT,
    logo_url TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT system_settings_singleton CHECK (id)
);

INSERT INTO system_settings (id) VALUES (true);

ALTER TABLE elections ADD COLUMN accent_color TEXT;
ALTER TABLE elections ADD COLUMN logo_url TEXT;

-- A second and third accent shade, alongside the existing accent_color -
-- still accent-only (neutrals/success/warning/danger stay fixed), just a
-- richer brand palette instead of one flat color everywhere.
ALTER TABLE system_settings ADD COLUMN accent_color_secondary TEXT;
ALTER TABLE system_settings ADD COLUMN accent_color_tertiary TEXT;

ALTER TABLE elections ADD COLUMN accent_color_secondary TEXT;
ALTER TABLE elections ADD COLUMN accent_color_tertiary TEXT;

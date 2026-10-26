-- The name shown in the dashboard sidebar - "Voting System" today is
-- just the fallback an admin hasn't overridden yet, same empty-means-
-- default convention as every other branding field.
ALTER TABLE system_settings ADD COLUMN platform_name TEXT;

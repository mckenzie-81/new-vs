-- Postgres has no "DROP VALUE" for enums. This rebuilds the type without
-- ADMIN/APPROVER, which only succeeds if no row currently uses them -
-- standard limitation of reversing an enum addition, not special to this
-- migration.
ALTER TYPE user_role RENAME TO user_role_old;

CREATE TYPE user_role AS ENUM ('SYSTEMADMIN');

ALTER TABLE users
    ALTER COLUMN role TYPE user_role USING role::text::user_role;

DROP TYPE user_role_old;

CREATE TABLE IF NOT EXISTS roles (
    id bigserial PRIMARY KEY,
    role_name text NOT NULL,
    user_id bigint NOT NULL REFERENCES users ON DELETE CASCADE
);
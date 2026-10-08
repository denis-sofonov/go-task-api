-- +goose Up
CREATE TABLE users (
    id                uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    email             text        NOT NULL,
    name              text        NOT NULL,
    password_hash     text        NOT NULL,
    email_verified_at timestamptz,
    created_at        timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT users_email_key UNIQUE (email),
    CONSTRAINT users_email_lowercase CHECK (email = lower(email))
);

-- +goose Down
DROP TABLE users;

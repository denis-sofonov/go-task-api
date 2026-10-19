-- +goose Up
CREATE TABLE tasks (
    id          uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    title       text        NOT NULL,
    description text        NOT NULL DEFAULT '',
    done        boolean     NOT NULL DEFAULT false,
    created_at  timestamptz NOT NULL DEFAULT now(),
    updated_at  timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT tasks_title_not_empty CHECK (title <> '')
);

CREATE INDEX tasks_user_id_idx ON tasks (user_id);

-- +goose Down
DROP TABLE tasks;

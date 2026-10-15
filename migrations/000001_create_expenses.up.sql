CREATE TABLE expenses (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL,
    amount NUMERIC(12, 2) NOT NULL CHECK (amount > 0),
    category VARCHAR(32) NOT NULL CHECK (
        category IN ('food', 'transport', 'housing', 'entertainment', 'health', 'shopping', 'other')
    ),
    spent_at DATE NOT NULL,
    comment TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX expenses_user_spent_at_idx ON expenses (user_id, spent_at DESC, id DESC);
CREATE INDEX expenses_category_idx ON expenses (category);

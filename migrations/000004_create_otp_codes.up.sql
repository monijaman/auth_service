CREATE TABLE IF NOT EXISTS otp_codes (
    id         UUID        PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    code       VARCHAR(10) NOT NULL,
    type       VARCHAR(30) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    UNIQUE (user_id, type)
);

CREATE INDEX idx_otp_codes_user_id    ON otp_codes (user_id);
CREATE INDEX idx_otp_codes_expires_at ON otp_codes (expires_at);

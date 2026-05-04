-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at          TIMESTAMPTZ,
    email               VARCHAR(255) NOT NULL,
    password_hash       VARCHAR(255) NOT NULL,
    identity_statement  TEXT NOT NULL DEFAULT '',
    identity_score      DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    timezone            VARCHAR(100) NOT NULL DEFAULT 'UTC',
    start_of_day_offset INT NOT NULL DEFAULT 0,
    CONSTRAINT users_email_unique UNIQUE (email),
    CONSTRAINT users_identity_score_range CHECK (identity_score >= 0.0 AND identity_score <= 100.0)
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_deleted_at ON users(deleted_at);

-- Habits table
CREATE TABLE IF NOT EXISTS habits (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    standard_title  VARCHAR(255) NOT NULL,
    emergency_title VARCHAR(255) NOT NULL DEFAULT '',
    status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    CONSTRAINT habits_status_check CHECK (status IN ('ACTIVE', 'PAUSED', 'ARCHIVED'))
);

CREATE INDEX IF NOT EXISTS idx_habits_user_id ON habits(user_id);
CREATE INDEX IF NOT EXISTS idx_habits_deleted_at ON habits(deleted_at);

-- Habit logs table
CREATE TABLE IF NOT EXISTS habit_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    habit_id        UUID NOT NULL REFERENCES habits(id) ON DELETE CASCADE,
    log_date        DATE NOT NULL,
    completion_type VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    is_synced       BOOLEAN NOT NULL DEFAULT FALSE,
    CONSTRAINT habit_logs_completion_type_check CHECK (completion_type IN ('PENDING', 'COMPLETED_STANDARD', 'COMPLETED_EMERGENCY', 'MISSED')),
    CONSTRAINT habit_logs_unique_habit_date UNIQUE (habit_id, log_date)
);

CREATE INDEX IF NOT EXISTS idx_habit_logs_habit_id ON habit_logs(habit_id);
CREATE INDEX IF NOT EXISTS idx_habit_logs_log_date ON habit_logs(log_date DESC);
CREATE INDEX IF NOT EXISTS idx_habit_logs_deleted_at ON habit_logs(deleted_at);

-- Таблица credits — банковские кредиты.
CREATE TABLE IF NOT EXISTS credits (
    id SERIAL PRIMARY KEY,
    account_id INT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    rate NUMERIC(5,2) NOT NULL,
    term_months INT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Таблица payment_schedules — график платежей по кредиту.
CREATE TABLE IF NOT EXISTS payment_schedules (
    id SERIAL PRIMARY KEY,
    credit_id INT NOT NULL REFERENCES credits(id) ON DELETE CASCADE,
    payment_date TIMESTAMPTZ NOT NULL,
    principal BIGINT NOT NULL,
    interest BIGINT NOT NULL,
    total BIGINT NOT NULL,
    remaining_balance BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX idx_credits_account_id ON credits (account_id);
CREATE INDEX idx_payment_schedules_credit_id ON payment_schedules (credit_id);
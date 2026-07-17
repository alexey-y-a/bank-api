CREATE TABLE IF NOT EXISTS cards (
    id SERIAL PRIMARY KEY,
    account_id INT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE
    number_hash TEXT NOT NULL,
    number_enc BYETA NOT NULL,
    cvc_hash TEXT NOT NULL,
    expiry_enc BYETA NOT NULL,
    status  VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW
)

CREATE INDEX idx_cards_account_id ON cards (account_id);
CREATE INDEX idx_cards_number_hash ON cards (number_hash);
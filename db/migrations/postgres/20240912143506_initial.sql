-- +goose Up
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users
CREATE TABLE dbank_users (
    pk           SERIAL        PRIMARY KEY,
    id           UUID          NOT NULL UNIQUE,
    username     TEXT          NOT NULL UNIQUE,
    email        TEXT          NOT NULL UNIQUE,
    password     TEXT          NOT NULL,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);
CREATE INDEX idx_dbank_users_id       ON dbank_users(id);
CREATE INDEX idx_dbank_users_username ON dbank_users(username);
CREATE INDEX idx_dbank_users_email    ON dbank_users(email);

-- Roles
CREATE TABLE dbank_roles (
    pk           SERIAL        PRIMARY KEY,
    id           UUID          NOT NULL UNIQUE,
    name         TEXT          NOT NULL UNIQUE,
    description  TEXT,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);
CREATE INDEX idx_dbank_roles_id   ON dbank_roles(id);
CREATE INDEX idx_dbank_roles_name ON dbank_roles(name);

-- Permissions
CREATE TABLE dbank_permissions (
    pk           SERIAL        PRIMARY KEY,
    id           UUID          NOT NULL UNIQUE,
    name         TEXT          NOT NULL UNIQUE,
    description  TEXT,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at   TIMESTAMPTZ
);
CREATE INDEX idx_dbank_perms_id   ON dbank_permissions(id);
CREATE INDEX idx_dbank_perms_name ON dbank_permissions(name);

-- Role ↔ Permission link (links by pk)
CREATE TABLE dbank_role_permissions (
    pk             SERIAL        PRIMARY KEY,
    role_pk        INT           NOT NULL,
    perm_pk        INT           NOT NULL,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    FOREIGN KEY (role_pk) REFERENCES dbank_roles(pk) ON DELETE CASCADE,
    FOREIGN KEY (perm_pk) REFERENCES dbank_permissions(pk) ON DELETE CASCADE
);
CREATE UNIQUE INDEX idx_dbank_role_perms_unique ON dbank_role_permissions(role_pk, perm_pk);

-- User ↔ Role assignment (links by pk)
CREATE TABLE dbank_user_roles (
    pk         SERIAL        PRIMARY KEY,
    user_pk    INT           NOT NULL,
    role_pk    INT           NOT NULL,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    FOREIGN KEY (user_pk) REFERENCES dbank_users(pk) ON DELETE CASCADE,
    FOREIGN KEY (role_pk) REFERENCES dbank_roles(pk) ON DELETE CASCADE
);
CREATE UNIQUE INDEX idx_dbank_user_roles_unique ON dbank_user_roles(user_pk, role_pk);

-- Accounts (linked by user pk)
CREATE TABLE dbank_accounts (
    pk             SERIAL        PRIMARY KEY,
    id             UUID          NOT NULL UNIQUE,
    user_pk        INT           NOT NULL,
    account_type   TEXT          NOT NULL,
    account_number TEXT          NOT NULL UNIQUE,
    balance        DECIMAL(20,6) NOT NULL DEFAULT 0.00,
    currency       TEXT          NOT NULL,
    status         TEXT          NOT NULL,
    account_name   TEXT          NOT NULL,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    FOREIGN KEY (user_pk) REFERENCES dbank_users(pk) ON DELETE NO ACTION
);
CREATE INDEX idx_dbank_accounts_id             ON dbank_accounts(id);
CREATE INDEX idx_dbank_accounts_user_pk        ON dbank_accounts(user_pk);
CREATE INDEX idx_dbank_accounts_account_number ON dbank_accounts(account_number);

-- Transactions (linked by account pk)
CREATE TABLE dbank_transactions (
    pk               SERIAL        PRIMARY KEY,
    id               UUID          NOT NULL UNIQUE,
    account_pk       INT           NOT NULL,
    transaction_type TEXT          NOT NULL,
    amount           DECIMAL(20,6) NOT NULL,
    currency         TEXT          NOT NULL,
    transaction_date TIMESTAMPTZ   NOT NULL DEFAULT now(),
    description      TEXT,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at       TIMESTAMPTZ,
    FOREIGN KEY (account_pk) REFERENCES dbank_accounts(pk) ON DELETE NO ACTION
);
CREATE INDEX idx_dbank_tx_id          ON dbank_transactions(id);
CREATE INDEX idx_dbank_tx_account_pk  ON dbank_transactions(account_pk);

-- Notifications (linked by user pk)
CREATE TABLE dbank_notifications (
    pk                SERIAL        PRIMARY KEY,
    id                UUID          NOT NULL UNIQUE,
    user_pk           INT           NOT NULL,
    notification_type TEXT          NOT NULL,
    message           TEXT          NOT NULL,
    is_read           BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    FOREIGN KEY (user_pk) REFERENCES dbank_users(pk) ON DELETE NO ACTION
);
CREATE INDEX idx_dbank_notifs_id      ON dbank_notifications(id);
CREATE INDEX idx_dbank_notifs_user_pk ON dbank_notifications(user_pk);

-- Ledgers (linked by account pk & tx pk)
CREATE TABLE dbank_ledgers (
    pk             SERIAL        PRIMARY KEY,
    id             UUID          NOT NULL UNIQUE,
    account_pk     INT           NOT NULL,
    transaction_pk INT           NOT NULL,
    balance        DECIMAL(20,6) NOT NULL,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at     TIMESTAMPTZ,
    FOREIGN KEY (account_pk)     REFERENCES dbank_accounts(pk)     ON DELETE NO ACTION,
    FOREIGN KEY (transaction_pk) REFERENCES dbank_transactions(pk) ON DELETE NO ACTION
);
CREATE INDEX idx_dbank_ledgers_id             ON dbank_ledgers(id);
CREATE INDEX idx_dbank_ledgers_account_pk     ON dbank_ledgers(account_pk);
CREATE INDEX idx_dbank_ledgers_transaction_pk ON dbank_ledgers(transaction_pk);

-- Audit Logs (linked by user pk)
CREATE TABLE dbank_audit_logs (
    pk         SERIAL        PRIMARY KEY,
    id         UUID          NOT NULL UNIQUE,
    user_pk    INT           NOT NULL,
    action     TEXT          NOT NULL,
    data       JSONB         NOT NULL,
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ   NOT NULL DEFAULT now(),
    deleted_at TIMESTAMPTZ,
    FOREIGN KEY (user_pk) REFERENCES dbank_users(pk) ON DELETE NO ACTION
);
CREATE INDEX idx_dbank_audit_id      ON dbank_audit_logs(id);
CREATE INDEX idx_dbank_audit_user_pk ON dbank_audit_logs(user_pk);

-- Users
INSERT INTO dbank_users (id, username, email, password) VALUES
  ('11111111-1111-1111-1111-111111111111','alice','alice@example.com','$2a$12$1vnOn9v5so66ZpKqAj4YtuQzyvXpN.X6Eso5N6lHbNTWpg9j8aG5u'),
  ('22222222-2222-2222-2222-222222222222','bob','bob@example.com','$2a$12$1vnOn9v5so66ZpKqAj4YtuQzyvXpN.X6Eso5N6lHbNTWpg9j8aG5u');


-- +goose Down
-- drop all indexes
DROP INDEX IF EXISTS idx_dbank_audit_user_pk;
DROP INDEX IF EXISTS idx_dbank_audit_id;
DROP INDEX IF EXISTS idx_dbank_ledgers_transaction_pk;
DROP INDEX IF EXISTS idx_dbank_ledgers_account_pk;
DROP INDEX IF EXISTS idx_dbank_ledgers_id;
DROP INDEX IF EXISTS idx_dbank_notifs_user_pk;
DROP INDEX IF EXISTS idx_dbank_notifs_id;
DROP INDEX IF EXISTS idx_dbank_tx_account_pk;
DROP INDEX IF EXISTS idx_dbank_tx_id;
DROP INDEX IF EXISTS idx_dbank_accounts_account_number;
DROP INDEX IF EXISTS idx_dbank_accounts_user_pk;
DROP INDEX IF EXISTS idx_dbank_accounts_id;
DROP INDEX IF EXISTS idx_dbank_user_roles_unique;
DROP INDEX IF EXISTS idx_dbank_role_perms_unique;
DROP INDEX IF EXISTS idx_dbank_perms_name;
DROP INDEX IF EXISTS idx_dbank_perms_id;
DROP INDEX IF EXISTS idx_dbank_roles_name;
DROP INDEX IF EXISTS idx_dbank_roles_id;
DROP INDEX IF EXISTS idx_dbank_users_email;
DROP INDEX IF EXISTS idx_dbank_users_username;
DROP INDEX IF EXISTS idx_dbank_users_id;

-- drop tables
DROP TABLE IF EXISTS dbank_audit_logs;
DROP TABLE IF EXISTS dbank_ledgers;
DROP TABLE IF EXISTS dbank_notifications;
DROP TABLE IF EXISTS dbank_transactions;
DROP TABLE IF EXISTS dbank_accounts;
DROP TABLE IF EXISTS dbank_user_roles;
DROP TABLE IF EXISTS dbank_role_permissions;
DROP TABLE IF EXISTS dbank_permissions;
DROP TABLE IF EXISTS dbank_roles;
DROP TABLE IF EXISTS dbank_users;

DROP EXTENSION IF EXISTS "pgcrypto";

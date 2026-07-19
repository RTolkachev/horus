-- SYNTHETIC TEST FIXTURES — nothing here is real data or production
-- schema. These scripts exist only for the local docker-compose
-- sandbox and integration tests; tests must not depend on the default
-- tables or their row counts (create your own tables and call the
-- seed procedures with a known count instead).
--
-- Files in /docker-entrypoint-initdb.d run alphabetically, once, when the
-- mysql container starts with an EMPTY data volume. `make db-up` on an
-- existing volume skips them entirely — `make db-reset` is the way to
-- re-run schema + seed from scratch.
--
-- These run as root, so we can create a second database and grant it to
-- the app user (compose only creates horus_test itself).

CREATE DATABASE IF NOT EXISTS billing;
GRANT ALL PRIVILEGES ON billing.* TO 'horus'@'%';

-- Tables mirror the fixtures in internal/config tests: horus_test has
-- events + audit_log, billing has invoices. All are id-strategy
-- candidates: BIGINT auto-increment PK, a timestamp the rows age by,
-- and no foreign keys (partitioned InnoDB tables cannot have any).

CREATE TABLE horus_test.events (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    user_id    INT UNSIGNED    NOT NULL,
    kind       VARCHAR(32)     NOT NULL,
    payload    VARCHAR(255)    NOT NULL,
    created_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_events_created_at (created_at)
);

CREATE TABLE horus_test.audit_log (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    actor_id   INT UNSIGNED    NOT NULL,
    action     VARCHAR(32)     NOT NULL,
    entity     VARCHAR(64)     NOT NULL,
    created_at DATETIME        NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (id),
    KEY idx_audit_created_at (created_at)
);

CREATE TABLE billing.invoices (
    id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
    account_id INT UNSIGNED    NOT NULL,
    amount     DECIMAL(10, 2)  NOT NULL,
    status     VARCHAR(16)     NOT NULL,
    issued_at  DATETIME        NOT NULL,
    PRIMARY KEY (id),
    KEY idx_invoices_issued_at (issued_at)
);

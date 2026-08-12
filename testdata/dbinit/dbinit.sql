-- SYNTHETIC TEST FIXTURES - nothing here is real data or production
-- schema. This script exists only for the local docker-compose
-- sandbox and integration tests; tests must not depend on the default
-- tables or their row counts (create your own tables and call the
-- seed procedures with a known count instead).
--
-- Files in /docker-entrypoint-initdb.d run alphabetically, once, when the
-- mysql container starts with an EMPTY data volume. `make db-up` on an
-- existing volume skips them entirely - `make db-reset` is the way to
-- re-run everything from scratch.
--
-- Runs as root, so it can create the extra databases and grant them
-- to the app user (compose only creates horus_test itself).

CREATE DATABASE IF NOT EXISTS billing;
GRANT ALL PRIVILEGES ON billing.* TO 'horus'@'%';

-- horus's own meta database, provisioned out-of-band exactly as a DBA
-- would: `horus init` assumes it exists and only creates tables inside
-- it (least privilege - no CREATE DATABASE grant needed by the app
-- user). Full rights here are legitimate: horus owns this schema. The
-- client databases above stay the tool's read/structural-alter targets.
CREATE DATABASE IF NOT EXISTS horus;
GRANT ALL PRIVILEGES ON horus.* TO 'horus'@'%';

-- privileges horus's analyze requires beyond plain select, granted to
-- the sandbox app user so dev mirrors a correctly-provisioned prod user:
--   process - information_schema.innodb_tables/innodb_tablespaces
--             (tablespace placement); global-only, hence *.*
--   trigger - makes information_schema.triggers rows visible; without it
--             the catalog silently returns no rows and "has triggers"
--             would report a false all-clear
GRANT PROCESS ON *.* TO 'horus'@'%';
GRANT TRIGGER ON horus_test.* TO 'horus'@'%';

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

-- Seed data, generated rather than hardcoded. Each seeder is a stored
-- procedure taking a row count, so the same code seeds 10k rows or 10M -
-- re-callable any time from `make db-shell`:
--
--     CALL horus_test.seed_events(1000000);
--
-- The row generator is a recursive CTE (MySQL 8.0+): seq produces
-- 1..n, and one INSERT ... SELECT materializes all rows in a single
-- statement - orders of magnitude faster than a WHILE loop of
-- single-row INSERTs.
--
-- Timestamps are spread over the ~365 days before NOW(), oldest row
-- first, so auto-increment ids correlate with created_at the way they
-- would in a real append-only table. That correlation is what makes
-- id-range partitions line up with time, so the seeder preserves it
-- deliberately.

DELIMITER $$

CREATE PROCEDURE horus_test.seed_events(IN n INT)
BEGIN
    SET SESSION cte_max_recursion_depth = 100000000;
    INSERT INTO horus_test.events (user_id, kind, payload, created_at)
    WITH RECURSIVE seq (i) AS (
        SELECT 1 UNION ALL SELECT i + 1 FROM seq WHERE i < n
    )
    SELECT
        1 + FLOOR(RAND() * 10000),
        ELT(1 + FLOOR(RAND() * 4), 'click', 'view', 'purchase', 'login'),
        CONCAT('payload-', i),
        NOW() - INTERVAL FLOOR((n - i) * 31536000 / n) SECOND
    FROM seq;
END$$

CREATE PROCEDURE horus_test.seed_audit_log(IN n INT)
BEGIN
    SET SESSION cte_max_recursion_depth = 100000000;
    INSERT INTO horus_test.audit_log (actor_id, action, entity, created_at)
    WITH RECURSIVE seq (i) AS (
        SELECT 1 UNION ALL SELECT i + 1 FROM seq WHERE i < n
    )
    SELECT
        1 + FLOOR(RAND() * 500),
        ELT(1 + FLOOR(RAND() * 4), 'create', 'update', 'delete', 'login'),
        CONCAT(ELT(1 + FLOOR(RAND() * 3), 'user/', 'invoice/', 'event/'),
               1 + FLOOR(RAND() * 100000)),
        NOW() - INTERVAL FLOOR((n - i) * 31536000 / n) SECOND
    FROM seq;
END$$

CREATE PROCEDURE billing.seed_invoices(IN n INT)
BEGIN
    SET SESSION cte_max_recursion_depth = 100000000;
    INSERT INTO billing.invoices (account_id, amount, status, issued_at)
    WITH RECURSIVE seq (i) AS (
        SELECT 1 UNION ALL SELECT i + 1 FROM seq WHERE i < n
    )
    SELECT
        1 + FLOOR(RAND() * 2000),
        ROUND(5 + RAND() * 995, 2),
        ELT(1 + FLOOR(RAND() * 3), 'paid', 'pending', 'void'),
        NOW() - INTERVAL FLOOR((n - i) * 31536000 / n) SECOND
    FROM seq;
END$$

DELIMITER ;

-- Initial volume is deliberately small: partition boundaries are
-- computed from id values, so 1000 rows with step=100 exercises the
-- same logic as millions - and `make db-reset` stays instant. When an
-- experiment needs real volume, scale on demand from `make db-shell`:
--
--     CALL horus_test.seed_events(500000);
CALL horus_test.seed_events(1000);
CALL horus_test.seed_audit_log(1000);
CALL billing.seed_invoices(1000);

-- Seed data, generated rather than hardcoded. Each seeder is a stored
-- procedure taking a row count, so the same code seeds 10k rows or 10M —
-- re-callable any time from `make db-shell`:
--
--     CALL horus_test.seed_events(1000000);
--
-- The row generator is a recursive CTE (MySQL 8.0+): seq produces
-- 1..n, and one INSERT ... SELECT materializes all rows in a single
-- statement — orders of magnitude faster than a WHILE loop of
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
-- same logic as millions — and `make db-reset` stays instant. When an
-- experiment needs real volume, scale on demand from `make db-shell`:
--
--     CALL horus_test.seed_events(500000);
CALL horus_test.seed_events(1000);
CALL horus_test.seed_audit_log(1000);
CALL billing.seed_invoices(1000);

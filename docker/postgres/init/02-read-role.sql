-- Local analogue of the credential used with CloudNativePG's read service.
-- Privileges are the hard boundary; default_transaction_read_only is defense
-- in depth for accidental writes issued through an otherwise read-only path.
CREATE ROLE evekill_read LOGIN PASSWORD 'evekill_read';
GRANT CONNECT ON DATABASE evekill TO evekill_read;
GRANT USAGE ON SCHEMA public TO evekill_read;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO evekill_read;
GRANT SELECT ON ALL SEQUENCES IN SCHEMA public TO evekill_read;
ALTER DEFAULT PRIVILEGES FOR ROLE evekill IN SCHEMA public
    GRANT SELECT ON TABLES TO evekill_read;
ALTER DEFAULT PRIVILEGES FOR ROLE evekill IN SCHEMA public
    GRANT SELECT ON SEQUENCES TO evekill_read;
ALTER ROLE evekill_read SET default_transaction_read_only = on;

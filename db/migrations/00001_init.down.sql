BEGIN TRANSACTION;
    DROP TABLE users;
    DROP TABLE passwords;
    DROP TABLE types;
    DROP TABLE data;
    DROP TABLE refresh_tokens;

    DROP INDEX idx_refresh_tokens_user;
    DROP INDEX idx_data_user;
COMMIT;

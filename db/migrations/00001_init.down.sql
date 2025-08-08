BEGIN TRANSACTION;
    DROP TABLE users;
    DROP TABLE passwords;
    DROP TABLE types;
    DROP TABLE metadata;
    DROP TABLE refresh_tokens;

    DROP INDEX idx_refresh_tokens_user;
    DROP INDEX idx_metadata_user;
COMMIT;

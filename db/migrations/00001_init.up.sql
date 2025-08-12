BEGIN TRANSACTION;
    CREATE TABLE users(
        id UUID PRIMARY KEY DEFAULT gen_random_uuid() ,
        hash TEXT NOT NULL);

    CREATE TABLE passwords(
        id UUID PRIMARY KEY DEFAULT gen_random_uuid (),
        id_user UUID REFERENCES users(id) ON DELETE CASCADE NOT NULL,
        hash TEXT NOT NULL);

    CREATE TABLE types(
        id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        name VARCHAR(30) NOT NULL UNIQUE);

    CREATE TABLE data(
        id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        id_user UUID REFERENCES users(id) ON DELETE RESTRICT NOT NULL,
        id_type INT REFERENCES types(id) ON DELETE RESTRICT NOT NULL,
        plain_data BYTEA,
        note TEXT NOT NULL,
        s3_key TEXT,
        CHECK (
            (plain_data IS NOT NULL AND s3_key IS NULL) OR
            (plain_data IS NULL AND s3_key IS NOT NULL)
            )
        );

    CREATE TABLE refresh_tokens(
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        id_user UUID REFERENCES users(id) ON DELETE CASCADE NOT NULL,
        expires_at timestamptz NOT NULL);

    CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(id_user);
    CREATE INDEX idx_data_user ON data(id_user);

COMMIT;

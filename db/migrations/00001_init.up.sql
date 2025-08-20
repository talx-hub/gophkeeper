BEGIN TRANSACTION;
    CREATE TABLE users(
        id UUID PRIMARY KEY DEFAULT gen_random_uuid() ,
        login TEXT NOT NULL UNIQUE);

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
        encrypted_data BYTEA, -- если id_type соотвествует types.name == "AUTH" или types.name == "CARD", то это поле заполняем, a в S3 хранилище ничего не кладем -> s3_key пуст
        note TEXT NOT NULL,
        s3_key TEXT,
        created_at TIMESTAMP WITH TIME ZONE NOT NULL,
        total_size BIGINT,
        CHECK (
            (encrypted_data IS NOT NULL AND s3_key IS NULL) OR
            (encrypted_data IS NULL AND s3_key IS NOT NULL)
            )
        );

    CREATE TABLE refresh_tokens(
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        id_user UUID REFERENCES users(id) ON DELETE CASCADE NOT NULL,
        expires_at timestamptz NOT NULL);

    CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(id_user);
    CREATE INDEX idx_data_user ON data(id_user);

COMMIT;

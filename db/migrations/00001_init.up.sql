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

    CREATE TABLE metadata(
        id INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        id_user UUID REFERENCES users(id) ON DELETE RESTRICT NOT NULL,
        id_type INT REFERENCES types(id) ON DELETE RESTRICT NOT NULL,
        note TEXT NOT NULL);

    CREATE TABLE refresh_tokens(
        id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        id_user UUID REFERENCES users(id) ON DELETE CASCADE NOT NULL,
        expires_at timestamptz NOT NULL);

    CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(id_user)
    WHERE expires_at > now();

    CREATE INDEX idx_metadata_user ON metadata(id_user);

COMMIT;

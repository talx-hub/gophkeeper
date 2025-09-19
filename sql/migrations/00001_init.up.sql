BEGIN TRANSACTION;
    CREATE TABLE users(
        id UUID PRIMARY KEY DEFAULT gen_random_uuid() ,
        login_hash BYTEA NOT NULL UNIQUE,
        password_phc TEXT NOT NULL
        );

    CREATE TABLE refresh_tokens (
        id          INT PRIMARY KEY GENERATED ALWAYS AS IDENTITY,
        user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
        token_hash  BYTEA NOT NULL UNIQUE,
        expires_at  TIMESTAMPTZ NOT NULL
    );
    CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);

    CREATE TABLE data_types (
        id   INT PRIMARY KEY,
        name VARCHAR(36) NOT NULL UNIQUE
    );

    CREATE TABLE objects (
        id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
        user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
        id_data_type   INT  NOT NULL REFERENCES data_types(id),
        name        TEXT NOT NULL,
        description TEXT,
        storage_locator  TEXT NOT NULL,
        created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
    );
    CREATE INDEX idx_objects_user ON objects(user_id);
    CREATE INDEX idx_objects__data_type
        ON objects (id_data_type);

    CREATE TABLE secret_blobs (
                                  id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
                                  objects_id  UUID NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
                                  sealed      BYTEA NOT NULL
    );

    CREATE TABLE chunk_manifest (
        objects_id   UUID NOT NULL REFERENCES objects(id) ON DELETE CASCADE,
        blob_id      UUID REFERENCES secret_blobs(id) ON DELETE CASCADE,
        chunk_index  INT  NOT NULL CHECK (chunk_index >= 0),
        length       INT  NOT NULL CHECK (length > 0),
        PRIMARY KEY (objects_id, chunk_index),
        UNIQUE (blob_id)
    );


    CREATE INDEX idx_secret_blobs_object ON secret_blobs(objects_id);
COMMIT;

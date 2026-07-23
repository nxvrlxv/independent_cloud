CREATE TABLE folders (
    folder_id       BIGSERIAL PRIMARY KEY,
    parent_id       BIGINT REFERENCES folders(folder_id) DEFAULT NULL,
    owner_id        BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    folder_name     TEXT NOT NULL,
    created_at      TIMESTAMPTZ DEFAULT now()  
);

ALTER TABLE files ADD COLUMN folder_id BIGINT REFERENCES folders(folder_id) ON DELETE CASCADE DEFAULT NULL;

CREATE TABLE IF NOT EXISTS friends (
    id bigserial UNIQUE NOT NULL,
    source_user_id bigint NOT NULL,
    destination_user_id bigint NOT NULL,
    status varchar(10) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP(0) with time zone DEFAULT now(),
    updated_at TIMESTAMP(0) with time zone DEFAULT now(),
    version INTEGER NOT NULL DEFAULT 1,

    FOREIGN KEY (source_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (destination_user_id) REFERENCES users(id) ON DELETE CASCADE
);

INSERT INTO friends 
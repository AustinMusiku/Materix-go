CREATE TABLE IF NOT EXISTS friends (
    id bigserial UNIQUE NOT NULL,
    source_user_id bigint NOT NULL,
    destination_user_id bigint NOT NULL,
    status varchar(10) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMP(0) with time zone DEFAULT now(),
    updated_at TIMESTAMP(0) with time zone DEFAULT now(),
    version INTEGER NOT NULL DEFAULT 1,
    pair_order text GENERATED ALWAYS AS ((LEAST(source_user_id, destination_user_id))::text || ',' ||
                                               (GREATEST(source_user_id, destination_user_id))::text) STORED;

    FOREIGN KEY (source_user_id) REFERENCES users(id) ON DELETE CASCADE,
    FOREIGN KEY (destination_user_id) REFERENCES users(id) ON DELETE CASCADE

    CONSTRAINT unique_friendship_pair UNIQUE (pair_order);
);

INSERT INTO friends 
CREATE TABLE IF NOT EXISTS free_times (
    id bigserial PRIMARY KEY NOT NULL,
    user_id bigint NOT NULL,
    start_time TIMESTAMP(0) with time zone NOT NULL,
    end_time TIMESTAMP(0) with time zone NOT NULL,
    created_at TIMESTAMP(0) with time zone DEFAULT now(),
    updated_at TIMESTAMP(0) with time zone DEFAULT now(),
    tags TEXT[] NOT NULL DEFAULT '{}',
    visibility VARCHAR(10) NOT NULL, -- 'public' or 'private'
    version INTEGER NOT NULL DEFAULT 1,

    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS free_time_viewer (
    id bigserial PRIMARY KEY NOT NULL,
    free_time_id bigint NOT NULL,
    user_id bigint NOT NULL,

    FOREIGN KEY (free_time_id) REFERENCES free_times(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
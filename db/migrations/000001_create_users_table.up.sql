CREATE TABLE IF NOT EXISTS users (
	id bigserial PRIMARY KEY NOT NULL,
	uuid UUID NOT NULL UNIQUE DEFAULT uuid_generate_v4(),
	name TEXT NOT NULL UNIQUE,
	email citext UNIQUE NOT NULL,
	password bytea,
	provider VARCHAR(20) NOT NULL,
	avatar_url TEXT, 
	created_at TIMESTAMP(0) with time zone DEFAULT now(),
	updated_at TIMESTAMP(0) with time zone DEFAULT now(),
	activated BOOLEAN NOT NULL DEFAULT FALSE,
	version INTEGER NOT NULL DEFAULT 1
);
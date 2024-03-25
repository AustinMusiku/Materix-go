ALTER TABLE users ADD COLUMN search tsvector;
UPDATE users SET search = 
    setweight(to_tsvector('english', name), 'A') ||
    setweight(to_tsvector('english', email), 'B');
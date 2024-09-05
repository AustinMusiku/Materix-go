CREATE FUNCTION users_search_trigger() RETURNS trigger AS $$
BEGIN
    NEW.search := 
        setweight(to_tsvector('english', NEW.name), 'A') ||
        setweight(to_tsvector('english', NEW.email), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER users_search_update BEFORE INSERT OR UPDATE
    ON users FOR EACH ROW EXECUTE FUNCTION users_search_trigger();
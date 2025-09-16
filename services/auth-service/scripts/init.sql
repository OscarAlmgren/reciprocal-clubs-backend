-- Create the auth_service database and user
-- This script runs when the PostgreSQL container starts for the first time

-- Create extensions if they don't exist
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Create indexes for better performance (these will be created by GORM migrations, but included for reference)
-- Note: GORM will handle the actual table creation and migrations

-- Grant necessary permissions
GRANT ALL PRIVILEGES ON DATABASE auth_service TO postgres;

-- Create additional schemas if needed
CREATE SCHEMA IF NOT EXISTS audit;

-- Set timezone to UTC
SET timezone = 'UTC';

-- Create a function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language plpgsql;

-- Create a function to generate nanoid
CREATE OR REPLACE FUNCTION generate_nanoid(size INT DEFAULT 12)
RETURNS TEXT AS $$
DECLARE
    alphabet TEXT := '0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz';
    id TEXT := '';
    i INT;
    pos INT;
BEGIN
    FOR i IN 1..size LOOP
        pos := 1 + (FLOOR(RANDOM() * LENGTH(alphabet)))::INT;
        id := id || SUBSTRING(alphabet FROM pos FOR 1);
    END LOOP;
    
    RETURN id;
END;
$$ LANGUAGE plpgsql;
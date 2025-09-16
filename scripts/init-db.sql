-- Database initialization script for Reciprocal Clubs Platform

-- Create databases if they don't exist
SELECT 'CREATE DATABASE reciprocal_clubs' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'reciprocal_clubs')\gexec
SELECT 'CREATE DATABASE auth_service' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'auth_service')\gexec
SELECT 'CREATE DATABASE member_service' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'member_service')\gexec
SELECT 'CREATE DATABASE reciprocal_service' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'reciprocal_service')\gexec
SELECT 'CREATE DATABASE analytics_service' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'analytics_service')\gexec
SELECT 'CREATE DATABASE governance_service' WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'governance_service')\gexec

-- Create users if they don't exist
DO
$do$
BEGIN
   IF NOT EXISTS (
      SELECT FROM pg_catalog.pg_roles
      WHERE  rolname = 'reciprocal_user') THEN
      
      CREATE ROLE reciprocal_user LOGIN PASSWORD 'reciprocal_pass';
   END IF;
END
$do$;

-- Grant privileges
GRANT ALL PRIVILEGES ON DATABASE reciprocal_clubs TO reciprocal_user;
GRANT ALL PRIVILEGES ON DATABASE auth_service TO reciprocal_user;
GRANT ALL PRIVILEGES ON DATABASE member_service TO reciprocal_user;
GRANT ALL PRIVILEGES ON DATABASE reciprocal_service TO reciprocal_user;
GRANT ALL PRIVILEGES ON DATABASE analytics_service TO reciprocal_user;
GRANT ALL PRIVILEGES ON DATABASE governance_service TO reciprocal_user;

-- Set default connection parameters
ALTER ROLE reciprocal_user SET client_encoding TO 'utf8';
ALTER ROLE reciprocal_user SET default_transaction_isolation TO 'read committed';
ALTER ROLE reciprocal_user SET timezone TO 'UTC';

\echo 'Database initialization completed successfully'
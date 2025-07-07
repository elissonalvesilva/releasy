-- migrations/init.sql

CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY,
    service_name VARCHAR(100),
    strategy VARCHAR(20),
    version VARCHAR(20),
    replicas INT,
    image TEXT,
    action VARCHAR(20),
    step VARCHAR(20),
    envs JSONB,
    created_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS slots (
    id TEXT PRIMARY KEY,
    service_name VARCHAR(100),
    version VARCHAR(20),
    weight SMALLINT,
    created_at TIMESTAMP,
    UNIQUE(service_name, version)
);

CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    service_name VARCHAR(100),
    message TEXT,
    created_at TIMESTAMP
);

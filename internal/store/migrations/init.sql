-- migrations/init.sql

CREATE TABLE IF NOT EXISTS deployments (
    id TEXT PRIMARY KEY,
    application VARCHAR(100),
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

CREATE TABLE IF NOT EXISTS services (
    id TEXT PRIMARY KEY,
    application VARCHAR(100),
    name VARCHAR(100),
    version VARCHAR(20),
    image TEXT,
    replicas INT,
    envs JSONB,
    weight INT DEFAULT 100,
    hostname TEXT,
    created_at TIMESTAMP,
    UNIQUE(application, name)
);

CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    application VARCHAR(100),
    service_name VARCHAR(100),
    message TEXT,
    created_at TIMESTAMP
);

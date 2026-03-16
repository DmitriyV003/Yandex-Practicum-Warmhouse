\c device_db;

CREATE TABLE IF NOT EXISTS users (
    id            SERIAL PRIMARY KEY,
    name          VARCHAR(100) NOT NULL,
    email         VARCHAR(100) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS houses (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    address    VARCHAR(255) NOT NULL,
    owner_id   INTEGER      NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS rooms (
    id       SERIAL PRIMARY KEY,
    name     VARCHAR(100) NOT NULL,
    house_id INTEGER      NOT NULL REFERENCES houses(id)
);

CREATE TABLE IF NOT EXISTS devices (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    type       VARCHAR(50)  NOT NULL,
    unit       VARCHAR(20),
    status     VARCHAR(20)  NOT NULL DEFAULT 'inactive',
    room_id    INTEGER      NOT NULL REFERENCES rooms(id),
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS scenarios (
    id         SERIAL PRIMARY KEY,
    name       VARCHAR(100) NOT NULL,
    house_id   INTEGER      NOT NULL REFERENCES houses(id),
    is_active  BOOLEAN      NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS scenario_actions (
    id          SERIAL PRIMARY KEY,
    scenario_id INTEGER      NOT NULL REFERENCES scenarios(id),
    device_id   INTEGER      NOT NULL REFERENCES devices(id),
    action      VARCHAR(100) NOT NULL,
    value       VARCHAR(255) NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_devices_room_id ON devices(room_id);
CREATE INDEX IF NOT EXISTS idx_devices_status ON devices(status);
CREATE INDEX IF NOT EXISTS idx_devices_type ON devices(type);
CREATE INDEX IF NOT EXISTS idx_houses_owner_id ON houses(owner_id);
CREATE INDEX IF NOT EXISTS idx_rooms_house_id ON rooms(house_id);
CREATE INDEX IF NOT EXISTS idx_scenarios_house_id ON scenarios(house_id);
CREATE INDEX IF NOT EXISTS idx_scenario_actions_scenario_id ON scenario_actions(scenario_id);
CREATE INDEX IF NOT EXISTS idx_scenario_actions_device_id ON scenario_actions(device_id);

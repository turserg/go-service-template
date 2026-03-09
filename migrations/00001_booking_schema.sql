-- +goose Up
CREATE TABLE IF NOT EXISTS events (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    currency TEXT NOT NULL DEFAULT 'USD',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS seat_inventory (
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    seat_id TEXT NOT NULL,
    price_minor BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'available',
    reserved_by TEXT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_id, seat_id)
);

CREATE TABLE IF NOT EXISTS reservations (
    id TEXT PRIMARY KEY,
    event_id TEXT NOT NULL REFERENCES events(id),
    user_id TEXT NOT NULL,
    status TEXT NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    total_amount_minor BIGINT NOT NULL,
    currency TEXT NOT NULL,
    idempotency_key TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS reservations_idempotency_key_uq
    ON reservations(idempotency_key);

CREATE TABLE IF NOT EXISTS reservation_seats (
    reservation_id TEXT NOT NULL REFERENCES reservations(id) ON DELETE CASCADE,
    event_id TEXT NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    seat_id TEXT NOT NULL,
    PRIMARY KEY (reservation_id, seat_id)
);

CREATE TABLE IF NOT EXISTS orders (
    id TEXT PRIMARY KEY,
    reservation_id TEXT NOT NULL UNIQUE REFERENCES reservations(id),
    user_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS payment_attempts (
    id BIGSERIAL PRIMARY KEY,
    order_id TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    transaction_id TEXT NOT NULL,
    status TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS seat_inventory_reserved_by_idx ON seat_inventory(reserved_by);
CREATE INDEX IF NOT EXISTS reservations_event_id_idx ON reservations(event_id);
CREATE INDEX IF NOT EXISTS orders_reservation_id_idx ON orders(reservation_id);
CREATE INDEX IF NOT EXISTS payment_attempts_order_id_idx ON payment_attempts(order_id);

INSERT INTO events (id, title, currency)
VALUES
    ('evt_rock_001', 'Rock Night', 'USD'),
    ('evt_jazz_002', 'Jazz Evening', 'USD')
ON CONFLICT (id) DO NOTHING;

INSERT INTO seat_inventory (event_id, seat_id, price_minor, status)
VALUES
    ('evt_rock_001', 'A-1', 5900, 'available'),
    ('evt_rock_001', 'A-2', 5900, 'available'),
    ('evt_rock_001', 'A-3', 5900, 'available'),
    ('evt_rock_001', 'A-4', 5900, 'available'),
    ('evt_rock_001', 'A-5', 5900, 'available'),
    ('evt_jazz_002', 'A-1', 4200, 'available'),
    ('evt_jazz_002', 'A-2', 4200, 'available'),
    ('evt_jazz_002', 'A-3', 4200, 'available'),
    ('evt_jazz_002', 'A-4', 4200, 'available'),
    ('evt_jazz_002', 'A-5', 4200, 'available')
ON CONFLICT (event_id, seat_id) DO NOTHING;

-- +goose Down
DROP TABLE IF EXISTS payment_attempts;
DROP TABLE IF EXISTS orders;
DROP TABLE IF EXISTS reservation_seats;
DROP INDEX IF EXISTS reservations_idempotency_key_uq;
DROP TABLE IF EXISTS reservations;
DROP TABLE IF EXISTS seat_inventory;
DROP TABLE IF EXISTS events;

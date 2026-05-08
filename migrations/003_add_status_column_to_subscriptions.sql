CREATE TYPE subscription_status AS ENUM ('active', 'canceled', 'expired');

ALTER TABLE subscriptions
    ADD COLUMN status subscription_status NOT NULL DEFAULT 'active';
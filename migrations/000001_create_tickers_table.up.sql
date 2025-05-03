CREATE TABLE IF NOT EXISTS tickers
(
    id         SERIAL PRIMARY KEY,
    symbol     VARCHAR(10) UNIQUE NOT NULL,
    notes      TEXT,
    created_at TIMESTAMP DEFAULT now(),
    updated_at TIMESTAMP DEFAULT now()
);

INSERT INTO tickers (symbol, notes)
VALUES ('PLTR', 'Palantir'),
       ('RTHT', 'Richtech Robotics'),
       ('TEM', 'Tempus AI'),
       ('SOUN', 'SoundHound AI');

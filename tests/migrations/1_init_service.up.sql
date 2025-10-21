INSERT INTO service (name)
VALUES ('Yandex Plus')
    ON CONFLICT (name) DO NOTHING;
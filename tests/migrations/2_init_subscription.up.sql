INSERT INTO subscription (user_id, service_id, price_rub, start_date, end_date)
SELECT '60601fee-2bf1-4721-ae6f-7636e79a0cba'::uuid, s.id,
       400,
       '2025-07-01'::date, NULL
FROM service s
WHERE s.name = 'Yandex Plus' ON CONFLICT (user_id, service_id, start_date) DO NOTHING;
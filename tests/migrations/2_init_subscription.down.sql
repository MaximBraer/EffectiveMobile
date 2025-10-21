DELETE FROM subscription
    USING service s
WHERE subscription.service_id = s.id
  AND s.name = 'Yandex Plus'
  AND subscription.user_id = '60601fee-2bf1-4721-ae6f-7636e79a0cba'::uuid
  AND subscription.start_date = '2025-07-01'::date;
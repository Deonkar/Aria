-- Phase 0: Seed data (idempotent). This script truncates and re-seeds to fixed volumes.
BEGIN;

TRUNCATE TABLE
  query_feedback,
  messages,
  conversations,
  payments,
  bookings,
  tasks,
  lead_activities,
  leads,
  properties,
  partners,
  users,
  teams
RESTART IDENTITY CASCADE;

-- Teams (4)
INSERT INTO teams (id, name, department) VALUES
  ('00000000-0000-0000-0000-000000000001', 'UK Pre-Sales', 'pre_sales'),
  ('00000000-0000-0000-0000-000000000002', 'Canada Pre-Sales', 'pre_sales'),
  ('00000000-0000-0000-0000-000000000003', 'Partnerships', 'partnerships'),
  ('00000000-0000-0000-0000-000000000004', 'Supply', 'supply');

-- Users (10): 8 agents + 2 admins
WITH u AS (
  SELECT *
  FROM (VALUES
    ('10000000-0000-0000-0000-000000000001'::uuid, 'google-agent-01', 'agent01@example.com', 'Agent One',  'agent', 'pre_sales',      '00000000-0000-0000-0000-000000000001'::uuid),
    ('10000000-0000-0000-0000-000000000002'::uuid, 'google-agent-02', 'agent02@example.com', 'Agent Two',  'agent', 'pre_sales',      '00000000-0000-0000-0000-000000000001'::uuid),
    ('10000000-0000-0000-0000-000000000003'::uuid, 'google-agent-03', 'agent03@example.com', 'Agent Three','agent', 'pre_sales',      '00000000-0000-0000-0000-000000000002'::uuid),
    ('10000000-0000-0000-0000-000000000004'::uuid, 'google-agent-04', 'agent04@example.com', 'Agent Four', 'agent', 'pre_sales',      '00000000-0000-0000-0000-000000000002'::uuid),
    ('10000000-0000-0000-0000-000000000005'::uuid, 'google-agent-05', 'agent05@example.com', 'Agent Five', 'agent', 'partnerships',  '00000000-0000-0000-0000-000000000003'::uuid),
    ('10000000-0000-0000-0000-000000000006'::uuid, 'google-agent-06', 'agent06@example.com', 'Agent Six',  'agent', 'partnerships',  '00000000-0000-0000-0000-000000000003'::uuid),
    ('10000000-0000-0000-0000-000000000007'::uuid, 'google-agent-07', 'agent07@example.com', 'Agent Seven','agent', 'supply',        '00000000-0000-0000-0000-000000000004'::uuid),
    ('10000000-0000-0000-0000-000000000008'::uuid, 'google-agent-08', 'agent08@example.com', 'Agent Eight','agent', 'supply',        '00000000-0000-0000-0000-000000000004'::uuid),
    ('20000000-0000-0000-0000-000000000001'::uuid, 'google-admin-01', 'admin01@example.com', 'Admin One',  'admin', 'pre_sales',      '00000000-0000-0000-0000-000000000001'::uuid),
    ('20000000-0000-0000-0000-000000000002'::uuid, 'google-admin-02', 'admin02@example.com', 'Admin Two',  'admin', 'finance',        '00000000-0000-0000-0000-000000000001'::uuid)
  ) AS t(id, google_id, email, full_name, role, department, team_id)
)
INSERT INTO users (id, google_id, email, full_name, role, department, team_id, timezone)
SELECT id, google_id, email, full_name, role, department, team_id, 'Asia/Kolkata'
FROM u;

-- Partners (20)
INSERT INTO partners (name, partner_type, country, contact_email, commission_rate)
SELECT
  CASE
    WHEN gs <= 15 THEN 'University Partner ' || gs
    ELSE 'Portal Partner ' || gs
  END,
  CASE
    WHEN gs <= 15 THEN 'university'
    ELSE 'portal'
  END,
  (ARRAY['UK','Canada','India'])[1 + (gs % 3)],
  'partner' || gs || '@example.com',
  5.00 + (gs % 10) * 0.25
FROM generate_series(1,20) gs;

-- Properties (50)
INSERT INTO properties (name, city, country, property_type, total_units, available_units, price_per_week, university_proximity_km, rating)
SELECT
  'Property ' || gs,
  CASE
    WHEN gs <= 20 THEN 'London'
    WHEN gs <= 30 THEN 'Manchester'
    WHEN gs <= 40 THEN 'Toronto'
    ELSE 'Vancouver'
  END,
  CASE
    WHEN gs <= 30 THEN 'UK'
    ELSE 'Canada'
  END,
  (ARRAY['student_hall','studio','shared_house'])[1 + (gs % 3)],
  100 + (gs % 50),
  10 + (gs % 20),
  150.00 + (gs % 50) * 2.5,
  0.5 + (gs % 15) * 0.2,
  3.5 + (gs % 15) * 0.1
FROM generate_series(1,50) gs;

-- Leads (500), distributed across 8 agents (~62/agent) with realistic-ish state/priority
WITH agents AS (
  SELECT id, row_number() OVER (ORDER BY id) AS rn
  FROM users
  WHERE role = 'agent'
  ORDER BY id
),
partner_ids AS (
  SELECT id, row_number() OVER (ORDER BY id) AS rn FROM partners
),
lead_rows AS (
  SELECT gs AS n,
         (SELECT id FROM agents WHERE rn = 1 + ((gs-1) % 8)) AS assigned_agent_id,
         (SELECT id FROM partner_ids WHERE rn = 1 + ((gs-1) % 20)) AS partner_id
  FROM generate_series(1,500) gs
)
INSERT INTO leads (
  external_id, assigned_agent_id, partner_id,
  first_name, last_name, email, phone, whatsapp_number,
  source_channel, state, priority, source_country, destination_country, destination_city,
  preferred_room_type, preferred_move_in_month, preferred_move_in_year,
  budget_min, budget_max, last_activity_at, created_at, updated_at
)
SELECT
  'L-' || lpad(n::text, 4, '0'),
  assigned_agent_id,
  partner_id,
  'LeadFirst' || n,
  'LeadLast' || n,
  'lead' || n || '@example.com',
  '+100000' || lpad((1000 + n)::text, 6, '0'),
  '+100000' || lpad((1000 + n)::text, 6, '0'),
  (ARRAY['partner','organic','paid','referral','direct'])[1 + (n % 5)],
  CASE
    WHEN (n % 100) < 15 THEN 'new'
    WHEN (n % 100) < 40 THEN 'contacted'
    WHEN (n % 100) < 70 THEN 'interested'
    WHEN (n % 100) < 85 THEN 'qualified'
    WHEN (n % 100) < 95 THEN 'booked'
    WHEN (n % 100) < 99 THEN 'lost'
    ELSE 'junk'
  END,
  CASE
    WHEN (n % 100) < 20 THEN 'low'
    WHEN (n % 100) < 65 THEN 'medium'
    WHEN (n % 100) < 90 THEN 'high'
    ELSE 'urgent'
  END,
  (ARRAY['India','Nigeria','China','UK','Canada'])[1 + (n % 5)],
  (ARRAY['UK','Canada'])[1 + (n % 2)],
  CASE WHEN (n % 2) = 0 THEN 'London' ELSE 'Toronto' END,
  (ARRAY['studio','ensuite','shared'])[1 + (n % 3)],
  1 + (n % 12),
  2026,
  120.00 + (n % 40) * 5,
  200.00 + (n % 60) * 5,
  NOW() - ((n % 21) || ' days')::interval,
  NOW() - ((n % 60) || ' days')::interval,
  NOW();

-- Lead activities (1000): 2 per lead
WITH lead_ids AS (
  SELECT id, assigned_agent_id, row_number() OVER (ORDER BY id) AS rn
  FROM leads
),
rows AS (
  SELECT
    (SELECT id FROM lead_ids WHERE rn = 1 + ((gs-1) % 500)) AS lead_id,
    (SELECT assigned_agent_id FROM lead_ids WHERE rn = 1 + ((gs-1) % 500)) AS agent_id,
    gs AS n
  FROM generate_series(1,1000) gs
)
INSERT INTO lead_activities (
  lead_id, agent_id, activity_type, subject, body, direction, duration_seconds, outcome, occurred_at
)
SELECT
  lead_id,
  agent_id,
  (ARRAY['call','email','whatsapp','note','state_change','assignment','sms'])[1 + (n % 7)],
  'Activity ' || n,
  'Body for activity ' || n,
  (ARRAY['inbound','outbound'])[1 + (n % 2)],
  30 + (n % 600),
  (ARRAY['connected','no_answer','voicemail','bounced'])[1 + (n % 4)],
  NOW() - ((n % 45) || ' days')::interval;

-- Tasks (300)
WITH lead_ids AS (
  SELECT id, assigned_agent_id, row_number() OVER (ORDER BY id) AS rn
  FROM leads
),
rows AS (
  SELECT
    (SELECT id FROM lead_ids WHERE rn = 1 + ((gs-1) % 500)) AS lead_id,
    (SELECT assigned_agent_id FROM lead_ids WHERE rn = 1 + ((gs-1) % 500)) AS assigned_agent_id,
    gs AS n
  FROM generate_series(1,300) gs
)
INSERT INTO tasks (
  lead_id, assigned_agent_id, created_by_agent_id, task_type, title, description,
  priority, status, due_date, completed_at, snoozed_until
)
SELECT
  lead_id,
  assigned_agent_id,
  assigned_agent_id,
  (ARRAY['follow_up_call','send_email','send_whatsapp','schedule_viewing','send_proposal','other'])[1 + (n % 6)],
  'Task ' || n,
  'Description ' || n,
  CASE
    WHEN (n % 100) < 20 THEN 'low'
    WHEN (n % 100) < 65 THEN 'medium'
    WHEN (n % 100) < 90 THEN 'high'
    ELSE 'urgent'
  END,
  CASE
    WHEN (n % 100) < 40 THEN 'pending'
    WHEN (n % 100) < 80 THEN 'completed'
    WHEN (n % 100) < 95 THEN 'pending'   -- used with past due_date to create overdue
    ELSE 'snoozed'
  END,
  CASE
    WHEN (n % 100) < 15 THEN CURRENT_DATE - (1 + (n % 10)) -- overdue pending slice
    ELSE CURRENT_DATE + (n % 14)
  END,
  CASE WHEN (n % 100) >= 40 AND (n % 100) < 80 THEN NOW() - ((n % 10) || ' days')::interval ELSE NULL END,
  CASE WHEN (n % 100) >= 95 THEN NOW() + ((n % 5) || ' days')::interval ELSE NULL END;

-- Bookings (200): link to first 200 leads
WITH lead_ids AS (
  SELECT id, assigned_agent_id, row_number() OVER (ORDER BY id) AS rn
  FROM leads
),
property_ids AS (
  SELECT id, row_number() OVER (ORDER BY id) AS rn FROM properties
)
INSERT INTO bookings (
  external_booking_ref, lead_id, property_id, agent_id, status, room_type,
  move_in_date, move_out_date, duration_weeks, weekly_rent, total_rent,
  deposit_amount, deposit_paid, commission_amount, commission_paid
)
SELECT
  'B-' || lpad(rn::text, 4, '0'),
  id,
  (SELECT id FROM property_ids WHERE rn = 1 + ((lead_ids.rn-1) % 50)),
  assigned_agent_id,
  (ARRAY['pending','confirmed','cancelled','completed','refunded'])[1 + (rn % 5)],
  (ARRAY['studio','ensuite','shared'])[1 + (rn % 3)],
  CURRENT_DATE + (rn % 30),
  CURRENT_DATE + (rn % 30) + ((16 + (rn % 24)) * 7),
  16 + (rn % 24),
  160.00 + (rn % 40) * 2.5,
  (160.00 + (rn % 40) * 2.5) * (16 + (rn % 24)),
  200.00 + (rn % 10) * 25,
  (rn % 2) = 0,
  400.00 + (rn % 20) * 15,
  (rn % 3) = 0
FROM lead_ids
WHERE rn <= 200;

-- Payments (150): link to first 150 bookings
WITH booking_ids AS (
  SELECT id, lead_id, agent_id, row_number() OVER (ORDER BY id) AS rn
  FROM bookings
)
INSERT INTO payments (
  booking_id, lead_id, agent_id, payment_type, amount, currency, status,
  payment_method, gateway_transaction_id, paid_at, due_date, notes
)
SELECT
  id,
  lead_id,
  agent_id,
  (ARRAY['deposit','first_installment','second_installment','full_payment','refund','commission'])[1 + (rn % 6)],
  250.00 + (rn % 50) * 10,
  'GBP',
  (ARRAY['pending','processing','completed','failed','refunded','disputed'])[1 + (rn % 6)],
  (ARRAY['card','bank_transfer','paypal','stripe'])[1 + (rn % 4)],
  'GTX-' || lpad(rn::text, 6, '0'),
  CASE WHEN (rn % 6) >= 2 THEN NOW() - ((rn % 15) || ' days')::interval ELSE NULL END,
  CURRENT_DATE + (rn % 20),
  'Seed payment ' || rn
FROM booking_ids
WHERE rn <= 150;

COMMIT;


-- Rich demo data for agent 10000000-0000-0000-0000-000000000001 (demo JWT / quick login).
-- Safe to run multiple times: uses fixed UUIDs + ON CONFLICT DO NOTHING.

BEGIN;

-- Team + user (in case only partial seed existed)
INSERT INTO teams (id, name, department)
VALUES ('c0000001-3333-3333-3333-333333333333', 'UK Pre-Sales', 'pre_sales')
ON CONFLICT (id) DO NOTHING;

INSERT INTO users (id, google_id, email, full_name, role, department, team_id, timezone)
VALUES (
  '10000000-0000-0000-0000-000000000001',
  'google-agent-01',
  'agent01@example.com',
  'Agent One',
  'agent',
  'pre_sales',
  'c0000001-3333-3333-3333-333333333333',
  'Europe/London'
)
ON CONFLICT (id) DO UPDATE SET
  email = EXCLUDED.email,
  full_name = EXCLUDED.full_name,
  role = EXCLUDED.role,
  team_id = EXCLUDED.team_id;

-- Partner + property (bookings need property)
INSERT INTO partners (id, name, partner_type, country, contact_email, commission_rate)
VALUES (
  'a0000001-1111-1111-1111-111111111111',
  'University of Manchester — Student Housing',
  'university',
  'UK',
  'partnerships@demo-uni.example',
  15.00
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO properties (id, name, city, country, address_line1, postcode, property_type, total_units, available_units, price_per_week, currency, rating)
VALUES (
  'b0000001-2222-2222-2222-222222222222',
  'Riverbank Student Halls',
  'Manchester',
  'UK',
  '1 Riverbank Way',
  'M1 5QD',
  'halls',
  320,
  42,
  189.00,
  'GBP',
  4.6
)
ON CONFLICT (id) DO NOTHING;

-- Leads assigned to demo agent (varied state / priority for interesting questions)
INSERT INTO leads (
  id, external_id, assigned_agent_id, partner_id,
  first_name, last_name, email, phone,
  state, priority, source_country, destination_country, destination_city,
  preferred_room_type, budget_min, budget_max, budget_currency,
  last_activity_at, assigned_at
) VALUES
(
  '11111111-1111-1111-1111-111111111101',
  'DEMO-L-001',
  '10000000-0000-0000-0000-000000000001',
  'a0000001-1111-1111-1111-111111111111',
  'Priya', 'Sharma', 'priya.sharma@example.com', '+44 7700 900101',
  'new', 'high', 'India', 'UK', 'Manchester',
  'ensuite', 650.00, 950.00, 'GBP',
  NOW() - INTERVAL '2 days', NOW() - INTERVAL '5 days'
),
(
  '11111111-1111-1111-1111-111111111102',
  'DEMO-L-002',
  '10000000-0000-0000-0000-000000000001',
  'a0000001-1111-1111-1111-111111111111',
  'James', 'Okonkwo', 'j.okonkwo@example.com', '+44 7700 900102',
  'contacted', 'urgent', 'Nigeria', 'UK', 'Manchester',
  'studio', 800.00, 1100.00, 'GBP',
  NOW() - INTERVAL '1 day', NOW() - INTERVAL '3 days'
),
(
  '11111111-1111-1111-1111-111111111103',
  'DEMO-L-003',
  '10000000-0000-0000-0000-000000000001',
  'a0000001-1111-1111-1111-111111111111',
  'Chen', 'Wei', 'chen.wei@example.com', '+44 7700 900103',
  'qualified', 'medium', 'China', 'UK', 'Manchester',
  'shared', 500.00, 750.00, 'GBP',
  NOW() - INTERVAL '6 hours', NOW() - INTERVAL '10 days'
),
(
  '11111111-1111-1111-1111-111111111104',
  'DEMO-L-004',
  '10000000-0000-0000-0000-000000000001',
  'a0000001-1111-1111-1111-111111111111',
  'Emily', 'Jones', 'emily.jones@example.com', '+44 7700 900104',
  'interested', 'high', 'UK', 'UK', 'Manchester',
  'ensuite', 700.00, 900.00, 'GBP',
  NOW() - INTERVAL '12 hours', NOW() - INTERVAL '7 days'
),
(
  '11111111-1111-1111-1111-111111111105',
  'DEMO-L-005',
  '10000000-0000-0000-0000-000000000001',
  'a0000001-1111-1111-1111-111111111111',
  'Alex', 'Murphy', 'alex.murphy@example.com', '+44 7700 900105',
  'new', 'low', 'Ireland', 'UK', 'Leeds',
  'studio', 400.00, 600.00, 'GBP',
  NOW() - INTERVAL '3 days', NOW() - INTERVAL '1 day'
)
ON CONFLICT (id) DO NOTHING;

-- Tasks (due dates spread around today / future)
INSERT INTO tasks (id, lead_id, assigned_agent_id, task_type, title, description, priority, status, due_date)
VALUES
(
  '22222222-2222-2222-2222-222222222201',
  '11111111-1111-1111-1111-111111111101',
  '10000000-0000-0000-0000-000000000001',
  'call',
  'Welcome call — Priya',
  'Intro call and confirm budget range',
  'high',
  'pending',
  CURRENT_DATE
),
(
  '22222222-2222-2222-2222-222222222202',
  '11111111-1111-1111-1111-111111111102',
  '10000000-0000-0000-0000-000000000001',
  'follow_up',
  'Send room options PDF',
  'Studio + ensuite options near campus',
  'urgent',
  'pending',
  CURRENT_DATE + 1
),
(
  '22222222-2222-2222-2222-222222222203',
  '11111111-1111-1111-1111-111111111103',
  '10000000-0000-0000-0000-000000000001',
  'viewing',
  'Book virtual viewing',
  'Confirm Teams link with property',
  'medium',
  'pending',
  CURRENT_DATE + 3
),
(
  '22222222-2222-2222-2222-222222222204',
  '11111111-1111-1111-1111-111111111104',
  '10000000-0000-0000-0000-000000000001',
  'email',
  'Send contract draft',
  NULL,
  'high',
  'pending',
  CURRENT_DATE - 1
)
ON CONFLICT (id) DO NOTHING;

-- Bookings (two confirmed-style, one pending)
INSERT INTO bookings (
  id, lead_id, property_id, agent_id, external_booking_ref, status,
  room_type, move_in_date, move_out_date, duration_weeks, weekly_rent, currency,
  deposit_paid, confirmed_at
) VALUES
(
  '33333333-3333-3333-3333-333333333301',
  '11111111-1111-1111-1111-111111111103',
  'b0000001-2222-2222-2222-222222222222',
  '10000000-0000-0000-0000-000000000001',
  'BK-DEMO-001',
  'confirmed',
  'ensuite',
  DATE '2026-09-12',
  DATE '2027-07-15',
  44,
  189.00,
  'GBP',
  TRUE,
  NOW() - INTERVAL '2 days'
),
(
  '33333333-3333-3333-3333-333333333302',
  '11111111-1111-1111-1111-111111111104',
  'b0000001-2222-2222-2222-222222222222',
  '10000000-0000-0000-0000-000000000001',
  'BK-DEMO-002',
  'pending',
  'ensuite',
  DATE '2026-10-01',
  NULL,
  NULL,
  195.00,
  'GBP',
  FALSE,
  NULL
)
ON CONFLICT (id) DO NOTHING;

-- Sample lead activities (for “recent activity” style questions)
INSERT INTO lead_activities (id, lead_id, agent_id, activity_type, subject, body, direction, outcome, occurred_at)
VALUES
(
  '44444444-4444-4444-4444-444444444401',
  '11111111-1111-1111-1111-111111111101',
  '10000000-0000-0000-0000-000000000001',
  'call',
  'Inbound budget check',
  'Student asked about bills-inclusive options',
  'inbound',
  'connected',
  NOW() - INTERVAL '1 day'
),
(
  '44444444-4444-4444-4444-444444444402',
  '11111111-1111-1111-1111-111111111102',
  '10000000-0000-0000-0000-000000000001',
  'email',
  'Sent brochure',
  'PDF with Manchester options',
  'outbound',
  NULL,
  NOW() - INTERVAL '4 hours'
)
ON CONFLICT (id) DO NOTHING;

COMMIT;

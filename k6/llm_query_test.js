import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 2 },
    { duration: '20s', target: 2 },
    { duration: '10s', target: 5 },
    { duration: '20s', target: 5 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.10'],
    http_req_duration: ['p(99)<10000'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const JWT = __ENV.JWT_TOKEN;

const QUESTIONS = [
  'what tasks are overdue?',
  'show me recent activity on my leads',
  'breakdown of my leads by state',
  'which leads are qualified?',
  'how many bookings did I close this month?'
];

export default function () {
  const question = QUESTIONS[Math.floor(Math.random() * QUESTIONS.length)];
  const res = http.post(
    `${BASE_URL}/query`,
    JSON.stringify({ question }),
    {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${JWT}`,
      },
      timeout: '60s',
    }
  );

  check(res, {
    'status is 200': (r) => r.status === 200,
    'no error payload': (r) => !(r.body || '').includes('"error"'),
  });

  sleep(2);
}

import http from 'k6/http';
import { check, sleep } from 'k6';

export const options = {
  stages: [
    { duration: '10s', target: 10 },
    { duration: '20s', target: 10 },
    { duration: '10s', target: 50 },
    { duration: '30s', target: 50 },
    { duration: '10s', target: 0 },
  ],
  thresholds: {
    http_req_duration: ['p(99)<200'],
    http_req_failed: ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';
const JWT = __ENV.JWT_TOKEN;

export default function () {
  const res = http.post(
    `${BASE_URL}/query`,
    JSON.stringify({ question: 'how many leads are assigned to me?' }),
    {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${JWT}`,
      },
      timeout: '10s',
    }
  );

  check(res, {
    'status is 200': (r) => r.status === 200,
    'was_cached true': (r) => (r.body || '').includes('"was_cached":true'),
  });

  sleep(1);
}

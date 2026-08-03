import http from 'k6/http';
import { check, sleep } from 'k6';
import { uuidv4 } from 'https://jslib.k6.io/k6-utils/1.4.0/index.js';

export const options = {
  vus: 5,
  duration: '30s',
  thresholds: {
    http_req_failed: ['rate<0.05'],
    http_req_duration: ['p(95)<800'],
  },
};

const BASE = __ENV.GATEWAY_URL || 'http://localhost:8080';

export function setup() {
  const email = `load-${uuidv4()}@example.com`;
  const password = 'password1';
  let res = http.post(`${BASE}/v1/auth/register`, JSON.stringify({ email, password }), {
    headers: { 'Content-Type': 'application/json' },
  });
  check(res, { 'register 201': (r) => r.status === 201 });
  res = http.post(`${BASE}/v1/auth/login`, JSON.stringify({ email, password }), {
    headers: { 'Content-Type': 'application/json' },
  });
  check(res, { 'login 200': (r) => r.status === 200 });
  const token = res.json('token');
  const w1 = http.post(`${BASE}/v1/wallets`, JSON.stringify({ currency: 'RUB' }), {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  });
  const w2 = http.post(`${BASE}/v1/wallets`, JSON.stringify({ currency: 'RUB' }), {
    headers: { Authorization: `Bearer ${token}`, 'Content-Type': 'application/json' },
  });
  return {
    token,
    from: w1.json('wallet_id'),
    to: w2.json('wallet_id'),
  };
}

export default function (data) {
  const key = uuidv4();
  const res = http.post(
    `${BASE}/v1/transfers`,
    JSON.stringify({
      from_wallet_id: data.from,
      to_wallet_id: data.to,
      amount_minor: 1,
      currency: 'RUB',
    }),
    {
      headers: {
        Authorization: `Bearer ${data.token}`,
        'Content-Type': 'application/json',
        'Idempotency-Key': key,
      },
    },
  );
  // may fail with insufficient funds if not seeded — still exercises auth path
  check(res, {
    'not 401': (r) => r.status !== 401,
  });

  const replay = http.post(
    `${BASE}/v1/transfers`,
    JSON.stringify({
      from_wallet_id: data.from,
      to_wallet_id: data.to,
      amount_minor: 1,
      currency: 'RUB',
    }),
    {
      headers: {
        Authorization: `Bearer ${data.token}`,
        'Content-Type': 'application/json',
        'Idempotency-Key': key,
      },
    },
  );
  check(replay, { 'idempotent status matches': (r) => r.status === res.status });
  sleep(0.3);
}

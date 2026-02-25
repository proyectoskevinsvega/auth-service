import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const tokenValidationTime = new Trend('token_validation_duration');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 100 },   // Ramp up to 100 users
    { duration: '1m', target: 500 },    // Ramp up to 500 users
    { duration: '2m', target: 1000 },   // Ramp up to 1000 users
    { duration: '1m', target: 1000 },   // Stay at 1000 users
    { duration: '30s', target: 0 },     // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<50', 'p(99)<100'], // 95% < 50ms, 99% < 100ms
    'errors': ['rate<0.01'], // Error rate < 1%
    'http_req_failed': ['rate<0.01'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Setup: Create a user and get a token
export function setup() {
  const registerPayload = JSON.stringify({
    username: `k6user_${Date.now()}`,
    email: `k6test_${Date.now()}@example.com`,
    password: 'K6TestPass123!',
  });

  const registerRes = http.post(`${BASE_URL}/auth/register`, registerPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (registerRes.status !== 201) {
    console.error('Failed to register user:', registerRes.body);
    return null;
  }

  const loginPayload = JSON.stringify({
    identifier: JSON.parse(registerPayload).username,
    password: 'K6TestPass123!',
  });

  const loginRes = http.post(`${BASE_URL}/auth/login`, loginPayload, {
    headers: { 'Content-Type': 'application/json' },
  });

  if (loginRes.status !== 200) {
    console.error('Failed to login:', loginRes.body);
    return null;
  }

  const { access_token } = JSON.parse(loginRes.body);
  console.log('Setup complete. Access token obtained.');
  return { token: access_token };
}

// Main test function
export default function(data) {
  if (!data || !data.token) {
    console.error('No token available');
    return;
  }

  const startTime = Date.now();

  const res = http.get(`${BASE_URL}/auth/me`, {
    headers: {
      'Authorization': `Bearer ${data.token}`,
    },
  });

  const duration = Date.now() - startTime;
  tokenValidationTime.add(duration);

  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 50ms': (r) => r.timings.duration < 50,
    'response time < 100ms': (r) => r.timings.duration < 100,
    'has user data': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.id !== undefined;
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!success);

  // Small pause between iterations (realistic)
  sleep(0.1);
}

// Summary at the end
export function handleSummary(data) {
  const avgDuration = data.metrics.http_req_duration.values.avg;
  const p95Duration = data.metrics.http_req_duration.values['p(95)'];
  const p99Duration = data.metrics.http_req_duration.values['p(99)'];
  const errorRate = data.metrics.errors.values.rate * 100;
  const rps = data.metrics.http_reqs.values.rate;

  console.log('\n========================================');
  console.log('📊 VALIDATE TOKEN TEST RESULTS');
  console.log('========================================');
  console.log(`⚡ Requests per second: ${rps.toFixed(2)}`);
  console.log(`⏱️  Avg response time: ${avgDuration.toFixed(2)}ms`);
  console.log(`⏱️  P95 response time: ${p95Duration.toFixed(2)}ms`);
  console.log(`⏱️  P99 response time: ${p99Duration.toFixed(2)}ms`);
  console.log(`❌ Error rate: ${errorRate.toFixed(2)}%`);
  console.log('========================================\n');

  return {
    'stdout': JSON.stringify(data, null, 2),
  };
}

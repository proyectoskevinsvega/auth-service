import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const loginTime = new Trend('login_duration');
const loginSuccess = new Counter('login_success');
const loginFailed = new Counter('login_failed');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 50 },    // Ramp up to 50 users
    { duration: '1m', target: 200 },    // Ramp up to 200 users
    { duration: '2m', target: 500 },    // Ramp up to 500 users
    { duration: '1m', target: 500 },    // Stay at 500 users
    { duration: '30s', target: 0 },     // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<100', 'p(99)<200'], // 95% < 100ms, 99% < 200ms
    'errors': ['rate<0.05'], // Error rate < 5%
    'login_duration': ['p(95)<80', 'p(99)<150'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

// Setup: Create test users
export function setup() {
  const users = [];
  const numUsers = 100;

  console.log(`Creating ${numUsers} test users...`);

  for (let i = 0; i < numUsers; i++) {
    const timestamp = Date.now() + i;
    const user = {
      username: `k6login_${timestamp}`,
      email: `k6login_${timestamp}@example.com`,
      password: 'K6TestPass123!',
    };

    const registerPayload = JSON.stringify(user);
    const res = http.post(`${BASE_URL}/auth/register`, registerPayload, {
      headers: { 'Content-Type': 'application/json' },
    });

    if (res.status === 201) {
      users.push(user);
    }

    // Small delay to avoid rate limiting during setup
    sleep(0.05);
  }

  console.log(`Setup complete. Created ${users.length} users.`);
  return { users };
}

// Main test function
export default function(data) {
  if (!data || !data.users || data.users.length === 0) {
    console.error('No users available');
    return;
  }

  // Pick a random user
  const user = data.users[Math.floor(Math.random() * data.users.length)];

  const payload = JSON.stringify({
    identifier: user.username,
    password: user.password,
  });

  const startTime = Date.now();

  const res = http.post(`${BASE_URL}/auth/login`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  const duration = Date.now() - startTime;
  loginTime.add(duration);

  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'has access_token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token !== undefined;
      } catch {
        return false;
      }
    },
    'has refresh_token': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.refresh_token !== undefined;
      } catch {
        return false;
      }
    },
    'response time < 100ms': (r) => r.timings.duration < 100,
  });

  if (success) {
    loginSuccess.add(1);
  } else {
    loginFailed.add(1);
    console.error(`Login failed: ${res.status} - ${res.body}`);
  }

  errorRate.add(!success);

  // Realistic pause between logins
  sleep(Math.random() * 2 + 1); // 1-3 seconds
}

// Summary at the end
export function handleSummary(data) {
  const avgDuration = data.metrics.http_req_duration.values.avg;
  const p95Duration = data.metrics.http_req_duration.values['p(95)'];
  const p99Duration = data.metrics.http_req_duration.values['p(99)'];
  const errorRate = data.metrics.errors.values.rate * 100;
  const rps = data.metrics.http_reqs.values.rate;

  const loginAvg = data.metrics.login_duration?.values.avg || 0;
  const loginP95 = data.metrics.login_duration?.values['p(95)'] || 0;
  const loginP99 = data.metrics.login_duration?.values['p(99)'] || 0;

  console.log('\n========================================');
  console.log('🔐 LOGIN TEST RESULTS');
  console.log('========================================');
  console.log(`⚡ Requests per second: ${rps.toFixed(2)}`);
  console.log(`⏱️  Avg login time: ${loginAvg.toFixed(2)}ms`);
  console.log(`⏱️  P95 login time: ${loginP95.toFixed(2)}ms`);
  console.log(`⏱️  P99 login time: ${loginP99.toFixed(2)}ms`);
  console.log(`✅ Success rate: ${(100 - errorRate).toFixed(2)}%`);
  console.log(`❌ Error rate: ${errorRate.toFixed(2)}%`);
  console.log('========================================\n');

  return {
    'stdout': JSON.stringify(data, null, 2),
  };
}

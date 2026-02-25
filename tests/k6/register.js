import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';

// Custom metrics
const errorRate = new Rate('errors');
const registerTime = new Trend('register_duration');
const registerSuccess = new Counter('register_success');
const registerFailed = new Counter('register_failed');

// Test configuration
export const options = {
  stages: [
    { duration: '30s', target: 20 },    // Ramp up to 20 users
    { duration: '1m', target: 100 },    // Ramp up to 100 users
    { duration: '2m', target: 300 },    // Ramp up to 300 users
    { duration: '1m', target: 300 },    // Stay at 300 users
    { duration: '30s', target: 0 },     // Ramp down
  ],
  thresholds: {
    'http_req_duration': ['p(95)<150', 'p(99)<300'], // 95% < 150ms, 99% < 300ms
    'errors': ['rate<0.05'], // Error rate < 5%
    'register_duration': ['p(95)<120', 'p(99)<250'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

let userCounter = 0;

// Main test function
export default function() {
  userCounter++;
  const timestamp = Date.now();
  const uniqueId = `${timestamp}_${userCounter}_${__VU}_${__ITER}`;

  const payload = JSON.stringify({
    username: `k6reg_${uniqueId}`,
    email: `k6reg_${uniqueId}@example.com`,
    password: 'K6TestPass123!',
  });

  const startTime = Date.now();

  const res = http.post(`${BASE_URL}/auth/register`, payload, {
    headers: { 'Content-Type': 'application/json' },
  });

  const duration = Date.now() - startTime;
  registerTime.add(duration);

  const success = check(res, {
    'status is 201': (r) => r.status === 201,
    'has user id': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.id !== undefined;
      } catch {
        return false;
      }
    },
    'has username': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.username !== undefined;
      } catch {
        return false;
      }
    },
    'response time < 150ms': (r) => r.timings.duration < 150,
  });

  if (success) {
    registerSuccess.add(1);
  } else {
    registerFailed.add(1);
    if (res.status !== 429) { // Don't log rate limit errors
      console.error(`Register failed: ${res.status} - ${res.body}`);
    }
  }

  errorRate.add(!success);

  // Realistic pause between registrations
  sleep(Math.random() * 3 + 2); // 2-5 seconds
}

// Summary at the end
export function handleSummary(data) {
  const avgDuration = data.metrics.http_req_duration.values.avg;
  const p95Duration = data.metrics.http_req_duration.values['p(95)'];
  const p99Duration = data.metrics.http_req_duration.values['p(99)'];
  const errorRate = data.metrics.errors.values.rate * 100;
  const rps = data.metrics.http_reqs.values.rate;

  const regAvg = data.metrics.register_duration?.values.avg || 0;
  const regP95 = data.metrics.register_duration?.values['p(95)'] || 0;
  const regP99 = data.metrics.register_duration?.values['p(99)'] || 0;

  console.log('\n========================================');
  console.log('📝 REGISTER TEST RESULTS');
  console.log('========================================');
  console.log(`⚡ Requests per second: ${rps.toFixed(2)}`);
  console.log(`⏱️  Avg register time: ${regAvg.toFixed(2)}ms`);
  console.log(`⏱️  P95 register time: ${regP95.toFixed(2)}ms`);
  console.log(`⏱️  P99 register time: ${regP99.toFixed(2)}ms`);
  console.log(`✅ Success rate: ${(100 - errorRate).toFixed(2)}%`);
  console.log(`❌ Error rate: ${errorRate.toFixed(2)}%`);
  console.log('========================================\n');

  return {
    'stdout': JSON.stringify(data, null, 2),
  };
}

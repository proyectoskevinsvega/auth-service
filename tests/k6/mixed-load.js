import http from 'k6/http';
import { check, sleep } from 'k6';
import { Rate, Trend, Counter } from 'k6/metrics';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// Custom metrics
const errorRate = new Rate('errors');
const validateTokenTime = new Trend('validate_token_duration');
const loginTime = new Trend('login_duration');
const registerTime = new Trend('register_duration');
const refreshTime = new Trend('refresh_duration');

const validateTokenCount = new Counter('validate_token_count');
const loginCount = new Counter('login_count');
const registerCount = new Counter('register_count');
const refreshCount = new Counter('refresh_count');

// Test configuration - Realistic mixed load
// 70% validate, 20% refresh, 8% login, 2% register
export const options = {
  scenarios: {
    validate_token: {
      executor: 'ramping-vus',
      exec: 'validateToken',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 200 },   // Ramp up
        { duration: '2m', target: 700 },    // 70% of 1000
        { duration: '1m', target: 700 },    // Sustain
        { duration: '30s', target: 0 },     // Ramp down
      ],
    },
    refresh_token: {
      executor: 'ramping-vus',
      exec: 'refreshToken',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 50 },    // Ramp up
        { duration: '2m', target: 200 },    // 20% of 1000
        { duration: '1m', target: 200 },    // Sustain
        { duration: '30s', target: 0 },     // Ramp down
      ],
    },
    login: {
      executor: 'ramping-vus',
      exec: 'login',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 20 },    // Ramp up
        { duration: '2m', target: 80 },     // 8% of 1000
        { duration: '1m', target: 80 },     // Sustain
        { duration: '30s', target: 0 },     // Ramp down
      ],
    },
    register: {
      executor: 'ramping-vus',
      exec: 'register',
      startVUs: 0,
      stages: [
        { duration: '30s', target: 5 },     // Ramp up
        { duration: '2m', target: 20 },     // 2% of 1000
        { duration: '1m', target: 20 },     // Sustain
        { duration: '30s', target: 0 },     // Ramp down
      ],
    },
  },
  thresholds: {
    'http_req_duration': ['p(95)<100', 'p(99)<200'],
    'errors': ['rate<0.05'],
    'validate_token_duration': ['p(95)<50', 'p(99)<100'],
    'login_duration': ['p(95)<100', 'p(99)<200'],
    'register_duration': ['p(95)<150', 'p(99)<300'],
    'refresh_duration': ['p(95)<80', 'p(99)<150'],
  },
};

const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080';

let userCounter = 0;
const sharedTokens = [];
const sharedRefreshTokens = [];

// Setup: Create base users
export function setup() {
  const users = [];
  const tokens = [];
  const numUsers = 50;

  console.log(`Creating ${numUsers} base users for testing...`);

  for (let i = 0; i < numUsers; i++) {
    const timestamp = Date.now() + i;
    const user = {
      username: `k6mixed_${timestamp}`,
      email: `k6mixed_${timestamp}@example.com`,
      password: 'K6TestPass123!',
    };

    // Register
    const registerRes = http.post(`${BASE_URL}/auth/register`, JSON.stringify(user), {
      headers: { 'Content-Type': 'application/json' },
    });

    if (registerRes.status === 201) {
      // Login to get tokens
      const loginRes = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
        identifier: user.username,
        password: user.password,
      }), {
        headers: { 'Content-Type': 'application/json' },
      });

      if (loginRes.status === 200) {
        const body = JSON.parse(loginRes.body);
        users.push(user);
        tokens.push({
          access: body.access_token,
          refresh: body.refresh_token,
        });
      }
    }

    sleep(0.05);
  }

  console.log(`Setup complete. Created ${users.length} users with tokens.`);
  return { users, tokens };
}

// Scenario 1: Validate Token (70% of traffic)
export function validateToken(data) {
  if (!data || !data.tokens || data.tokens.length === 0) {
    return;
  }

  const token = data.tokens[randomIntBetween(0, data.tokens.length - 1)].access;

  const startTime = Date.now();
  const res = http.get(`${BASE_URL}/auth/me`, {
    headers: { 'Authorization': `Bearer ${token}` },
  });
  const duration = Date.now() - startTime;

  validateTokenTime.add(duration);
  validateTokenCount.add(1);

  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'response time < 50ms': (r) => r.timings.duration < 50,
  });

  errorRate.add(!success);
  sleep(0.1);
}

// Scenario 2: Refresh Token (20% of traffic)
export function refreshToken(data) {
  if (!data || !data.tokens || data.tokens.length === 0) {
    return;
  }

  const tokenPair = data.tokens[randomIntBetween(0, data.tokens.length - 1)];

  const startTime = Date.now();
  const res = http.post(`${BASE_URL}/auth/refresh`, JSON.stringify({
    refresh_token: tokenPair.refresh,
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
  const duration = Date.now() - startTime;

  refreshTime.add(duration);
  refreshCount.add(1);

  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'has new access_token': (r) => {
      try {
        return JSON.parse(r.body).access_token !== undefined;
      } catch {
        return false;
      }
    },
  });

  // Update token if successful
  if (success) {
    const body = JSON.parse(res.body);
    tokenPair.access = body.access_token;
    tokenPair.refresh = body.refresh_token;
  }

  errorRate.add(!success);
  sleep(randomIntBetween(1, 3));
}

// Scenario 3: Login (8% of traffic)
export function login(data) {
  if (!data || !data.users || data.users.length === 0) {
    return;
  }

  const user = data.users[randomIntBetween(0, data.users.length - 1)];

  const startTime = Date.now();
  const res = http.post(`${BASE_URL}/auth/login`, JSON.stringify({
    identifier: user.username,
    password: user.password,
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
  const duration = Date.now() - startTime;

  loginTime.add(duration);
  loginCount.add(1);

  const success = check(res, {
    'status is 200': (r) => r.status === 200,
    'has tokens': (r) => {
      try {
        const body = JSON.parse(r.body);
        return body.access_token && body.refresh_token;
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!success);
  sleep(randomIntBetween(2, 5));
}

// Scenario 4: Register (2% of traffic)
export function register() {
  userCounter++;
  const timestamp = Date.now();
  const uniqueId = `${timestamp}_${userCounter}_${__VU}`;

  const startTime = Date.now();
  const res = http.post(`${BASE_URL}/auth/register`, JSON.stringify({
    username: `k6new_${uniqueId}`,
    email: `k6new_${uniqueId}@example.com`,
    password: 'K6TestPass123!',
  }), {
    headers: { 'Content-Type': 'application/json' },
  });
  const duration = Date.now() - startTime;

  registerTime.add(duration);
  registerCount.add(1);

  const success = check(res, {
    'status is 201': (r) => r.status === 201,
    'has user id': (r) => {
      try {
        return JSON.parse(r.body).id !== undefined;
      } catch {
        return false;
      }
    },
  });

  errorRate.add(!success);
  sleep(randomIntBetween(3, 8));
}

// Summary
export function handleSummary(data) {
  const totalRps = data.metrics.http_reqs.values.rate;
  const errorRate = data.metrics.errors.values.rate * 100;

  const validateAvg = data.metrics.validate_token_duration?.values.avg || 0;
  const validateP99 = data.metrics.validate_token_duration?.values['p(99)'] || 0;
  const loginAvg = data.metrics.login_duration?.values.avg || 0;
  const loginP99 = data.metrics.login_duration?.values['p(99)'] || 0;
  const registerAvg = data.metrics.register_duration?.values.avg || 0;
  const registerP99 = data.metrics.register_duration?.values['p(99)'] || 0;
  const refreshAvg = data.metrics.refresh_duration?.values.avg || 0;
  const refreshP99 = data.metrics.refresh_duration?.values['p(99)'] || 0;

  console.log('\n========================================');
  console.log('🌐 MIXED LOAD TEST RESULTS (REALISTIC)');
  console.log('========================================');
  console.log(`⚡ Total RPS: ${totalRps.toFixed(2)}`);
  console.log(`❌ Error rate: ${errorRate.toFixed(2)}%`);
  console.log('');
  console.log('📊 By Operation:');
  console.log(`  Validate Token: avg ${validateAvg.toFixed(2)}ms | p99 ${validateP99.toFixed(2)}ms`);
  console.log(`  Login:          avg ${loginAvg.toFixed(2)}ms | p99 ${loginP99.toFixed(2)}ms`);
  console.log(`  Register:       avg ${registerAvg.toFixed(2)}ms | p99 ${registerP99.toFixed(2)}ms`);
  console.log(`  Refresh:        avg ${refreshAvg.toFixed(2)}ms | p99 ${refreshP99.toFixed(2)}ms`);
  console.log('========================================\n');

  return {
    'stdout': JSON.stringify(data, null, 2),
  };
}

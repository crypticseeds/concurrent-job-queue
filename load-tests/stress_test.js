// stress_test.js
import http from 'k6/http';
import { sleep, check } from 'k6';
import { Rate } from 'k6/metrics';
import { randomIntBetween } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

// --- Test Configuration ---
export const options = {
  // Load Profile:
  // 1. Ramp up to 50 users in 1 minute
  // 2. Sustain 100 users for 3 minutes (Heavy Load)
  // 3. Ramp down to 0
  stages: [
    { duration: '1m', target: 50 },
    { duration: '3m', target: 100 },
    { duration: '1m', target: 50 },
    { duration: '30s', target: 0 },
  ],
  thresholds: {
    http_req_failed: ['rate<0.01'], // Error rate < 1%
    http_req_duration: ['p(95)<500'], // 95th percentile response time < 500ms
  },
};

// --- Custom Metrics ---
const failedPostRequests = new Rate('failed_post_requests');
const failedGetRequests = new Rate('failed_get_requests');

// State to track tasks for status checks
let createdTaskIds = [];
const MAX_TASK_IDS_TO_TRACK = 2000;

export default function () {
  const baseUrl = 'http://localhost:8080';
  const operationType = randomIntBetween(1, 10);

  // 70% Create Task (High Ingestion)
  if (operationType <= 7) {
    const payload = JSON.stringify({ payload: { data: `stress_data_${__VU}_${Date.now()}` } });
    const res = http.post(`${baseUrl}/tasks`, payload, {
      headers: { 'Content-Type': 'application/json' },
    });

    if (check(res, { 'POST 201': (r) => r.status === 201 })) {
      try {
        const body = JSON.parse(res.body);
        if (body && body.id) {
          createdTaskIds.push(body.id);
          if (createdTaskIds.length > MAX_TASK_IDS_TO_TRACK) createdTaskIds.shift();
        } else {
          // If 201 but body.id is missing, it's a failure
          failedPostRequests.add(1);
        }
      } catch (e) {
        // If JSON.parse fails, it's a failure
        failedPostRequests.add(1);
      }
    } else {
      failedPostRequests.add(1);
    }
    sleep(randomIntBetween(0.5, 2));
  } 
  // 30% Get Task (Observe Processing)
  else {
    if (createdTaskIds.length > 0) {
      const id = createdTaskIds[randomIntBetween(0, createdTaskIds.length - 1)];
      const res = http.get(`${baseUrl}/tasks/${id}`);
      
      if (!check(res, { 'GET 200': (r) => r.status === 200 })) {
        failedGetRequests.add(1);
      }
    }
    sleep(randomIntBetween(1, 4));
  }
}

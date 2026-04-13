import http from "k6/http";
import { check, sleep } from "k6";

// Health endpoint load test.
// Run: make test-load  (or: k6 run ./tests/load/health.js)
// Excluded from make test and make ci — manual only.

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";

export const options = {
  stages: [
    // Smoke: 5 VUs for 10s
    { duration: "10s", target: 5 },
    // Load: ramp to 50 VUs over 30s
    { duration: "30s", target: 50 },
    // Stress: ramp to 100 VUs over 30s
    { duration: "30s", target: 100 },
    // Cool down
    { duration: "10s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<200"],
    http_req_failed: ["rate<0.01"],
  },
};

export default function () {
  const res = http.get(`${BASE_URL}/v1/health`);

  check(res, {
    "status is 200": (r) => r.status === 200,
    "has data field": (r) => JSON.parse(r.body).data !== undefined,
    "has message field": (r) => JSON.parse(r.body).message !== undefined,
    "has status field": (r) => JSON.parse(r.body).status !== undefined,
    "has success field": (r) => JSON.parse(r.body).success === true,
    "response envelope valid": (r) => {
      const body = JSON.parse(r.body);
      return (
        body.status === "success" &&
        body.success === true &&
        body.data !== null &&
        typeof body.data.version === "string"
      );
    },
  });

  sleep(0.1);
}

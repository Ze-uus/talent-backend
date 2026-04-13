import http from "k6/http";
import { check, sleep } from "k6";

// Tracking endpoint load test.
// Run: make test-load-tracking  (or: k6 run ./tests/load/tracking.js)
// Requires a running server with at least one active tracking link.
// Excluded from make test and make ci — manual only.

const BASE_URL = __ENV.BASE_URL || "http://localhost:8080";
const TRACK_TOKEN = __ENV.TRACK_TOKEN || "test-token";

export const options = {
  stages: [
    { duration: "10s", target: 10 },
    { duration: "30s", target: 100 },
    { duration: "30s", target: 200 },
    { duration: "10s", target: 0 },
  ],
  thresholds: {
    http_req_duration: ["p(95)<500"],
    http_req_failed: ["rate<0.05"],
  },
};

export default function () {
  const payload = JSON.stringify({
    event_type: "click",
    idempotency_key: `k6-${__VU}-${__ITER}-${Date.now()}`,
  });

  const params = {
    headers: { "Content-Type": "application/json" },
  };

  const res = http.post(
    `${BASE_URL}/v1/track/${TRACK_TOKEN}`,
    payload,
    params
  );

  check(res, {
    "status is 204 or 404": (r) => r.status === 204 || r.status === 404,
  });

  sleep(0.05);
}

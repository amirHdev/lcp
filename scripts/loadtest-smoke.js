import http from "k6/http";
import { check } from "k6";

export const options = {
  vus: 20,
  duration: "30s",
  thresholds: {
    http_req_failed: ["rate<0.01"],
    http_req_duration: ["p(95)<200"],
  },
};

const baseURL = __ENV.BASE_URL || "http://localhost:8080";
const token = __ENV.TOKEN || "";

export default function () {
  const res = http.get(`${baseURL}/api/v1/lcp/status`, {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  });
  check(res, {
    "status is 200": (r) => r.status === 200,
  });
}

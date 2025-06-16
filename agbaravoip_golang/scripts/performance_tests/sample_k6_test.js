import http from 'k6/http';
import { check, sleep, group } from 'k6';
import { Rate } from 'k6/metrics';
import { encoding } from "k6/encoding"; // For base64 encoding if needed directly in script

// Configuration
const BASE_URL = __ENV.BASE_URL || 'http://localhost:8080/api/v1'; // Set via environment variable
const VU = parseInt(__ENV.VU) || 10; // Virtual Users, ensure it's an integer
const DURATION = __ENV.DURATION || '30s';

// Custom Metrics
let errorRate = new Rate('errors');

// Test Options
export const options = {
  vus: VU,
  duration: DURATION,
  thresholds: {
    'http_req_duration': ['p(95)<500'], // 95% of requests should be below 500ms
    'errors': ['rate<0.1'], // Error rate should be less than 10%
  },
};

// Test Account Credentials (replace with actual test account SIDs and tokens)
// These should ideally be pre-provisioned in the test environment.
const TEST_ACCOUNTS = [
  { sid: __ENV.TEST_ACCOUNT_SID_1 || 'ACtestuser001', token: __ENV.TEST_ACCOUNT_TOKEN_1 || 'testtoken001' },
  // Add more test accounts if needed for scaling user-specific actions by passing more ENV VARS
  // { sid: __ENV.TEST_ACCOUNT_SID_2 || 'ACtestuser002', token: __ENV.TEST_ACCOUNT_TOKEN_2 || 'testtoken002' },
];
let authToken = ''; // Will be fetched by setup function

// Setup function: Authenticate and get a token for subsequent tests
export function setup() {
  if (TEST_ACCOUNTS.length === 0 || !TEST_ACCOUNTS[0].sid || !TEST_ACCOUNTS[0].token || TEST_ACCOUNTS[0].sid === 'ACtestuser001') {
    console.warn('Test account credentials are placeholders or not fully configured via ENV VARS. Some tests might be skipped or use a default placeholder token.');
    // Return a placeholder token if setup cannot complete, allowing some basic tests to run.
    return { token: "PLACEHOLDER_TOKEN_SETUP_FAILED", accountSid: "ACdummySid" };
  }

  const account = TEST_ACCOUNTS[0]; // Using the first test account for a shared token

  // For Basic Auth to get JWT:
  // The k6/encoding module can be used for base64.
  const credentials = `${account.sid}:${account.token}`;
  const encodedCredentials = encoding.b64encode(credentials);
  const basicAuthHeader = `Basic ${encodedCredentials}`;

  console.log(`Using base URL: ${BASE_URL} for setup.`);
  console.log(`Attempting to fetch token for Account SID: ${account.sid}`);

  const loginRes = http.post(`${BASE_URL}/auth/token`, null, {
    headers: { Authorization: basicAuthHeader },
    tags: { name: "AuthTokenGeneration" } // Tag for k6 reports
  });

  if (loginRes.status !== 200) {
    console.error(`Failed to login for setup: ${loginRes.status} Body: ${loginRes.body}`);
    // Depending on strictness, you might want to throw an error or return a placeholder
    // throw new Error(`Failed to login for setup: ${loginRes.status} ${loginRes.body}`);
    return { token: "PLACEHOLDER_TOKEN_LOGIN_FAILED", accountSid: account.sid }; // Allow tests to proceed with placeholder
  }

  const token = loginRes.json('token');
  if (!token) {
    console.error(`Token not found in login response: ${loginRes.body}`);
    return { token: "PLACEHOLDER_TOKEN_NO_TOKEN_IN_RESPONSE", accountSid: account.sid };
  }

  console.log(`Successfully fetched token for Account SID: ${account.sid}`);
  return { token: token, accountSid: account.sid }; // Return data to be used in default function
}


export default function (data) {
  const token = data.token;
  const accountSidToUse = data.accountSid;

  const headers = {
    Authorization: `Bearer ${token}`,
    'Content-Type': 'application/json',
  };

  group('API Health and Basic Endpoints', () => {
    group('Health Check', () => {
      const res = http.get(`${BASE_URL}/health`);
      const success = check(res, {
        'Health check status is 200': (r) => r.status === 200,
      });
      errorRate.add(!success);
      sleep(1);
    });

    if (token && !token.startsWith("PLACEHOLDER_TOKEN_") && accountSidToUse && accountSidToUse !== "ACdummySid") {
        group('List Applications', () => {
            const res = http.get(`${BASE_URL}/accounts/${accountSidToUse}/applications`, { headers });
            const successCheck = check(res, {
                'List Applications status is 200': (r) => r.status === 200,
            });
            errorRate.add(!successCheck);
            if(res.status !== 200) {
                console.warn(`List Applications failed: ${res.status} Body: ${res.body}`);
            }
            sleep(1);
        });
    } else {
        if (__ITER === 0) console.log("Skipping token-dependent 'List Applications' test as token/account SID is placeholder or setup failed.");
    }
  });

  group('Call Origination', () => {
    if (token && !token.startsWith("PLACEHOLDER_TOKEN_") && accountSidToUse && accountSidToUse !== "ACdummySid") {
        const callPayload = JSON.stringify({
            from: '+1555000111', // Use a valid test 'from' number registered to the test account
            to: '+1555000222',   // Use a valid test 'to' number
            answer_url: 'http://example.com/k6_test_answer.xml', // A simple, publicly accessible XML
        });

        const res = http.post(`${BASE_URL}/accounts/${accountSidToUse}/calls`, callPayload, { headers });
        const successCheck = check(res, {
            'Call Origination status is 201': (r) => r.status === 201,
            'Call SID is present': (r) => r.json('sid') !== undefined && String(r.json('sid')).startsWith('CA'),
        });
        errorRate.add(!successCheck);
        if (res.status !== 201) {
            console.warn(`Call Origination Failed: ${res.status} Body: ${res.body}`);
        }
        sleep(2);
    } else {
        if (__ITER === 0) console.log("Skipping token-dependent 'Call Origination' test as token/account SID is placeholder or setup failed.");
    }
  });

  // Example for Admin endpoint (if JWT token has admin role and adminSIDs are configured in AgbaraVOIP)
  // This assumes the token obtained in setup() would have admin role if TEST_ACCOUNT_SID_1 is an admin.
  if (token && !token.startsWith("PLACEHOLDER_TOKEN_") && (accountSidToUse.includes("admin") || TEST_ACCOUNTS[0].sid.includes("admin"))) { // Heuristic for admin
    group('Admin List Gateways', () => {
        const res = http.get(`${BASE_URL}/admin/gateways`, { headers });
        const successCheck = check(res, {
            'Admin List Gateways status is 200': (r) => r.status === 200,
        });
        errorRate.add(!successCheck);
        if(res.status !== 200) {
            console.warn(`Admin List Gateways failed: ${res.status} Body: ${res.body}`);
        }
        sleep(1);
    });
  } else {
    if (__ITER === 0) console.log("Skipping 'Admin List Gateways' test as token/account SID is placeholder, setup failed, or account is not indicative of admin.");
  }
}

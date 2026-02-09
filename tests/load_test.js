import http from 'k6/http';
import { check, group, sleep } from 'k6';
import { Counter } from 'k6/metrics';

// Custom metrics
const successfulLogins = new Counter('successful_logins');
const successfulPosts = new Counter('successful_posts');

// Test configuration
export const options = {
  stages: [
    { duration: '10s', target: 5 },  // Ramp up to 5 users
    { duration: '20s', target: 5 },  // Stay at 5 users
    { duration: '10s', target: 0 },  // Ramp down to 0
  ],
  thresholds: {
    http_req_duration: ['p(95)<1000'], // 95% of requests must complete below 1000ms
    http_req_failed: ['rate<0.01'],   // http errors should be less than 1%
  },
};

const BASE_URL = 'http://localhost:8000/api/v1';

// Helper to generate random string
function randomString(length) {
  const charset = 'abcdefghijklmnopqrstuvwxyz0123456789';
  let res = '';
  for (let i = 0; i < length; i++) {
    res += charset[Math.floor(Math.random() * charset.length)];
  }
  return res;
}

export default function () {
  const username = `testuser_${randomString(8)}`;
  const email = `${username}@example.com`;
  const password = 'password123';
  let authToken = '';
  let userId = '';
  let did = '';

  group('Authentication Flow', function () {
    // 1. Register
    const registerPayload = JSON.stringify({
      username: username,
      email: email,
      password: password,
      display_name: 'Load Test User',
    });

    const registerRes = http.post(`${BASE_URL}/auth/register`, registerPayload, {
      headers: { 'Content-Type': 'application/json' },
    });

    check(registerRes, {
      'register status is 201': (r) => r.status === 201,
      'register has token': (r) => r.json('token') !== undefined,
    });

    if (registerRes.status === 201) {
      authToken = registerRes.json('token');
      // Store user info if needed for later
    }

    sleep(1);

    // 2. Login (sanity check, though we got token from register)
    const loginPayload = JSON.stringify({
      username: username,
      password: password,
    });

    const loginRes = http.post(`${BASE_URL}/auth/login`, loginPayload, {
      headers: { 'Content-Type': 'application/json' },
    });

    check(loginRes, {
      'login status is 200': (r) => r.status === 200,
      'login has token': (r) => r.json('token') !== undefined,
    });

    if (loginRes.status === 200) {
      authToken = loginRes.json('token');
      userId = loginRes.json('user.id');
      did = loginRes.json('user.did');
      successfulLogins.add(1);
    }
  });

  sleep(1);

  if (authToken) {
    const params = {
      headers: {
        'Authorization': `Bearer ${authToken}`,
        'Content-Type': 'application/json',
      },
    };

    group('Content Flow', function () {
      // 3. Create Post
       // k6 doesn't strictly support multipart/form-data well with simple http.post for file uploads 
       // without some workarounds, but we can test JSON post creation logic if supported or just content.
       // IMPORTANT: If the API strictly requires multipart/form-data for posts even without files, 
       // we might need a different approach. Assuming it accepts it, or we use a workaround if needed.
       // Looking at the Go code, it uses `c.FormValue("content")`. This usually implies multipart/form-data or x-www-form-urlencoded.
       // Let's try x-www-form-urlencoded first as it's easier in k6 if no file is needed.

      const postPayload = {
        content: `Load test post content ${randomString(10)}`,
        visibility: 'public',
      };

      const postRes = http.post(`${BASE_URL}/posts`, postPayload, {
        headers: { 
            'Authorization': `Bearer ${authToken}`,
            // k6 adds the correct content-type for object payloads (x-www-form-urlencoded) automatically 
            // OR if we pass a string it assumes JSON.
            // Since the Go handler uses `c.FormValue`, it expects form data.
        },
      });

      check(postRes, {
        'create post status is 201': (r) => r.status === 201,
        'post has id': (r) => r.json('id') !== undefined,
      });

      if (postRes.status === 201) {
        successfulPosts.add(1);
        const postId = postRes.json('id');

        sleep(1);

        // 4. Like passed post (Simulate interaction immediately)
        const likeRes = http.post(`${BASE_URL}/posts/${postId}/like`, null, params);
        check(likeRes, {
            'like post status is 201': (r) => r.status === 201,
        });
      }
    });

    sleep(1);

    group('Feed Retrieval', function () {
      // 5. Get Public Feed
      const feedRes = http.get(`${BASE_URL}/posts/public?limit=10`, params);
      
      check(feedRes, {
        'feed status is 200': (r) => r.status === 200,
        'feed is array': (r) => Array.isArray(r.json()),
      });
    });
  }
}

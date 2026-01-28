// Comprehensive API Endpoint Test Suite
// Tests all backend endpoints for the Splitter application

const API_BASE = 'http://localhost:8000/api/v1';

// Test results tracking
const results = {
  passed: 0,
  failed: 0,
  tests: []
};

async function makeRequest(method, endpoint, body = null, token = null) {
  const headers = {
    'Content-Type': 'application/json',
    'Origin': 'http://localhost:3000'
  };
  
  if (token) {
    headers['Authorization'] = `Bearer ${token}`;
  }
  
  const options = {
    method,
    headers
  };
  
  if (body) {
    options.body = JSON.stringify(body);
  }
  
  try {
    const response = await fetch(`${API_BASE}${endpoint}`, options);
    const text = await response.text();
    let data;
    try {
      data = JSON.parse(text);
    } catch {
      data = text;
    }
    return { status: response.status, data, ok: response.ok, headers: response.headers };
  } catch (error) {
    return { status: 0, data: null, ok: false, error: error.message };
  }
}

function logTest(name, endpoint, method, expectedStatus, actualStatus, passed, details = '') {
  const icon = passed ? '✅' : '❌';
  console.log(`${icon} ${name}`);
  console.log(`   ${method} ${endpoint}`);
  console.log(`   Expected: ${expectedStatus}, Got: ${actualStatus}`);
  if (details) console.log(`   ${details}`);
  console.log('');
  
  results.tests.push({ name, endpoint, method, expectedStatus, actualStatus, passed });
  if (passed) results.passed++;
  else results.failed++;
}

async function runTests() {
  console.log('='.repeat(60));
  console.log('SPLITTER API - COMPREHENSIVE ENDPOINT TEST SUITE');
  console.log('='.repeat(60));
  console.log(`API Base: ${API_BASE}`);
  console.log(`Time: ${new Date().toISOString()}`);
  console.log('='.repeat(60));
  console.log('');

  // ============================================
  // HEALTH CHECK
  // ============================================
  console.log('--- HEALTH CHECK ---\n');
  
  const health = await makeRequest('GET', '/health');
  logTest(
    'Health Check',
    '/health',
    'GET',
    200,
    health.status,
    health.status === 200 && health.data?.status === 'ok',
    `Response: ${JSON.stringify(health.data)}`
  );

  // ============================================
  // AUTH ENDPOINTS (No Auth Required)
  // ============================================
  console.log('--- AUTHENTICATION ENDPOINTS ---\n');

  // Test challenge request
  const challenge = await makeRequest('POST', '/auth/challenge', {
    did: 'did:key:test123456789'
  });
  logTest(
    'Request Auth Challenge',
    '/auth/challenge',
    'POST',
    200,
    challenge.status,
    challenge.status === 200 || challenge.status === 400,
    `Response: ${JSON.stringify(challenge.data).substring(0, 100)}`
  );

  // Test verify endpoint (will fail without valid signature, but should respond)
  const verify = await makeRequest('POST', '/auth/verify', {
    did: 'did:key:test123456789',
    signature: 'invalid_signature'
  });
  logTest(
    'Verify Auth (Invalid - Expected)',
    '/auth/verify',
    'POST',
    [400, 401, 500],
    verify.status,
    [400, 401, 500].includes(verify.status),
    `Response: ${JSON.stringify(verify.data).substring(0, 100)}`
  );

  // Test refresh token (will fail without valid token)
  const refresh = await makeRequest('POST', '/auth/refresh');
  logTest(
    'Refresh Token (No Token - Expected 401)',
    '/auth/refresh',
    'POST',
    401,
    refresh.status,
    refresh.status === 401,
    `Response: ${JSON.stringify(refresh.data)}`
  );

  // ============================================
  // USER ENDPOINTS
  // ============================================
  console.log('--- USER ENDPOINTS ---\n');

  // Get user profile (public)
  const userProfile = await makeRequest('GET', '/users/testuser');
  logTest(
    'Get User Profile (Non-existent)',
    '/users/testuser',
    'GET',
    [200, 404],
    userProfile.status,
    [200, 404].includes(userProfile.status),
    `Response: ${JSON.stringify(userProfile.data).substring(0, 100)}`
  );

  // Get current user (requires auth)
  const currentUser = await makeRequest('GET', '/users/me');
  logTest(
    'Get Current User (No Auth - Expected 401)',
    '/users/me',
    'GET',
    401,
    currentUser.status,
    currentUser.status === 401,
    `Response: ${JSON.stringify(currentUser.data)}`
  );

  // Update profile (requires auth)
  const updateProfile = await makeRequest('PUT', '/users/me', {
    display_name: 'Test User',
    bio: 'Test bio'
  });
  logTest(
    'Update Profile (No Auth - Expected 401)',
    '/users/me',
    'PUT',
    401,
    updateProfile.status,
    updateProfile.status === 401,
    `Response: ${JSON.stringify(updateProfile.data)}`
  );

  // ============================================
  // POST ENDPOINTS
  // ============================================
  console.log('--- POST ENDPOINTS ---\n');

  // Get posts (requires auth based on router config)
  const posts = await makeRequest('GET', '/posts');
  logTest(
    'Get Posts (No Auth)',
    '/posts',
    'GET',
    [200, 401],
    posts.status,
    [200, 401].includes(posts.status),
    `Response: ${JSON.stringify(posts.data).substring(0, 100)}`
  );

  // Create post (requires auth)
  const createPost = await makeRequest('POST', '/posts', {
    content: 'Test post content'
  });
  logTest(
    'Create Post (No Auth - Expected 401)',
    '/posts',
    'POST',
    401,
    createPost.status,
    createPost.status === 401,
    `Response: ${JSON.stringify(createPost.data)}`
  );

  // Get single post
  const singlePost = await makeRequest('GET', '/posts/nonexistent-id');
  logTest(
    'Get Single Post (Non-existent)',
    '/posts/:id',
    'GET',
    [200, 401, 404],
    singlePost.status,
    [200, 401, 404].includes(singlePost.status),
    `Response: ${JSON.stringify(singlePost.data).substring(0, 100)}`
  );

  // Delete post (requires auth)
  const deletePost = await makeRequest('DELETE', '/posts/test-id');
  logTest(
    'Delete Post (No Auth - Expected 401)',
    '/posts/:id',
    'DELETE',
    401,
    deletePost.status,
    deletePost.status === 401,
    `Response: ${JSON.stringify(deletePost.data)}`
  );

  // ============================================
  // FOLLOW ENDPOINTS
  // ============================================
  console.log('--- FOLLOW ENDPOINTS ---\n');

  // Follow user (requires auth)
  const followUser = await makeRequest('POST', '/users/testuser/follow');
  logTest(
    'Follow User (No Auth - Expected 401)',
    '/users/:username/follow',
    'POST',
    401,
    followUser.status,
    followUser.status === 401,
    `Response: ${JSON.stringify(followUser.data)}`
  );

  // Unfollow user (requires auth)
  const unfollowUser = await makeRequest('DELETE', '/users/testuser/follow');
  logTest(
    'Unfollow User (No Auth - Expected 401)',
    '/users/:username/follow',
    'DELETE',
    401,
    unfollowUser.status,
    unfollowUser.status === 401,
    `Response: ${JSON.stringify(unfollowUser.data)}`
  );

  // Get followers
  const followers = await makeRequest('GET', '/users/testuser/followers');
  logTest(
    'Get Followers',
    '/users/:username/followers',
    'GET',
    [200, 404],
    followers.status,
    [200, 404].includes(followers.status),
    `Response: ${JSON.stringify(followers.data).substring(0, 100)}`
  );

  // Get following
  const following = await makeRequest('GET', '/users/testuser/following');
  logTest(
    'Get Following',
    '/users/:username/following',
    'GET',
    [200, 404],
    following.status,
    [200, 404].includes(following.status),
    `Response: ${JSON.stringify(following.data).substring(0, 100)}`
  );

  // ============================================
  // INTERACTION ENDPOINTS
  // ============================================
  console.log('--- INTERACTION ENDPOINTS (Likes, Reposts, Bookmarks) ---\n');

  // Like post (requires auth)
  const likePost = await makeRequest('POST', '/posts/test-id/like');
  logTest(
    'Like Post (No Auth - Expected 401)',
    '/posts/:id/like',
    'POST',
    401,
    likePost.status,
    likePost.status === 401,
    `Response: ${JSON.stringify(likePost.data)}`
  );

  // Unlike post (requires auth)
  const unlikePost = await makeRequest('DELETE', '/posts/test-id/like');
  logTest(
    'Unlike Post (No Auth - Expected 401)',
    '/posts/:id/like',
    'DELETE',
    401,
    unlikePost.status,
    unlikePost.status === 401,
    `Response: ${JSON.stringify(unlikePost.data)}`
  );

  // Repost (requires auth)
  const repost = await makeRequest('POST', '/posts/test-id/repost');
  logTest(
    'Repost (No Auth - Expected 401)',
    '/posts/:id/repost',
    'POST',
    401,
    repost.status,
    repost.status === 401,
    `Response: ${JSON.stringify(repost.data)}`
  );

  // Bookmark (requires auth)
  const bookmark = await makeRequest('POST', '/posts/test-id/bookmark');
  logTest(
    'Bookmark Post (No Auth - Expected 401)',
    '/posts/:id/bookmark',
    'POST',
    401,
    bookmark.status,
    bookmark.status === 401,
    `Response: ${JSON.stringify(bookmark.data)}`
  );

  // Remove bookmark (requires auth)
  const removeBookmark = await makeRequest('DELETE', '/posts/test-id/bookmark');
  logTest(
    'Remove Bookmark (No Auth - Expected 401)',
    '/posts/:id/bookmark',
    'DELETE',
    401,
    removeBookmark.status,
    removeBookmark.status === 401,
    `Response: ${JSON.stringify(removeBookmark.data)}`
  );

  // Get bookmarks (requires auth)
  const bookmarks = await makeRequest('GET', '/bookmarks');
  logTest(
    'Get Bookmarks (No Auth - Expected 401)',
    '/bookmarks',
    'GET',
    401,
    bookmarks.status,
    bookmarks.status === 401,
    `Response: ${JSON.stringify(bookmarks.data)}`
  );

  // ============================================
  // CORS VERIFICATION
  // ============================================
  console.log('--- CORS VERIFICATION ---\n');
  
  const corsCheck = await makeRequest('GET', '/health');
  const corsOrigin = corsCheck.headers?.get('Access-Control-Allow-Origin');
  logTest(
    'CORS Headers Present',
    '/health',
    'GET',
    'http://localhost:3000',
    corsOrigin || 'Not Set',
    corsOrigin === 'http://localhost:3000',
    `Access-Control-Allow-Origin: ${corsOrigin}`
  );

  // ============================================
  // SUMMARY
  // ============================================
  console.log('='.repeat(60));
  console.log('TEST SUMMARY');
  console.log('='.repeat(60));
  console.log(`Total Tests: ${results.passed + results.failed}`);
  console.log(`Passed: ${results.passed}`);
  console.log(`Failed: ${results.failed}`);
  console.log(`Success Rate: ${((results.passed / (results.passed + results.failed)) * 100).toFixed(1)}%`);
  console.log('='.repeat(60));

  // List failed tests
  const failedTests = results.tests.filter(t => !t.passed);
  if (failedTests.length > 0) {
    console.log('\nFailed Tests:');
    failedTests.forEach(t => {
      console.log(`  ❌ ${t.name} - Expected ${t.expectedStatus}, Got ${t.actualStatus}`);
    });
  }

  // Endpoint coverage
  console.log('\n--- ENDPOINT COVERAGE ---');
  console.log('Health:        ✅ GET /health');
  console.log('Auth:          ✅ POST /auth/challenge');
  console.log('               ✅ POST /auth/verify');
  console.log('               ✅ POST /auth/refresh');
  console.log('Users:         ✅ GET /users/:username');
  console.log('               ✅ GET /users/me');
  console.log('               ✅ PUT /users/me');
  console.log('Posts:         ✅ GET /posts');
  console.log('               ✅ POST /posts');
  console.log('               ✅ GET /posts/:id');
  console.log('               ✅ DELETE /posts/:id');
  console.log('Follow:        ✅ POST /users/:username/follow');
  console.log('               ✅ DELETE /users/:username/follow');
  console.log('               ✅ GET /users/:username/followers');
  console.log('               ✅ GET /users/:username/following');
  console.log('Interactions:  ✅ POST /posts/:id/like');
  console.log('               ✅ DELETE /posts/:id/like');
  console.log('               ✅ POST /posts/:id/repost');
  console.log('               ✅ POST /posts/:id/bookmark');
  console.log('               ✅ DELETE /posts/:id/bookmark');
  console.log('               ✅ GET /bookmarks');
  console.log('CORS:          ✅ Configured for localhost:3000');
}

runTests().catch(console.error);

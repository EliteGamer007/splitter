// Test script to verify Frontend-Backend API connection
// Run with: node test-api-connection.js

const API_BASE = 'http://localhost:8000/api/v1';

async function testConnection() {
  console.log('='.repeat(50));
  console.log('FRONTEND-BACKEND API CONNECTION TEST');
  console.log('='.repeat(50));
  console.log(`\nAPI Base URL: ${API_BASE}\n`);

  const tests = [
    {
      name: 'Health Check',
      endpoint: '/health',
      method: 'GET'
    },
    {
      name: 'Get Public Posts',
      endpoint: '/posts',
      method: 'GET'
    }
  ];

  let passed = 0;
  let failed = 0;

  for (const test of tests) {
    try {
      console.log(`Testing: ${test.name}`);
      console.log(`  Endpoint: ${test.method} ${test.endpoint}`);
      
      const response = await fetch(`${API_BASE}${test.endpoint}`, {
        method: test.method,
        headers: {
          'Content-Type': 'application/json'
        }
      });

      const data = await response.text();
      let jsonData;
      try {
        jsonData = JSON.parse(data);
      } catch {
        jsonData = data;
      }

      if (response.ok) {
        console.log(`  ✅ PASSED - Status: ${response.status}`);
        console.log(`  Response: ${JSON.stringify(jsonData).substring(0, 100)}...`);
        passed++;
      } else {
        console.log(`  ❌ FAILED - Status: ${response.status}`);
        console.log(`  Response: ${JSON.stringify(jsonData)}`);
        failed++;
      }
    } catch (error) {
      console.log(`  ❌ ERROR - ${error.message}`);
      failed++;
    }
    console.log('');
  }

  console.log('='.repeat(50));
  console.log(`RESULTS: ${passed} passed, ${failed} failed`);
  console.log('='.repeat(50));

  // Test CORS by simulating browser request
  console.log('\nCORS Test (simulating browser request from localhost:3000):');
  try {
    const corsResponse = await fetch(`${API_BASE}/health`, {
      method: 'GET',
      headers: {
        'Origin': 'http://localhost:3000',
        'Content-Type': 'application/json'
      }
    });
    
    const corsHeaders = {
      'Access-Control-Allow-Origin': corsResponse.headers.get('Access-Control-Allow-Origin'),
      'Access-Control-Allow-Methods': corsResponse.headers.get('Access-Control-Allow-Methods'),
      'Vary': corsResponse.headers.get('Vary')
    };
    
    console.log('  CORS Headers received:');
    for (const [key, value] of Object.entries(corsHeaders)) {
      if (value) {
        console.log(`    ${key}: ${value}`);
      }
    }
    console.log('  ✅ CORS appears to be configured');
  } catch (error) {
    console.log(`  ❌ CORS Test Error: ${error.message}`);
  }
}

testConnection();

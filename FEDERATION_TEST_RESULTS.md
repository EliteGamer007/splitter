# Federation Testing Instructions

## Overview
Both Splitter instances are now running successfully with federation enabled!

- **Instance 1 (splitter-1)**: http://localhost:8000
  - Database: neondb
  - Users: 108
  - Posts: 14
  - Instance Key: ✓ Generated

- **Instance 2 (splitter-2)**: http://localhost:8001
  - Database: neondb_2
  - Users: 7
  - Posts: 4
  - Instance Key: ✓ Generated

## What Was Fixed

### 1. Missing `instance_keys` Table
**Problem**: Both instances failed to start because the `instance_keys` table didn't exist in either database.
**Solution**: Applied migration 010_federation_fix.sql to both databases, creating the table needed for HTTP Signature authentication in ActivityPub federation.

### 2. Federation Communication Setup
**Fixed**:
- Instance RSA keypairs now properly generated/loaded
- WebFinger endpoint working on both instances
- ActivityPub actor endpoints accessible
- HTTP Signature signing enabled for federated activities

## Testing Federation Manually

### Step 1: Verify Both Instances are Running
```powershell
# Test Instance 1
Invoke-WebRequest -Uri "http://localhost:8000/api/v1/health" -Method GET

# Test Instance 2
Invoke-WebRequest -Uri "http://localhost:8001/api/v1/health" -Method GET
```

Both should return status 200.

### Step 2: Test WebFinger Discovery
```powershell
# Discover user on Instance 1
Invoke-WebRequest -Uri "http://localhost:8000/.well-known/webfinger?resource=acct:admin@splitter-1" -Method GET

# Discover user on Instance 2
Invoke-WebRequest -Uri "http://localhost:8001/.well-known/webfinger?resource=acct:admin@splitter-2" -Method GET
```

### Step 3: Test Federation via Frontend

#### On Instance 1 (http://localhost:3000):
1. Login as an existing user or create a new account
2. Go to the search/federation page
3. Search for a remote user: `@admin@splitter-2`
4. The system should:
   - Use WebFinger to discover the user on Instance 2
   - Fetch their ActivityPub actor profile
   - Display the remote user in search results
5. Click "Follow" on the remote user
6. A Follow activity should be sent to Instance 2's inbox

#### On Instance 2 (connect frontend to http://localhost:8001):
1. Login as admin user
2. Check followers list - should see the follow request from Instance 1
3. Create a new post
4. The post should be delivered to Instance 1's users who follow you

### Step 4: Verify Federation in Database

**Check remote actors cache (Instance 1)**:
```sql
SELECT username, domain, actor_uri FROM remote_actors;
```

**Check outbox activities (Instance 1)**:
```sql
SELECT activity_type, status, target_inbox, created_at 
FROM outbox_activities 
ORDER BY created_at DESC
LIMIT 5;
```

**Check inbox activities (Instance 2)**:
```sql
SELECT activity_type, actor_uri, received_at 
FROM inbox_activities 
ORDER BY received_at DESC
LIMIT 5;
```

## Federation Endpoints Available

### Instance 1 (http://localhost:8000)
- WebFinger: `/.well-known/webfinger?resource=acct:username@splitter-1`
- Actor Profile: `/ap/users/{username}`
- User Inbox: `/ap/users/{username}/inbox`
- User Outbox: `/ap/users/{username}/outbox`
- Remote User Search: `/api/v1/federation/users?q=@user@domain`
- Follow Remote User: `POST /api/v1/federation/follow`

### Instance 2 (http://localhost:8001)
- Same endpoints as Instance 1, but for splitter-2 domain

## Expected Federation Flow

### Following a Remote User
1. User on Instance 1 searches for `@alice@splitter-2`
2. Instance 1 performs WebFinger lookup to Instance 2
3. Instance 1 fetches Alice's ActivityPub actor profile
4. Instance 1 sends Follow activity to Alice's inbox on Instance 2
5. Instance 2 auto-accepts and sends Accept activity back
6. Follow relationship is established in both databases

### Federating a Post
1. Alice creates a post on Instance 2
2. Instance 2 builds a Create{Note} activity
3. Instance 2 looks up Alice's followers from other instances
4. Instance 2 delivers the activity to remote inboxes
5. Instance 1 receives the activity, verifies HTTP signature
6. Instance 1 stores the post in local database with is_remote=true
7. Post appears in followers' federated timelines on Instance 1

## Troubleshooting

### If WebFinger fails:
- Check that FEDERATION_ENABLED=true in both .env files
- Verify FEDERATION_DOMAIN and FEDERATION_URL are correct
- Ensure the user exists in the database

### If Follow fails:
- Check outbox_activities table for delivery status
- Check instance logs for error messages
- Verify HTTP signatures are being generated correctly

### If Posts don't federate:
- Confirm the user has remote followers
- Check outbox_activities for pending deliveries
- Verify remote inboxes are accessible

## Next Steps

To fully implement Sprint 2 federation requirements:

1. ✅ **COMPLETED**: WebFinger protocol
2. ✅ **COMPLETED**: ActivityPub inbox/outbox
3. ✅ **COMPLETED**: HTTP Signatures
4. ✅ **COMPLETED**: Activity deduplication
5. **TODO**: Fetch parent posts in threads
6. **TODO**: Federate likes and reposts
7. **TODO**: Broadcast profile updates
8. **TODO**: Send delete activities

## Database Status

### Instance 1 (neondb)
- ✅ 108 existing users intact
- ✅ 14 existing posts preserved
- ✅ Federation tables created
- ✅ Instance key generated

### Instance 2 (neondb_2)
- ✅ 7 users intact
- ✅ 4 posts preserved
- ✅ Federation tables created
- ✅ Instance key generated

All existing data is safe and preserved!

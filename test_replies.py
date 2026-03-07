import requests
import json

BASE = "https://splitter-m0kv.onrender.com/api/v1"

# Login as admin
r = requests.post(f"{BASE}/auth/login", json={"username":"admin","password":"splitteradmin"}, verify=False)
token = r.json()["token"]
headers = {"Authorization": f"Bearer {token}"}

# Get public posts
r = requests.get(f"{BASE}/posts/public?limit=50", headers=headers, verify=False)
data = r.json()
posts = data.get("posts", data) if isinstance(data, dict) else data

print(f"Total posts fetched: {len(posts)}")
for p in posts:
    if p.get("total_reply_count", 0) > 0 or "split" in p.get("content", "").lower():
        pid = p["id"]
        print(f"\nPost ID: {pid}")
        print(f"  Content: {p['content'][:80]}")
        print(f"  Reply count: {p.get('total_reply_count', 0)}")
        
        # Fetch replies
        r2 = requests.get(f"{BASE}/posts/{pid}/replies", headers=headers, verify=False)
        print(f"  Replies API status: {r2.status_code}")
        replies = r2.json()
        print(f"  Replies returned: {len(replies) if isinstance(replies, list) else replies}")
        if isinstance(replies, list):
            for rep in replies[:3]:
                print(f"    - @{rep.get('username','?')}: {rep.get('content','')[:60]}")

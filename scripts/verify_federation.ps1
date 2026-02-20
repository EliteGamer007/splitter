# Federation Verification Script
$ErrorActionPreference = "Stop"

function Get-Token {
    param ($url, $username, $password)
    try {
        $body = @{ username = $username; password = $password } | ConvertTo-Json
        $response = Invoke-RestMethod -Uri "$url/api/v1/auth/login" -Method POST -Body $body -ContentType "application/json" -ErrorAction Stop
        return $response.token
    } catch {
        return $null
    }
}

function Register-User {
    param ($url, $username, $email, $password)
    try {
        $body = @{ username = $username; email = $email; password = $password; display_name = $username } | ConvertTo-Json
        Invoke-RestMethod -Uri "$url/api/v1/auth/register" -Method POST -Body $body -ContentType "application/json" -ErrorAction SilentlyContinue | Out-Null
        Write-Host "Re-registered $username on $url"
    } catch {
        Write-Host "Registration failed or user exists: $_"
    }
}

Write-Host "=== 1. Setup & Login ==="

# Instance 1
$token1 = Get-Token "http://localhost:8000" "alice1" "password123"
if (-not $token1) {
    Write-Host "Registering Alice..."
    Register-User "http://localhost:8000" "alice1" "alice1@splitter-1.test" "password123"
    $token1 = Get-Token "http://localhost:8000" "alice1" "password123"
}
if ($token1) { Write-Host "✅ Alice logged in to Instance 1" } else { Write-Error "Failed to login Alice" }

# Instance 2
$token2 = Get-Token "http://localhost:8001" "bob2" "password123"
if (-not $token2) {
    Write-Host "Registering Bob..."
    Register-User "http://localhost:8001" "bob2" "bob2@splitter-2.test" "password123"
    $token2 = Get-Token "http://localhost:8001" "bob2" "password123"
}
if ($token2) { Write-Host "✅ Bob logged in to Instance 2" } else { Write-Error "Failed to login Bob" }

Write-Host "`n=== 2. Testing WebFinger (Instance 1 resolves Bob on Instance 2) ==="
try {
    $wf = Invoke-RestMethod -Uri "http://localhost:8000/.well-known/webfinger?resource=acct:bob2@splitter-2"
    if ($wf.links[0].href -like "*bob2*") {
        Write-Host "✅ WebFinger success: Resolved bob2@splitter-2 to $($wf.links[0].href)"
    } else {
        Write-Error "❌ WebFinger failed: Unexpected response $wf"
    }
} catch {
    Write-Error "❌ WebFinger request failed: $_"
}

Write-Host "`n=== 3. Alice follows Bob (Cross-Instance) ==="
try {
    $headers = @{ Authorization = "Bearer $token1" }
    $body = @{ handle = "@bob2@splitter-2" } | ConvertTo-Json
    $followResp = Invoke-RestMethod -Uri "http://localhost:8000/api/v1/federation/follow" -Method POST -Headers $headers -Body $body -ContentType "application/json"
    Write-Host "✅ Follow request sent: $($followResp.message)"
    
    # Wait for async processing
    Start-Sleep -Seconds 3
} catch {
    Write-Error "❌ Follow request failed: $_"
}

Write-Host "`n=== 4. Bob creates a post on Instance 2 ==="
$postContent = "Hello from the federated universe! Time: $(Get-Date)"
try {
    $headers2 = @{ Authorization = "Bearer $token2" }
    $postBody = @{ content = $postContent; visibility = "public" } | ConvertTo-Json
    $postResp = Invoke-RestMethod -Uri "http://localhost:8001/api/v1/posts" -Method POST -Headers $headers2 -Body $postBody -ContentType "application/json"
    Write-Host "✅ Bob created post: $($postResp.post.id)"
    
    # Wait for propagation
    Start-Sleep -Seconds 3
} catch {
    Write-Error "❌ Bob failed to post: $_"
}

Write-Host "`n=== 5. Alice checks Federated Timeline on Instance 1 ==="
try {
    $headers1 = @{ Authorization = "Bearer $token1" }
    $timeline = Invoke-RestMethod -Uri "http://localhost:8000/api/v1/federation/timeline" -Headers $headers1
    
    $found = $false
    foreach ($post in $timeline.posts) {
        if ($post.content -eq $postContent) {
            $found = $true
            Write-Host "✅ FOUND: Bob's post appeared in Alice's federated timeline!"
            Write-Host "   - ID: $($post.id)"
            Write-Host "   - IsRemote: $($post.is_remote)"
            Write-Host "   - Author: $($post.username) from $($post.domain)"
            break
        }
    }
    
    if (-not $found) {
        Write-Warning "⚠️ Post not found in timeline yet."
        Write-Host "Recent posts in timeline:"
        $timeline.posts | Select-Object -First 3 | Format-Table content, is_remote, username
    }
} catch {
    Write-Error "❌ Failed to fetch timeline: $_"
}

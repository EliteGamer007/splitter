$ErrorActionPreference = 'Stop'

$ts = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$s1User = "fed1_$ts"
$s2User = "fed2_$ts"
$pw = "Pass@1234"

function JsonBody($obj) {
    return ($obj | ConvertTo-Json -Depth 5 -Compress)
}

# register users
Invoke-RestMethod -Method POST -Uri "http://localhost:8000/api/v1/auth/register" -ContentType "application/json" -Body (JsonBody @{username=$s1User; email="$s1User@test.local"; password=$pw; display_name=$s1User}) | Out-Null
Invoke-RestMethod -Method POST -Uri "http://localhost:8001/api/v1/auth/register" -ContentType "application/json" -Body (JsonBody @{username=$s2User; email="$s2User@test.local"; password=$pw; display_name=$s2User}) | Out-Null

# login
$l1 = Invoke-RestMethod -Method POST -Uri "http://localhost:8000/api/v1/auth/login" -ContentType "application/json" -Body (JsonBody @{username=$s1User; password=$pw})
$l2 = Invoke-RestMethod -Method POST -Uri "http://localhost:8001/api/v1/auth/login" -ContentType "application/json" -Body (JsonBody @{username=$s2User; password=$pw})
$t1 = $l1.token
$t2 = $l2.token

# search remote user on instance1
$search = Invoke-RestMethod -Uri "http://localhost:8000/api/v1/federation/users?q=@$s2User@splitter-2"
$remote = $search.users[0]

# follow remote from instance1
Invoke-RestMethod -Method POST -Uri "http://localhost:8000/api/v1/federation/follow" -Headers @{ Authorization = "Bearer $t1" } -ContentType "application/json" -Body (JsonBody @{ handle = "@$s2User@splitter-2" }) | Out-Null

# create public post on instance2 (multipart)
$postResp = & curl.exe -s -X POST "http://localhost:8001/api/v1/posts" -H "Authorization: Bearer $t2" -F "content=FED_POST_$ts" -F "visibility=public"
if (-not $postResp) {
    throw "Failed to create post on instance2"
}

Start-Sleep -Seconds 4

# verify federated timeline on instance1 has remote post
$timeline = Invoke-RestMethod -Uri "http://localhost:8000/api/v1/federation/timeline"
$remotePost = $timeline.posts | Where-Object { $_.content -eq "FED_POST_$ts" }
$timelineOk = $null -ne $remotePost

# verify home feed on instance1 has followed remote post
$feed = Invoke-RestMethod -Uri "http://localhost:8000/api/v1/posts/feed" -Headers @{ Authorization = "Bearer $t1" }
$feedPost = $feed | Where-Object { $_.content -eq "FED_POST_$ts" }
$feedOk = $null -ne $feedPost

# send DM from instance1 to remote user
Invoke-RestMethod -Method POST -Uri "http://localhost:8000/api/v1/messages/send" -Headers @{ Authorization = "Bearer $t1" } -ContentType "application/json" -Body (JsonBody @{ recipient_id = $remote.id; content = "FED_DM_$ts" }) | Out-Null

Start-Sleep -Seconds 4

# verify DM arrived in instance2
$threads2 = Invoke-RestMethod -Uri "http://localhost:8001/api/v1/messages/threads" -Headers @{ Authorization = "Bearer $t2" }
$dmFound = $false
foreach ($th in $threads2.threads) {
    $msgs = Invoke-RestMethod -Uri ("http://localhost:8001/api/v1/messages/threads/{0}" -f $th.id) -Headers @{ Authorization = "Bearer $t2" }
    if ($msgs.messages | Where-Object { $_.content -eq "FED_DM_$ts" }) {
        $dmFound = $true
        break
    }
}

[PSCustomObject]@{
    s1_user = $s1User
    s2_user = $s2User
    remote_user_id_on_s1 = $remote.id
    timeline_remote_post_visible = $timelineOk
    home_feed_remote_post_visible = $feedOk
    dm_cross_instance_visible = $dmFound
} | ConvertTo-Json -Depth 4

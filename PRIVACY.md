# Privacy and data handling

Splitter is a college team project. This page describes the repaired web client and the hosted portfolio configuration as of 3 November 2026. An independently operated instance may have different settings.

## Account and social data

The API stores account details such as username, email, bcrypt password hashes, public signing/encryption keys and profile settings. It also stores posts, media, replies, follows, audiences, reports and messaging metadata needed for the app. Public profile responses omit email and private moderation-request details; authenticated owners can access their account details.

Public posts and profiles can be read by other people and external servers. Followers-only and circle content uses audience checks, including its replies and media. Federation can deliver content to other administrators' servers; those operators control their own copies and retention.

## Browser keys and messages

The current web client encrypts direct messages before sending them and omits their plaintext from encrypted send/edit payloads. The service retains ciphertext, wrapped message keys and routing/delivery metadata. Legacy clients and historical records may use different formats, so use the current encrypted flow for confidential messages.

Private signing and message-encryption keys stay in browser storage, scoped to the account and API origin. Recovery exports use a passphrase and validate imported keys. Keep an encrypted backup before clearing browser storage or changing devices. A password reset cannot reconstruct a lost message-encryption private key.

Logout clears the session, private caches and queued sends. Account-scoped key material is retained locally to avoid losing access to historical messages; manage that material on devices you share.

## Hosting, analytics and logs

The hosted frontend uses Vercel Analytics for site measurement. Vercel, Render and Neon also process operational data needed to host and secure their services. The API request logger can include IP addresses, request paths and response metadata. Passwords and provider credentials belong in private configuration, not source code or diagnostic output.

The app does not display advertising in its feed. Hosting-provider data handling and retention are governed by those providers' settings and policies; the project does not promise zero tracking or zero operational logging.

## Expiry, deletion and backups

Expired stories stop being served and are cleaned up opportunistically. Posts and other records may use soft deletion, and operational backups can retain earlier copies. Remote federation peers may retain content they have already received. Deletion on the primary instance cannot guarantee deletion of every external copy.

An encrypted deployment backup was taken before the November schema upgrade. Hosting recovery history and storage limits depend on the selected provider plan.

## Administration

Administrators can moderate accounts and content through role-guarded tools. Operators should protect deployment secrets, use explicit migrations, review access and keep private recovery material secure. Use the instance's support or administrator contact when you need account assistance; do not publish private keys, passwords or recovery exports in issues or chat.

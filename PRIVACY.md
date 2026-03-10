# Privacy & Data Ethics

Splitter is built with a "Privacy by Design" philosophy. We believe that users should own their identity and have full control over their data.

## Data Minimization

Splitter follows the principle of data minimization. We only collect and store the data necessary to provide a functional social experience.

- **No Tracking**: We do not use third-party trackers, analytics, or advertising scripts.
- **No IP Logging**: By default, Splitter instances are configured not to log full IP addresses in application logs (unless required for emergency DDoS mitigation).
- **Pseudonymous Identity**: Users are identified by DIDs, not real names or phone numbers.

## Security & Encryption

### End-to-End Encryption (E2EE)
All direct messages (DMs) in Splitter are end-to-end encrypted.
- **Client-Side Encryption**: Messages are encrypted on the sender's device before being sent to the server.
- **No Server Access**: The server only stores encrypted "blobs" (ciphertext). Instance administrators cannot read your private messages.
- **Perfect Forward Secrecy**: We are working towards implementing double-ratchet algorithms to provide forward secrecy for all conversations.

### Decentralized Identity
Your identity is tied to your cryptographic keypair, not the server.
- **Self-Sovereign**: You can move your identity across different servers (instances) without losing your social graph.
- **Anti-Censorship**: No single instance owner can "delete" your identity from the entire network; they can only moderate your access to their specific server.

## Data Persistence & Federation

Users should be aware of how data behaves in a federated network:

1. **Deletion Latency**: When you delete a post, your server sends a `Delete` activity to all servers that received it. However, we cannot force remote servers to comply (though standard ActivityPub implementations do).
2. **Public Content**: Posts marked as "Public" are visible to anyone on the web and may be indexed by search engines or archived by third parties.
3. **Instance Selection**: Your privacy also depends on the instance you choose. Read the privacy policy of your specific instance administrator before signing up.

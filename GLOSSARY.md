# Glossary of Terms

This document defines the key concepts and terminology used throughout the Splitter project.

## Decentralized Identity (DID)

- **DID (Decentralized Identifier)**: A globally unique identifier that does not require a centralized registration authority. In Splitter, we primarily use `did:key`.
- **Keypair**: A set of two related cryptographic keys: a **Private Key** (kept secret by the user) and a **Public Key** (shared with the network).
- **did:key**: A specific DID method where the identifier is derived directly from the public key itself.
- **Challenge-Response**: An authentication mechanism where the server sends a random nonce (challenge) and the client must sign it with their private key (response) to prove identity.

## Federation & ActivityPub

- **Fediverse**: The network of federated social media servers that communicate using open protocols.
- **ActivityPub**: The decentralized social networking protocol used for server-to-server and client-to-server communication.
- **Actor**: An entity in ActivityPub (usually a person or a bot) that can perform activities.
- **Inbox**: The endpoint where an Actor receives incoming activities from other servers.
- **Outbox**: The endpoint where an Actor publishes their activities to be distributed to others.
- **Shared Inbox**: A single inbox on an instance used to receive messages destined for multiple users on that same instance, optimizing network traffic.
- **WebFinger**: A protocol used to discover information about people or entities by their email-like addresses (e.g., `@user@domain.com`).

## Messaging & Security

- **E2EE (End-to-End Encryption)**: A system of communication where only the communicating users can read the messages.
- **ECDH (Elliptic Curve Diffie-Hellman)**: A key agreement protocol that allows two parties to establish a shared secret over an insecure channel.
- **AES-GCM**: Advanced Encryption Standard in Galois/Counter Mode, used for high-performance authenticated encryption.
- **JWT (JSON Web Token)**: A compact, URL-safe means of representing claims to be transferred between two parties.
- **JWT Rotation**: A security practice where a new JWT is issued periodically to limit the window of opportunity for a stolen token.

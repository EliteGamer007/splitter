# Security Policy

## Supported Versions

The following versions of Splitter are currently supported with security updates:

| Version | Supported          |
| ------- | ------------------ |
| 1.0.x   | ✅ Yes             |
| < 1.0   | ❌ No              |

## Reporting a Vulnerability

We take the security of Splitter seriously. If you believe you have found a security vulnerability, please report it to us responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

Please send security reports to: `security@splitter.social` (placeholder)

### What to include in your report:
- A description of the vulnerability.
- Steps to reproduce the issue (proof of concept).
- Potential impact of the vulnerability.
- Any suggested mitigations.

### Our Commitment:
- We will acknowledge receipt of your report within 48 hours.
- We will provide a timeline for fixes and keep you updated on progress.
- We will credit you for the discovery (unless you prefer to remain anonymous) once the fix is public.

## Security Architecture Overview

Splitter employs several layers of security:
1. **Decentralized Identity (DID)**: Client-side key generation via Ed25519/ECDSA.
2. **End-to-End Encryption (E2EE)**: ECDH + AES-GCM for direct messaging.
3. **HTTP Signatures**: All federated traffic is signed using instance-level and user-level keys.
4. **JWT Auth**: Secure, short-lived session management.

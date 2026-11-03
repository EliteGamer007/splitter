# Splitter demo and interview walkthrough

Splitter is a college team project: a full-stack federated social network with browser-managed identity and encrypted messaging. Use the [live frontend](https://splitter-social.vercel.app) for the repaired production release deployed on 3 November 2026.

## A five-minute walkthrough

1. Open the landing page and explain the purpose: independent social instances with user-controlled identity. Show the peach and dark themes and the mobile navigation.
2. Sign in, publish a short post with a hashtag, like it, open its thread and add a reply. Refresh the thread to show persistence. Open the profile and hashtag views.
3. With a second account, demonstrate a locked profile: request a follow, accept it from the owner, then show the followers-only post. Mention circle audiences and bookmarks.
4. Open a DM, send a message and show it in the conversation. Explain that browser encryption keeps message plaintext out of encrypted API writes; recovery keys must be backed up using the Security page.
5. Open the instance/federation view and explain the one-primary deployment. Demonstrate a second peer only if it is explicitly configured and available. Show admin/moderation with a suitably privileged account; ordinary accounts correctly see an access restriction.

Stories, media uploads, recovery exports and the optional AI bot are useful follow-up demonstrations. Configure a provider before promising an AI reply. Before a presentation, open the app once to allow a sleeping free backend to start; do not run a periodic keep-alive service.

## What makes the project worth discussing

- End-to-end delivery across a React UI, Go REST API, PostgreSQL persistence and three hosting providers.
- Shared authorization rules for public, follower and circle content, including media and replies.
- The difference between Ed25519 signing identity and P-256 message encryption keys, plus account-scoped key persistence and encrypted recovery.
- Signed federation requests, audience-aware delivery and defenses against unsafe remote URLs.
- Operating within a small budget: a four-connection pool, pagination, compressed durable images, request deadlines and no idle polling workers.
- Explicit, repeatable migrations and tests that cannot accidentally target the production database.

## Resume wording

Select statements that match **your actual contribution**, and name this as a team project. Do not claim sole ownership or invent user counts, latency improvements or production uptime.

- Collaborated on Splitter, a federated social platform built with Next.js, Go/Echo and PostgreSQL, featuring threaded posts, private audiences and encrypted messaging.
- Implemented [your specific subsystem], including [the concrete interaction or data flow you owned], and integrated it with the team's frontend/backend.
- Improved [your specific reliability or deployment contribution] using [the relevant technique], with validation through [the checks you actually performed].

## Handoff

The frontend and backend have separate Git repositories and release histories. Keep both revisions together when presenting a repaired release. See [REPAIR_STATUS.md](REPAIR_STATUS.md) for what is local, what was checked and what still requires a production rollout.

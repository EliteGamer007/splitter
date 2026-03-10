"""
Splitter Bot — Populates the app with AI-generated posts from 100 bot accounts.
Uses Google Gemini API (free tier) for content generation.

Usage:
  - Set GEMINI_API_KEY, SPLITTER_INSTANCE_1_URL, SPLITTER_INSTANCE_2_URL as env vars.
  - Run: python populate.py
  - Or let GitHub Actions run it on a schedule.

Modes (set via BOT_MODE env var):
  - "seed"   : Bulk-create ~500 posts per run (5 per bot x 100 bots). Run multiple times.
  - "drip"   : Post one per bot, every invocation (for scheduled cron runs).
"""

import os
import sys
import time
import random
import requests

# ---------------------------------------------------------------------------
# Configuration from environment
# ---------------------------------------------------------------------------
GEMINI_API_KEY = os.environ.get("GEMINI_API_KEY", "")
INSTANCE_1_URL = os.environ.get("SPLITTER_INSTANCE_1_URL", "https://splitter-m0kv.onrender.com")
INSTANCE_2_URL = os.environ.get("SPLITTER_INSTANCE_2_URL", "https://splitter-2.onrender.com")
BOT_MODE = os.environ.get("BOT_MODE", "drip")  # "seed" or "drip"
BOT_PASSWORD = os.environ.get("BOT_PASSWORD", "BotPass#2026!")

# Seed mode: 5 posts per bot x 100 bots = 500 posts per run. Run 8x for 4000.
SEED_POSTS_PER_BOT = 5
# Delay between posts (seconds) — keeps us under Gemini free-tier 15 RPM
POST_DELAY = 5
# Extra delay after each Gemini API call (seconds)
GEMINI_COOLDOWN = 4

# ---------------------------------------------------------------------------
# 25 topic categories — each has 4 bot personas = 100 bots total.
# ---------------------------------------------------------------------------
TOPIC_TEMPLATES = [
    # ── Splitter / Federation / Decentralization (~55-60% of posts) ──
    {
        "category": "splitter_hype",
        "hashtags": ["#Splitter", "#Federation", "#FutureOfSocial", "#NoAlgorithms"],
        "prompt": "Write a short casual social media post (1-3 sentences) from someone who just discovered Splitter and is blown away by the decentralized, no-algorithm feed. Vary your tone randomly between excited, chill, nerdy, or sarcastic. Include 1-2 hashtags like {hashtags}. Never start with 'Just'."
    },
    {
        "category": "splitter_vs_bigtech",
        "hashtags": ["#Splitter", "#ByeTwitter", "#Decentralization", "#Web3"],
        "prompt": "Write a short casual social media post (1-2 sentences) comparing Splitter positively to mainstream social media (Twitter, Instagram, etc). Be witty or thoughtful. Include 1-2 hashtags like {hashtags}. Don't be preachy."
    },
    {
        "category": "federation_technical",
        "hashtags": ["#Federation", "#OpenProtocol", "#Splitter", "#SelfHosting"],
        "prompt": "Write a short casual social media post (1-3 sentences) from a developer or techie excited about federation, open protocols, or self-hosting social media. Sound knowledgeable but approachable. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "splitter_feature_love",
        "hashtags": ["#SplitterApp", "#Trending", "#E2E", "#Federation"],
        "prompt": "Write a short casual social media post (1-2 sentences) praising a specific Splitter feature like trending hashtags, E2E encrypted DMs, the clean UI, or the AI bot. Pick ONE feature and be specific. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "crypto_privacy",
        "hashtags": ["#Crypto", "#Privacy", "#DeFi", "#Decentralization", "#Web3"],
        "prompt": "Write a short casual social media post (1-3 sentences) about data privacy, owning your own data, or why centralized platforms are problematic. Sound passionate but not conspiratorial. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "splitter_community",
        "hashtags": ["#Splitter", "#Community", "#Fediverse", "#NewHere"],
        "prompt": "Write a short casual social media post (1-2 sentences) about how friendly or refreshing the Splitter community feels compared to toxic mainstream platforms. Be genuine. Include 1-2 hashtags like {hashtags}."
    },
    # ── Generic / Everyday (~40-45% of posts) ──
    {
        "category": "dev_life",
        "hashtags": ["#Programming", "#AI", "#Coding", "#Developer", "#BuildInPublic"],
        "prompt": "Write a short casual social media post (1-3 sentences) about a coding win, debugging frustration, learning a new language, or shipping a side project. Be relatable. Vary between triumphant, exhausted, or humorous. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "daily_life",
        "hashtags": ["#WeekendVibes", "#Coffee", "#DailyGrind", "#MorningRoutine", "#Chill"],
        "prompt": "Write a short casual social media post (1-3 sentences) about your morning, coffee, weather, small daily wins, or weekend plans. Sound like a real person. Vary between happy, tired, reflective, or sarcastic. Include 1 hashtag like {hashtags}."
    },
    {
        "category": "food_and_hobbies",
        "hashtags": ["#Foodie", "#Cooking", "#Gaming", "#Music", "#Movies"],
        "prompt": "Write a short casual social media post (1-2 sentences) about food you ate, a game you played, a song stuck in your head, or a movie you watched. Be specific with a detail. Include 1 hashtag like {hashtags}."
    },
    {
        "category": "random_thoughts",
        "hashtags": ["#RandomThoughts", "#Mood", "#ShowerThoughts", "#Unpopular"],
        "prompt": "Write a unique shower thought, hot take, funny observation, or mini-rant in 1-2 sentences that sounds like something someone would actually tweet. Be original and avoid cliches. Include 1 hashtag like {hashtags}."
    }
]

NAMES_AND_BIOS = [
    ("alex_smith", "Alex", "Just a regular person trying to figure out this federated stuff."),
    ("sarah_j", "Sarah J", "Coffee addict. Code tinkerer."),
    ("mike_r", "Mike", "Living life one post at a time. Loving Splitter."),
    ("emily_x", "Emily", "Digital nomad. Exploring the decentralized web."),
    ("chris_p", "Chris", "Here for the tech and the memes."),
    ("jessica_t", "Jess", "Posting thoughts into the void."),
    ("david_m", "David", "Tech enthusiast. Early adopter."),
    ("laura_k", "Laura", "Trying out Splitter! It looks awesome."),
    ("dan_w", "Dan", "Avid gamer. Crypto curious."),
    ("rachel_b", "Rachel B", "Art, design, and random musings."),
    ("tom_h", "Tom", "Software engineer by day, gamer by night."),
    ("amy_l", "Amy", "Just joined Splitter! Say hi."),
    ("james_c", "James", "Decentralize everything."),
    ("olivia_s", "Olivia", "Thoughts, opinions, and too much coffee."),
    ("kevin_d", "Kevin", "Building the future of social media."),
    ("megan_f", "Megan", "Exploring new platforms. Splitter is neat."),
    ("brian_g", "Brian", "Web3 and beyond."),
    ("hannah_v", "Hannah", "Just vibing on Splitter."),
    ("ryan_p", "Ryan", "Code, coffee, sleep, repeat."),
    ("chloe_m", "Chloe", "Hello world! This is my first post here.")
]

def generate_bot_profiles():
    """Generate 100 bot profiles dynamically."""
    profiles = []
    import random
    for i in range(100):
        base_name_tuple = NAMES_AND_BIOS[i % len(NAMES_AND_BIOS)]
        suffix = str(random.randint(10, 999)) if i >= len(NAMES_AND_BIOS) else ""
        username = base_name_tuple[0] + suffix
        display = base_name_tuple[1] + ("" if not suffix else f" {suffix}")
        bio = base_name_tuple[2]
        instance = 1 if i % 2 == 0 else 2
        
        # Pick a random template and resolve the hashtags in the prompt string
        template = random.choice(TOPIC_TEMPLATES)
        chosen_hashtags = " ".join(random.sample(template["hashtags"], min(2, len(template["hashtags"]))))
        resolved_prompt = template["prompt"].replace("{hashtags}", chosen_hashtags)
        
        profiles.append({
            "username": username,
            "email": f"{username}@bot.local",
            "display_name": display,
            "bio": bio,
            "instance": instance,
            "category": template["category"],
            "prompt": resolved_prompt
        })
    return profiles


BOT_PROFILES = generate_bot_profiles()

# ---------------------------------------------------------------------------
# Gemini API helper (uses REST directly — no SDK needed)
# ---------------------------------------------------------------------------
GEMINI_URL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-1.5-flash-latest:generateContent"


def generate_post_text(prompt: str, max_retries: int = 3) -> str:
    """Call Gemini API to generate a single post, with retry on 429."""
    if not GEMINI_API_KEY:
        return random_fallback_post()

    headers = {"Content-Type": "application/json"}
    payload = {
        "contents": [{"parts": [{"text": prompt}]}],
        "generationConfig": {
            "temperature": 1.2,
            "topP": 0.95,
            "topK": 40,
            "maxOutputTokens": 150,
        },
    }
    for attempt in range(max_retries):
        try:
            resp = requests.post(
                f"{GEMINI_URL}?key={GEMINI_API_KEY}",
                headers=headers,
                json=payload,
                timeout=30,
            )
            if resp.status_code == 429:
                wait = 30 * (attempt + 1)  # 30s, 60s, 90s backoff
                print(f"  [Gemini 429] Rate limited — waiting {wait}s (attempt {attempt+1}/{max_retries})")
                time.sleep(wait)
                continue
            resp.raise_for_status()
            data = resp.json()
            text = data["candidates"][0]["content"]["parts"][0]["text"]
            text = text.strip().strip('"').strip("'")
            # Throttle: wait after each successful Gemini call
            time.sleep(GEMINI_COOLDOWN)
            return text
        except Exception as e:
            if attempt < max_retries - 1 and "429" in str(e):
                time.sleep(30 * (attempt + 1))
                continue
            print(f"  [Gemini Error] {e} — using fallback")
            return random_fallback_post()
    return random_fallback_post()


FALLBACK_TEMPLATES = [
    "Just another day in the grind! What's everyone working on? #Trending #Splitter",
    "Hot take: the best code is the code you don't write. #Tech #Coding #DevLife",
    "Anyone else feel like time moves differently on weekends? #Relatable #Mood",
    "New week, new goals. Let's get it! #Motivation #Hustle #Growth",
    "The internet never sleeps and neither do I apparently. #LOL #InternetCulture",
    "Just discovered something amazing. Can't wait to share more! #Trending #News",
    "Coffee count today: 4. Productivity count: debatable. #DevLife #Coding #Coffee",
    "Weekend plans: absolutely nothing and I'm excited about it. #Mood #Relatable #Vibes",
    "Learning something new every day. That's the whole point right? #Learning #Growth",
    "This community is growing fast! Love seeing all the activity here. #Splitter #Community",
    "Sometimes you just need to log off and touch grass. #MentalHealth #SelfCare #Mood",
    "You ever solve a bug and feel like a superhero? That was me today. #DevLife #Coding",
    "My favorite algorithm is the one that gets food delivered to my door. #Foodie #Tech",
    "Reading a good book is the original streaming. No buffering required. #Books #Reading",
    "Nature doesn't need an update. It just works. #Nature #Environment #GoGreen",
]


def random_fallback_post() -> str:
    """If Gemini is unavailable, generate a simple templated post."""
    return random.choice(FALLBACK_TEMPLATES)


# ---------------------------------------------------------------------------
# Splitter API helpers
# ---------------------------------------------------------------------------

def get_instance_url(instance_num: int) -> str:
    return INSTANCE_1_URL if instance_num == 1 else INSTANCE_2_URL


def register_bot(bot: dict) -> str | None:
    """Register a bot account. Returns JWT token or None."""
    url = f"{get_instance_url(bot['instance'])}/api/v1/auth/register"
    payload = {
        "username": bot["username"],
        "email": bot["email"],
        "password": BOT_PASSWORD,
        "display_name": bot["display_name"],
        "bio": bot["bio"],
    }
    try:
        resp = requests.post(url, json=payload, timeout=30)
        if resp.status_code == 201:
            print(f"  [+] Registered {bot['username']} on instance {bot['instance']}")
            return resp.json().get("token")
        elif resp.status_code == 409 or "already" in resp.text.lower():
            return login_bot(bot)
        else:
            print(f"  [!] Register failed for {bot['username']}: {resp.status_code} {resp.text[:200]}")
            return login_bot(bot)
    except Exception as e:
        print(f"  [!] Register error for {bot['username']}: {e}")
        return None


def login_bot(bot: dict) -> str | None:
    """Login a bot account. Returns JWT token or None."""
    url = f"{get_instance_url(bot['instance'])}/api/v1/auth/login"
    payload = {"username": bot["username"], "password": BOT_PASSWORD}
    try:
        resp = requests.post(url, json=payload, timeout=30)
        if resp.status_code == 200:
            return resp.json().get("token")
        else:
            print(f"  [!] Login failed for {bot['username']}: {resp.status_code} {resp.text[:200]}")
            return None
    except Exception as e:
        print(f"  [!] Login error for {bot['username']}: {e}")
        return None


def create_post(bot: dict, token: str, content: str) -> bool:
    """Create a post using multipart/form-data. Returns True on success."""
    url = f"{get_instance_url(bot['instance'])}/api/v1/posts"
    headers = {"Authorization": f"Bearer {token}"}
    # Use 'files' param to force multipart/form-data encoding (required by Echo)
    multipart_fields = {
        "content": (None, content),
        "visibility": (None, "public"),
    }
    try:
        resp = requests.post(url, headers=headers, files=multipart_fields, timeout=30)
        if resp.status_code in (200, 201):
            return True
        else:
            print(f"  [!] Post failed for {bot['username']}: {resp.status_code} {resp.text[:200]}")
            return False
    except Exception as e:
        print(f"  [!] Post error for {bot['username']}: {e}")
        return False


# ---------------------------------------------------------------------------
# Main logic
# ---------------------------------------------------------------------------
# Dynamic prompt builder — picks a fresh template each call for variety
# ---------------------------------------------------------------------------
TONE_MODIFIERS = [
    "Be enthusiastic.", "Be chill and laid-back.", "Be slightly sarcastic.",
    "Be thoughtful and reflective.", "Be funny.", "Use gen-z slang.",
    "Be straightforward.", "Be nerdy.", "Keep it mysterious.",
    "Sound tired but happy.", "Be optimistic.", "Be philosophical.",
]

def fresh_prompt():
    """Build a unique prompt each time by picking a random template + random tone."""
    template = random.choice(TOPIC_TEMPLATES)
    chosen_tags = " ".join(random.sample(template["hashtags"], min(2, len(template["hashtags"]))))
    base = template["prompt"].replace("{hashtags}", chosen_tags)
    tone = random.choice(TONE_MODIFIERS)
    return f"{base} {tone} Do NOT start with 'Just'. Be unique — never repeat common phrases."

# ---------------------------------------------------------------------------

def authenticate_all_bots() -> dict:
    """Register/login all bots and return {username: token} map."""
    tokens = {}
    for bot in BOT_PROFILES:
        token = register_bot(bot)
        if token:
            tokens[bot["username"]] = token
        else:
            print(f"  [SKIP] Could not authenticate {bot['username']}")
        time.sleep(0.5)
    return tokens


def run_seed_mode(tokens: dict):
    """Bulk-create posts: SEED_POSTS_PER_BOT per bot. 5 x 100 = 500 per run."""
    total = 0
    failed = 0
    target = SEED_POSTS_PER_BOT * len(tokens)
    print(f"\n{'='*60}")
    print(f"SEED MODE: Generating ~{target} posts across {len(tokens)} bots")
    print(f"({SEED_POSTS_PER_BOT} posts/bot x {len(tokens)} bots)")
    print(f"{'='*60}\n")

    for round_num in range(SEED_POSTS_PER_BOT):
        print(f"\n--- Round {round_num+1}/{SEED_POSTS_PER_BOT} ---")
        shuffled = list(BOT_PROFILES)
        random.shuffle(shuffled)
        for i, bot in enumerate(shuffled):
            token = tokens.get(bot["username"])
            if not token:
                continue

            content = generate_post_text(fresh_prompt())
            if not content:
                failed += 1
                continue

            success = create_post(bot, token, content)
            if success:
                total += 1
                if total % 25 == 0 or total <= 5:
                    print(f"  [{total}/{target}] @{bot['username']}: {content[:80]}...")
            else:
                failed += 1

            time.sleep(POST_DELAY)

        # Cooldown between rounds to let rate limits reset
        if round_num < SEED_POSTS_PER_BOT - 1:
            print(f"  Round {round_num+1} done. Cooling down 30s...")
            time.sleep(30)

    print(f"\n{'='*60}")
    print(f"SEED COMPLETE: {total} posts created, {failed} failures")
    print(f"{'='*60}")


def run_drip_mode(tokens: dict):
    """Post once per bot — designed for cron/scheduled invocations."""
    total = 0
    failed = 0
    shuffled = list(BOT_PROFILES)
    random.shuffle(shuffled)
    
    # Process only 15 bots per drip run to ensure it finishes well under 15 minutes without overlapping
    selected_bots = shuffled[:15]

    print(f"\nDRIP MODE: Posting for a random subset of {len(selected_bots)} bots (out of {len(tokens)})\n")

    for bot in selected_bots:
        token = tokens.get(bot["username"])
        if not token:
            continue

        content = generate_post_text(fresh_prompt())
        if not content:
            failed += 1
            continue

        success = create_post(bot, token, content)
        if success:
            total += 1
            print(f"  @{bot['username']}: {content[:80]}...")
        else:
            failed += 1

        time.sleep(POST_DELAY)

    print(f"\nDRIP COMPLETE: {total} posts created, {failed} failures")


def main():
    if not GEMINI_API_KEY:
        print("[WARNING] GEMINI_API_KEY not set — using fallback templates (no AI generation)")

    print(f"Splitter Bot Populator — Mode: {BOT_MODE}")
    print(f"Instance 1: {INSTANCE_1_URL}")
    print(f"Instance 2: {INSTANCE_2_URL}")
    print(f"Total bots: {len(BOT_PROFILES)}")
    print()

    tokens = authenticate_all_bots()
    print(f"\nAuthenticated {len(tokens)}/{len(BOT_PROFILES)} bots\n")

    if not tokens:
        print("ERROR: No bots could be authenticated. Exiting.")
        sys.exit(1)

    if BOT_MODE == "seed":
        run_seed_mode(tokens)
    else:
        run_drip_mode(tokens)


if __name__ == "__main__":
    main()

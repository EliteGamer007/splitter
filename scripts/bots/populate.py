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
    {
        "category": "splitter_evangelist",
        "hashtags": ["#Splitter", "#Federation", "#Web3", "#FutureOfSocial"],
        "prompt": "Write a short casual social media post (1-3 sentences) praising Splitter, the new federated app we are on. Mention decentralization or no algorithms. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "splitter_support",
        "hashtags": ["#SplitterApp", "#TechSupport", "#Federation"],
        "prompt": "Write a short casual social media post (1-2 sentences) about how smooth the Splitter UI is, or seeing live trending hashtags here. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "crypto_decentralization",
        "hashtags": ["#Crypto", "#Blockchain", "#DeFi", "#Decentralization"],
        "prompt": "Write a short casual social media post (1-3 sentences) about decentralization and taking back control of data. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "generic_tech",
        "hashtags": ["#Programming", "#AI", "#Coding", "#Developer"],
        "prompt": "Write a short casual social media post (1-3 sentences) about coding, learning a new technology, or building software. Don't mention specific apps. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "generic_life_1",
        "hashtags": ["#WeekendVibes", "#Coffee", "#DailyGrind"],
        "prompt": "Write a short casual social media post (1-3 sentences) about enjoying a coffee, morning routines, or weekend plans. Sound like a regular person posting an update. Include 1 hashtag like {hashtags}."
    },
    {
        "category": "generic_life_2",
        "hashtags": ["#Foodie", "#Cooking", "#LunchBreak"],
        "prompt": "Write a short casual social media post (1-2 sentences) about trying a new recipe, eating good food, or a lunch break. Sound like a regular person. Include 1 hashtag like {hashtags}."
    },
    {
        "category": "generic_entertainment",
        "hashtags": ["#Gaming", "#Movies", "#NowWatching", "#Music"],
        "prompt": "Write a short casual social media post (1-3 sentences) about watching a good movie, listening to music, or playing a video game. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "generic_thoughts",
        "hashtags": ["#RandomThoughts", "#Mood", "#JustThinking"],
        "prompt": "Write a short casual social media post (1-2 sentences) sharing a random shower thought, minor observation about life, or a sudden realization. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "generic_nature",
        "hashtags": ["#Nature", "#Outdoors", "#Walking", "#Sunlight"],
        "prompt": "Write a short casual social media post (1-2 sentences) about going for a walk, enjoying the weather, or spending time outdoors. Include 1-2 hashtags like {hashtags}."
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
    """Generate bot profiles by mixing up random names and bios."""
    profiles = []
    import random
    
    # We want 100 bots generated dynamically
    for i in range(100):
        base_name_tuple = NAMES_AND_BIOS[i % len(NAMES_AND_BIOS)]
        # For duplicates, add a random number suffix to username and display
        suffix = str(random.randint(10, 999)) if i >= len(NAMES_AND_BIOS) else ""
        
        username = base_name_tuple[0] + suffix
        display = base_name_tuple[1] + ("" if not suffix else f" {suffix}")
        bio = base_name_tuple[2]
        
        # Distribute them across instance 1 and 2
        instance = 1 if i % 2 == 0 else 2
        
        profiles.append({
            "username": username,
            "email": f"{username}@bot.local",
            "display_name": display,
            "bio": bio,
            "instance": instance
        })
    return profiles



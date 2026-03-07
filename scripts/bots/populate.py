"""
Splitter Bot — Populates the app with AI-generated posts from multiple bot accounts.
Uses Google Gemini API (free tier) for content generation.

Usage:
  - Set GEMINI_API_KEY, SPLITTER_INSTANCE_1_URL, SPLITTER_INSTANCE_2_URL as env vars.
  - Run: python populate.py
  - Or let GitHub Actions run it on a schedule.

Modes (set via BOT_MODE env var):
  - "seed"   : Bulk-create ~4000 posts across all bots (one-time baseline).
  - "drip"   : Post one per bot, every invocation (for scheduled cron runs).
"""

import os
import sys
import time
import random
import requests
import json

# ---------------------------------------------------------------------------
# Configuration from environment
# ---------------------------------------------------------------------------
GEMINI_API_KEY = os.environ.get("GEMINI_API_KEY", "")
INSTANCE_1_URL = os.environ.get("SPLITTER_INSTANCE_1_URL", "https://splitter-m0kv.onrender.com")
INSTANCE_2_URL = os.environ.get("SPLITTER_INSTANCE_2_URL", "https://splitter-2.onrender.com")
BOT_MODE = os.environ.get("BOT_MODE", "drip")  # "seed" or "drip"
BOT_PASSWORD = os.environ.get("BOT_PASSWORD", "BotPass#2026!")

# How many posts per bot in seed mode (total = SEED_POSTS_PER_BOT * number_of_bots)
SEED_POSTS_PER_BOT = 200
# Delay between posts (seconds) — stay gentle on free-tier Render
POST_DELAY = 12  # ~12s between posts

# ---------------------------------------------------------------------------
# Bot personas — each bot has a theme, hashtags, and prompt style
# ---------------------------------------------------------------------------
BOT_PROFILES = [
    {
        "username": "techie_tara",
        "email": "techie_tara@splitter.bot",
        "display_name": "Tara | Tech & Code",
        "bio": "Full-stack dev. Open source enthusiast. Coffee-powered.",
        "instance": 1,
        "topics": ["programming", "webdev", "opensource", "AI", "python", "golang", "javascript", "rust"],
        "hashtags": ["#Tech", "#Coding", "#WebDev", "#OpenSource", "#AI", "#Python", "#GoLang", "#JavaScript", "#DevLife", "#Programming"],
        "prompt": "Write a short casual social media post (1-3 sentences) about software development, programming, or tech news. Include 2-3 relevant hashtags from this list: #Tech #Coding #WebDev #OpenSource #AI #Python #GoLang #JavaScript #DevLife #Programming. Sound like a real person, not a corporate account. Vary the tone — sometimes excited, sometimes reflective, sometimes asking a question."
    },
    {
        "username": "sports_sam",
        "email": "sports_sam@splitter.bot",
        "display_name": "Sam | Sports Fan",
        "bio": "Living for game day. Football, basketball, F1. Hot takes guaranteed.",
        "instance": 1,
        "topics": ["football", "basketball", "F1", "cricket", "sports"],
        "hashtags": ["#Sports", "#Football", "#Basketball", "#F1", "#Cricket", "#GameDay", "#FitnessGoals", "#Athlete", "#TeamWork", "#Championship"],
        "prompt": "Write a short casual social media post (1-3 sentences) about sports — could be football, basketball, F1, cricket, or fitness. Include 2-3 relevant hashtags from this list: #Sports #Football #Basketball #F1 #Cricket #GameDay #FitnessGoals #Athlete #TeamWork #Championship. Sound like a passionate fan, not a news reporter."
    },
    {
        "username": "foodie_fiona",
        "email": "foodie_fiona@splitter.bot",
        "display_name": "Fiona | Food & Travel",
        "bio": "Eating my way through the world one city at a time.",
        "instance": 1,
        "topics": ["food", "cooking", "travel", "restaurants", "recipes"],
        "hashtags": ["#Foodie", "#Cooking", "#Travel", "#Recipe", "#Yummy", "#Restaurant", "#FoodPhotography", "#Wanderlust", "#StreetFood", "#HomeCooking"],
        "prompt": "Write a short casual social media post (1-3 sentences) about food, cooking, a restaurant experience, or travel. Include 2-3 relevant hashtags from this list: #Foodie #Cooking #Travel #Recipe #Yummy #Restaurant #FoodPhotography #Wanderlust #StreetFood #HomeCooking. Sound like a real food lover sharing their experience."
    },
    {
        "username": "music_mike",
        "email": "music_mike@splitter.bot",
        "display_name": "Mike | Music & Vibes",
        "bio": "If it has a beat, I'm in. DJ on weekends.",
        "instance": 2,
        "topics": ["music", "concerts", "DJing", "playlists", "albums"],
        "hashtags": ["#Music", "#NowPlaying", "#Concert", "#DJ", "#Playlist", "#NewMusic", "#HipHop", "#Rock", "#EDM", "#Vibes"],
        "prompt": "Write a short casual social media post (1-3 sentences) about music — could be a new song, a concert, DJing, or a playlist recommendation. Include 2-3 relevant hashtags from this list: #Music #NowPlaying #Concert #DJ #Playlist #NewMusic #HipHop #Rock #EDM #Vibes. Sound like a real music fan."
    },
    {
        "username": "gamer_grace",
        "email": "gamer_grace@splitter.bot",
        "display_name": "Grace | Gamer",
        "bio": "PC gamer. RPG addict. Streaming sometimes.",
        "instance": 2,
        "topics": ["gaming", "PC", "RPG", "streaming", "esports"],
        "hashtags": ["#Gaming", "#PCGaming", "#RPG", "#Streaming", "#Esports", "#GamerLife", "#IndieGames", "#PlayStation", "#Nintendo", "#GameReview"],
        "prompt": "Write a short casual social media post (1-3 sentences) about video games — could be a game review, a gaming moment, streaming, or esports. Include 2-3 relevant hashtags from this list: #Gaming #PCGaming #RPG #Streaming #Esports #GamerLife #IndieGames #PlayStation #Nintendo #GameReview. Sound like a real gamer."
    },
    {
        "username": "fitness_frank",
        "email": "fitness_frank@splitter.bot",
        "display_name": "Frank | Fitness & Health",
        "bio": "Gym 6 days a week. Sharing what works.",
        "instance": 1,
        "topics": ["fitness", "gym", "health", "nutrition", "workout"],
        "hashtags": ["#Fitness", "#Gym", "#Health", "#Workout", "#Nutrition", "#GymLife", "#HealthyLiving", "#Gains", "#Cardio", "#MealPrep"],
        "prompt": "Write a short casual social media post (1-3 sentences) about fitness, gym workouts, nutrition, or healthy living. Include 2-3 relevant hashtags from this list: #Fitness #Gym #Health #Workout #Nutrition #GymLife #HealthyLiving #Gains #Cardio #MealPrep. Sound like a real gym-goer sharing tips or experiences."
    },
    {
        "username": "art_anna",
        "email": "art_anna@splitter.bot",
        "display_name": "Anna | Art & Design",
        "bio": "Digital artist. Illustrator. Color is my language.",
        "instance": 2,
        "topics": ["art", "design", "illustration", "digital art", "creativity"],
        "hashtags": ["#Art", "#Design", "#Illustration", "#DigitalArt", "#Creative", "#ArtLife", "#Drawing", "#GraphicDesign", "#Sketch", "#Aesthetic"],
        "prompt": "Write a short casual social media post (1-3 sentences) about art, digital illustration, design, or creativity. Include 2-3 relevant hashtags from this list: #Art #Design #Illustration #DigitalArt #Creative #ArtLife #Drawing #GraphicDesign #Sketch #Aesthetic. Sound like a real artist sharing their process or thoughts."
    },
    {
        "username": "news_nick",
        "email": "news_nick@splitter.bot",
        "display_name": "Nick | News & Opinions",
        "bio": "Following the world so you don't have to. Hot takes daily.",
        "instance": 1,
        "topics": ["news", "politics", "economy", "world", "opinion"],
        "hashtags": ["#News", "#Breaking", "#WorldNews", "#Opinion", "#Economy", "#Politics", "#Trending", "#Discussion", "#Today", "#HotTake"],
        "prompt": "Write a short casual social media post (1-3 sentences) sharing a fictional but realistic-sounding opinion about current events, economics, or world affairs. Include 2-3 relevant hashtags from this list: #News #Breaking #WorldNews #Opinion #Economy #Politics #Trending #Discussion #Today #HotTake. Sound like a regular person sharing their take, not a news anchor."
    },
    {
        "username": "movie_maria",
        "email": "movie_maria@splitter.bot",
        "display_name": "Maria | Movies & TV",
        "bio": "Binge-watcher. Film critic in my own head.",
        "instance": 2,
        "topics": ["movies", "TV shows", "Netflix", "cinema", "reviews"],
        "hashtags": ["#Movies", "#TVShows", "#Netflix", "#Cinema", "#FilmReview", "#Binge", "#Streaming", "#Hollywood", "#SciFi", "#Drama"],
        "prompt": "Write a short casual social media post (1-3 sentences) about movies, TV shows, streaming, or cinema. Include 2-3 relevant hashtags from this list: #Movies #TVShows #Netflix #Cinema #FilmReview #Binge #Streaming #Hollywood #SciFi #Drama. Sound like someone who just finished watching something and wants to talk about it."
    },
    {
        "username": "science_sara",
        "email": "science_sara@splitter.bot",
        "display_name": "Sara | Science & Space",
        "bio": "Astrophysics grad. Space nerd. Making science fun.",
        "instance": 1,
        "topics": ["science", "space", "physics", "biology", "research"],
        "hashtags": ["#Science", "#Space", "#Physics", "#Biology", "#Research", "#NASA", "#STEM", "#ScienceFacts", "#Universe", "#Innovation"],
        "prompt": "Write a short casual social media post (1-3 sentences) about science, space exploration, physics, biology, or a cool research finding. Include 2-3 relevant hashtags from this list: #Science #Space #Physics #Biology #Research #NASA #STEM #ScienceFacts #Universe #Innovation. Sound enthusiastic and accessible, like a science communicator."
    },
    {
        "username": "crypto_carl",
        "email": "crypto_carl@splitter.bot",
        "display_name": "Carl | Crypto & Finance",
        "bio": "DeFi maximalist. Not financial advice.",
        "instance": 2,
        "topics": ["crypto", "blockchain", "finance", "investing", "DeFi"],
        "hashtags": ["#Crypto", "#Bitcoin", "#Blockchain", "#DeFi", "#Finance", "#Investing", "#Web3", "#Ethereum", "#Trading", "#HODL"],
        "prompt": "Write a short casual social media post (1-3 sentences) about cryptocurrency, blockchain, DeFi, or personal finance. Include 2-3 relevant hashtags from this list: #Crypto #Bitcoin #Blockchain #DeFi #Finance #Investing #Web3 #Ethereum #Trading #HODL. Sound like a regular crypto enthusiast, not a shill."
    },
    {
        "username": "book_betty",
        "email": "book_betty@splitter.bot",
        "display_name": "Betty | Books & Writing",
        "bio": "Reader. Writer. Library card collector.",
        "instance": 1,
        "topics": ["books", "reading", "writing", "literature", "fiction"],
        "hashtags": ["#Books", "#Reading", "#Writing", "#BookReview", "#Fiction", "#Literature", "#Bookworm", "#AmReading", "#AuthorLife", "#Library"],
        "prompt": "Write a short casual social media post (1-3 sentences) about books, reading, writing, or literature. Include 2-3 relevant hashtags from this list: #Books #Reading #Writing #BookReview #Fiction #Literature #Bookworm #AmReading #AuthorLife #Library. Sound like someone who genuinely loves books."
    },
    {
        "username": "pet_paul",
        "email": "pet_paul@splitter.bot",
        "display_name": "Paul | Pets & Animals",
        "bio": "Dog dad x3. Cat tolerator. Wildlife photographer.",
        "instance": 2,
        "topics": ["pets", "dogs", "cats", "animals", "wildlife"],
        "hashtags": ["#Pets", "#DogsOfSplitter", "#CatsOfSplitter", "#Animals", "#Wildlife", "#PetLife", "#DogLover", "#CatLover", "#Cute", "#Adopt"],
        "prompt": "Write a short casual social media post (1-3 sentences) about pets, dogs, cats, animals, or wildlife. Include 2-3 relevant hashtags from this list: #Pets #DogsOfSplitter #CatsOfSplitter #Animals #Wildlife #PetLife #DogLover #CatLover #Cute #Adopt. Sound like someone who adores their pets."
    },
    {
        "username": "startup_steve",
        "email": "startup_steve@splitter.bot",
        "display_name": "Steve | Startups & Hustle",
        "bio": "Serial entrepreneur. Building in public. Shipping fast.",
        "instance": 1,
        "topics": ["startups", "entrepreneurship", "business", "SaaS", "productivity"],
        "hashtags": ["#Startup", "#Entrepreneur", "#Business", "#SaaS", "#Productivity", "#BuildInPublic", "#Hustle", "#Growth", "#Founder", "#MVP"],
        "prompt": "Write a short casual social media post (1-3 sentences) about startups, entrepreneurship, building products, or productivity tips. Include 2-3 relevant hashtags from this list: #Startup #Entrepreneur #Business #SaaS #Productivity #BuildInPublic #Hustle #Growth #Founder #MVP. Sound like a real founder sharing their journey."
    },
    {
        "username": "meme_lord_max",
        "email": "meme_lord_max@splitter.bot",
        "display_name": "Max | Memes & Humor",
        "bio": "Professional time waster. Internet historian.",
        "instance": 2,
        "topics": ["memes", "humor", "internet", "jokes", "viral"],
        "hashtags": ["#Memes", "#Funny", "#Humor", "#LOL", "#Viral", "#InternetCulture", "#Relatable", "#Jokes", "#Mood", "#TooReal"],
        "prompt": "Write a short funny or sarcastic social media post (1-2 sentences) — could be an observation about daily life, internet culture, or a relatable situation. Include 2-3 relevant hashtags from this list: #Memes #Funny #Humor #LOL #Viral #InternetCulture #Relatable #Jokes #Mood #TooReal. Be genuinely witty, not cringe."
    },
    {
        "username": "eco_emma",
        "email": "eco_emma@splitter.bot",
        "display_name": "Emma | Environment",
        "bio": "Climate activist. Zero-waste journey. Trees > everything.",
        "instance": 1,
        "topics": ["environment", "climate", "sustainability", "nature", "green"],
        "hashtags": ["#Environment", "#Climate", "#Sustainability", "#Nature", "#GoGreen", "#EcoFriendly", "#ClimateAction", "#ZeroWaste", "#Planet", "#Trees"],
        "prompt": "Write a short casual social media post (1-3 sentences) about environmental issues, sustainability, nature, or eco-friendly living. Include 2-3 relevant hashtags from this list: #Environment #Climate #Sustainability #Nature #GoGreen #EcoFriendly #ClimateAction #ZeroWaste #Planet #Trees. Sound passionate but not preachy."
    },
    {
        "username": "photo_pete",
        "email": "photo_pete@splitter.bot",
        "display_name": "Pete | Photography",
        "bio": "Chasing golden hour. Street & landscape photographer.",
        "instance": 2,
        "topics": ["photography", "cameras", "landscape", "street photography", "editing"],
        "hashtags": ["#Photography", "#PhotoOfTheDay", "#Landscape", "#StreetPhotography", "#Camera", "#GoldenHour", "#Lightroom", "#NaturePhotography", "#Portrait", "#Shutterbug"],
        "prompt": "Write a short casual social media post (1-3 sentences) about photography, cameras, editing photos, or a shooting experience. Include 2-3 relevant hashtags from this list: #Photography #PhotoOfTheDay #Landscape #StreetPhotography #Camera #GoldenHour #Lightroom #NaturePhotography #Portrait #Shutterbug. Sound like a photographer sharing their work or thoughts."
    },
    {
        "username": "edu_elena",
        "email": "edu_elena@splitter.bot",
        "display_name": "Elena | Education",
        "bio": "Teacher by day. Lifelong learner always.",
        "instance": 1,
        "topics": ["education", "learning", "teaching", "students", "online courses"],
        "hashtags": ["#Education", "#Learning", "#Teaching", "#Students", "#EdTech", "#OnlineLearning", "#Knowledge", "#StudyTips", "#Teacher", "#MOOC"],
        "prompt": "Write a short casual social media post (1-3 sentences) about education, teaching, learning, or study tips. Include 2-3 relevant hashtags from this list: #Education #Learning #Teaching #Students #EdTech #OnlineLearning #Knowledge #StudyTips #Teacher #MOOC. Sound like a real educator or student."
    },
    {
        "username": "fashion_faye",
        "email": "fashion_faye@splitter.bot",
        "display_name": "Faye | Fashion & Style",
        "bio": "Thrift queen. Street style diary.",
        "instance": 2,
        "topics": ["fashion", "style", "outfits", "thrifting", "trends"],
        "hashtags": ["#Fashion", "#Style", "#OOTD", "#Thrifting", "#Trends", "#StreetStyle", "#Outfit", "#FashionInspo", "#Wardrobe", "#Vintage"],
        "prompt": "Write a short casual social media post (1-3 sentences) about fashion, style, outfits, thrifting, or trends. Include 2-3 relevant hashtags from this list: #Fashion #Style #OOTD #Thrifting #Trends #StreetStyle #Outfit #FashionInspo #Wardrobe #Vintage. Sound like someone sharing their personal style."
    },
    {
        "username": "diy_dana",
        "email": "diy_dana@splitter.bot",
        "display_name": "Dana | DIY & Crafts",
        "bio": "If I can build it, I will. Woodworking + electronics.",
        "instance": 1,
        "topics": ["DIY", "crafts", "woodworking", "electronics", "maker"],
        "hashtags": ["#DIY", "#Crafts", "#Maker", "#Woodworking", "#Electronics", "#Handmade", "#BuildStuff", "#Upcycle", "#Workshop", "#Create"],
        "prompt": "Write a short casual social media post (1-3 sentences) about a DIY project, crafting, woodworking, electronics tinkering, or making things. Include 2-3 relevant hashtags from this list: #DIY #Crafts #Maker #Woodworking #Electronics #Handmade #BuildStuff #Upcycle #Workshop #Create. Sound like someone excited about building things."
    },
]

# ---------------------------------------------------------------------------
# Gemini API helper (uses REST directly — no SDK needed)
# ---------------------------------------------------------------------------
GEMINI_URL = "https://generativelanguage.googleapis.com/v1beta/models/gemini-2.0-flash:generateContent"


def generate_post_text(prompt: str) -> str:
    """Call Gemini API to generate a single post."""
    if not GEMINI_API_KEY:
        # Fallback: pick a random pre-written post if no API key
        return random_fallback_post(prompt)

    headers = {"Content-Type": "application/json"}
    payload = {
        "contents": [{"parts": [{"text": prompt}]}],
        "generationConfig": {
            "temperature": 1.0,
            "maxOutputTokens": 150,
        },
    }
    try:
        resp = requests.post(
            f"{GEMINI_URL}?key={GEMINI_API_KEY}",
            headers=headers,
            json=payload,
            timeout=30,
        )
        resp.raise_for_status()
        data = resp.json()
        text = data["candidates"][0]["content"]["parts"][0]["text"]
        # Clean up: remove surrounding quotes if Gemini wraps it
        text = text.strip().strip('"').strip("'")
        return text
    except Exception as e:
        print(f"  [Gemini Error] {e} — using fallback")
        return random_fallback_post(prompt)


def random_fallback_post(prompt: str) -> str:
    """If Gemini is unavailable, generate a simple templated post."""
    templates = [
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
    ]
    return random.choice(templates)


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
            # Already exists — just login
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
    form_data = {"content": content, "visibility": "public"}
    try:
        resp = requests.post(url, headers=headers, data=form_data, timeout=30)
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

def authenticate_all_bots() -> dict:
    """Register/login all bots and return {username: token} map."""
    tokens = {}
    for bot in BOT_PROFILES:
        token = register_bot(bot)
        if token:
            tokens[bot["username"]] = token
        else:
            print(f"  [SKIP] Could not authenticate {bot['username']}")
        time.sleep(1)  # Don't hammer the API during auth
    return tokens


def run_seed_mode(tokens: dict):
    """Bulk-create posts: SEED_POSTS_PER_BOT per bot."""
    total = 0
    failed = 0
    target = SEED_POSTS_PER_BOT * len(tokens)
    print(f"\n{'='*60}")
    print(f"SEED MODE: Generating ~{target} posts across {len(tokens)} bots")
    print(f"{'='*60}\n")

    for round_num in range(SEED_POSTS_PER_BOT):
        # Shuffle bots each round for variety
        shuffled = list(BOT_PROFILES)
        random.shuffle(shuffled)
        for bot in shuffled:
            token = tokens.get(bot["username"])
            if not token:
                continue

            content = generate_post_text(bot["prompt"])
            if not content:
                failed += 1
                continue

            success = create_post(bot, token, content)
            if success:
                total += 1
                print(f"  [{total}/{target}] @{bot['username']}: {content[:80]}...")
            else:
                failed += 1

            time.sleep(POST_DELAY)

    print(f"\n{'='*60}")
    print(f"SEED COMPLETE: {total} posts created, {failed} failures")
    print(f"{'='*60}")


def run_drip_mode(tokens: dict):
    """Post once per bot — designed for cron/scheduled invocations."""
    total = 0
    failed = 0
    shuffled = list(BOT_PROFILES)
    random.shuffle(shuffled)

    print(f"\nDRIP MODE: Posting 1 per bot ({len(tokens)} bots)\n")

    for bot in shuffled:
        token = tokens.get(bot["username"])
        if not token:
            continue

        content = generate_post_text(bot["prompt"])
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
    print(f"Bots: {len(BOT_PROFILES)}")
    print()

    # Step 1: Authenticate all bots
    tokens = authenticate_all_bots()
    print(f"\nAuthenticated {len(tokens)}/{len(BOT_PROFILES)} bots\n")

    if not tokens:
        print("ERROR: No bots could be authenticated. Exiting.")
        sys.exit(1)

    # Step 2: Run the selected mode
    if BOT_MODE == "seed":
        run_seed_mode(tokens)
    else:
        run_drip_mode(tokens)


if __name__ == "__main__":
    main()

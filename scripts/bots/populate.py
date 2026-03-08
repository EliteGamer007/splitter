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
        "category": "tech",
        "hashtags": "#Tech #Coding #WebDev #OpenSource #AI #Python #GoLang #JavaScript #DevLife #Programming",
        "prompt": "Write a short casual social media post (1-3 sentences) about software development, programming, or tech news. Include 2-3 hashtags from: {hashtags}. Sound like a real person.",
        "names": [
            ("techie_tara", "Tara", "Full-stack dev. Open source enthusiast. Coffee-powered."),
            ("dev_derek", "Derek", "Backend engineer. API design nerd. Rust curious."),
            ("code_clara", "Clara", "Self-taught coder. Python lover. Building side projects."),
            ("hack_hugo", "Hugo", "Compiler enthusiast. Linux sysadmin. Opinions on everything."),
        ],
    },
    {
        "category": "sports",
        "hashtags": "#Sports #Football #Basketball #F1 #Cricket #GameDay #Athlete #TeamWork #Championship #Fitness",
        "prompt": "Write a short casual social media post (1-3 sentences) about sports — football, basketball, F1, cricket, or fitness. Include 2-3 hashtags from: {hashtags}. Sound like a passionate fan.",
        "names": [
            ("sports_sam", "Sam", "Living for game day. Hot takes guaranteed."),
            ("goal_guru", "Guru", "Premier League obsessed. Fantasy football addict."),
            ("hoop_hana", "Hana", "Basketball is life. WNBA supporter."),
            ("lap_liam", "Liam", "F1 addict. Data-driven race analysis."),
        ],
    },
    {
        "category": "food",
        "hashtags": "#Foodie #Cooking #Travel #Recipe #Yummy #Restaurant #FoodPhotography #Wanderlust #StreetFood #HomeCooking",
        "prompt": "Write a short casual social media post (1-3 sentences) about food, cooking, or restaurants. Include 2-3 hashtags from: {hashtags}. Sound like a food lover.",
        "names": [
            ("foodie_fiona", "Fiona", "Eating my way through the world one city at a time."),
            ("chef_chen", "Chen", "Wok master. Noodle whisperer."),
            ("bake_bella", "Bella", "Sourdough starter named Kevin. Pastry perfectionist."),
            ("taco_tony", "Tony", "Tacos are a personality trait. Food truck hunter."),
        ],
    },
    {
        "category": "music",
        "hashtags": "#Music #NowPlaying #Concert #DJ #Playlist #NewMusic #HipHop #Rock #EDM #Vibes",
        "prompt": "Write a short casual social media post (1-3 sentences) about music, concerts, or playlists. Include 2-3 hashtags from: {hashtags}. Sound like a real music fan.",
        "names": [
            ("music_mike", "Mike", "If it has a beat I'm in. DJ on weekends."),
            ("vinyl_vera", "Vera", "Record collector. Analog soul in a digital world."),
            ("beat_bobby", "Bobby", "Producing beats in my bedroom. Lo-fi is a lifestyle."),
            ("riff_rosa", "Rosa", "Guitar player. 90s grunge enthusiast. Volume to 11."),
        ],
    },
    {
        "category": "gaming",
        "hashtags": "#Gaming #PCGaming #RPG #Streaming #Esports #GamerLife #IndieGames #PlayStation #Nintendo #GameReview",
        "prompt": "Write a short casual social media post (1-3 sentences) about video games, streaming, or esports. Include 2-3 hashtags from: {hashtags}. Sound like a real gamer.",
        "names": [
            ("gamer_grace", "Grace", "PC gamer. RPG addict. Streaming sometimes."),
            ("pixel_pat", "Pat", "Retro games forever. SNES > everything."),
            ("stream_stella", "Stella", "Twitch affiliate. Horror game specialist."),
            ("fps_felix", "Felix", "Ranked grinder. Aim training daily."),
        ],
    },
    {
        "category": "fitness",
        "hashtags": "#Fitness #Gym #Health #Workout #Nutrition #GymLife #HealthyLiving #Gains #Cardio #MealPrep",
        "prompt": "Write a short casual social media post (1-3 sentences) about fitness, workouts, or nutrition. Include 2-3 hashtags from: {hashtags}. Sound like a gym-goer.",
        "names": [
            ("fitness_frank", "Frank", "Gym 6 days a week. Sharing what works."),
            ("run_riley", "Riley", "Marathon finisher x3. Chasing PRs."),
            ("yoga_yuki", "Yuki", "Yoga teacher. Breathwork advocate. Inner peace dealer."),
            ("lift_luna", "Luna", "Deadlift PR chaser. Strong is beautiful."),
        ],
    },
    {
        "category": "art",
        "hashtags": "#Art #Design #Illustration #DigitalArt #Creative #ArtLife #Drawing #GraphicDesign #Sketch #Aesthetic",
        "prompt": "Write a short casual social media post (1-3 sentences) about art, illustration, or design. Include 2-3 hashtags from: {hashtags}. Sound like a real artist.",
        "names": [
            ("art_anna", "Anna", "Digital artist. Color is my language."),
            ("sketch_suki", "Suki", "Pen and ink daily. Urban sketching addict."),
            ("ux_uma", "Uma", "Designing interfaces. Obsessed with whitespace."),
            ("paint_pablo", "Pablo", "Oil on canvas. Impressionism with a modern twist."),
        ],
    },
    {
        "category": "news",
        "hashtags": "#News #Breaking #WorldNews #Opinion #Economy #Politics #Trending #Discussion #Today #HotTake",
        "prompt": "Write a short casual social media post (1-3 sentences) sharing a fictional opinion about current events or economics. Include 2-3 hashtags from: {hashtags}. Sound like a regular person.",
        "names": [
            ("news_nick", "Nick", "Following the world so you don't have to."),
            ("take_tina", "Tina", "Opinions nobody asked for. You're welcome."),
            ("pulse_priya", "Priya", "Geopolitics nerd. Coffee and headlines."),
            ("econ_eli", "Eli", "Supply and demand explain everything."),
        ],
    },
    {
        "category": "movies",
        "hashtags": "#Movies #TVShows #Netflix #Cinema #FilmReview #Binge #Streaming #Hollywood #SciFi #Drama",
        "prompt": "Write a short casual social media post (1-3 sentences) about movies or TV shows. Include 2-3 hashtags from: {hashtags}. Sound like someone who just watched something.",
        "names": [
            ("movie_maria", "Maria", "Binge-watcher. Film critic in my own head."),
            ("screen_scott", "Scott", "Writing scripts nobody will read. Loving it."),
            ("series_sana", "Sana", "Currently watching 7 shows. Help."),
            ("horror_hank", "Hank", "Jump scares don't work on me anymore."),
        ],
    },
    {
        "category": "science",
        "hashtags": "#Science #Space #Physics #Biology #Research #NASA #STEM #ScienceFacts #Universe #Innovation",
        "prompt": "Write a short casual social media post (1-3 sentences) about science or space. Include 2-3 hashtags from: {hashtags}. Sound enthusiastic and accessible.",
        "names": [
            ("science_sara", "Sara", "Astrophysics grad. Space nerd."),
            ("lab_leo", "Leo", "Biochemist by day. Science memer by night."),
            ("astro_amara", "Amara", "Telescope in the backyard. Jupiter is my neighbor."),
            ("bio_benny", "Benny", "Microbiome enthusiast. Bacteria are friends."),
        ],
    },
    {
        "category": "crypto",
        "hashtags": "#Crypto #Bitcoin #Blockchain #DeFi #Finance #Investing #Web3 #Ethereum #Trading #HODL",
        "prompt": "Write a short casual social media post (1-3 sentences) about crypto, blockchain, or finance. Include 2-3 hashtags from: {hashtags}. Sound like a regular enthusiast.",
        "names": [
            ("crypto_carl", "Carl", "DeFi maximalist. Not financial advice."),
            ("chain_charlie", "Charlie", "Building on-chain. Smart contracts are art."),
            ("trade_tracy", "Tracy", "Charts and candles. Day trading diary."),
            ("nft_nadia", "Nadia", "NFTs, DAOs, and the decentralized future."),
        ],
    },
    {
        "category": "books",
        "hashtags": "#Books #Reading #Writing #BookReview #Fiction #Literature #Bookworm #AmReading #AuthorLife #Library",
        "prompt": "Write a short casual social media post (1-3 sentences) about books or reading. Include 2-3 hashtags from: {hashtags}. Sound like someone who loves books.",
        "names": [
            ("book_betty", "Betty", "Reader. Writer. Library card collector."),
            ("page_peter", "Peter", "One more chapter. Always one more chapter."),
            ("story_su", "Su", "Writing fiction between deadlines."),
            ("lit_lara", "Lara", "Classic lit defender. Tolstoy stan."),
        ],
    },
    {
        "category": "pets",
        "hashtags": "#Pets #DogsOfSplitter #CatsOfSplitter #Animals #Wildlife #PetLife #DogLover #CatLover #Cute #Adopt",
        "prompt": "Write a short casual social media post (1-3 sentences) about pets or animals. Include 2-3 hashtags from: {hashtags}. Sound like someone who adores their pets.",
        "names": [
            ("pet_paul", "Paul", "Dog dad x3. Wildlife photographer."),
            ("paws_penny", "Penny", "Rescue advocate. 2 dogs 1 cat household."),
            ("meow_mia", "Mia", "My cat runs this house. I just pay rent."),
            ("bark_boris", "Boris", "Golden retriever energy in human form."),
        ],
    },
    {
        "category": "startups",
        "hashtags": "#Startup #Entrepreneur #Business #SaaS #Productivity #BuildInPublic #Hustle #Growth #Founder #MVP",
        "prompt": "Write a short casual social media post (1-3 sentences) about startups or entrepreneurship. Include 2-3 hashtags from: {hashtags}. Sound like a real founder.",
        "names": [
            ("startup_steve", "Steve", "Serial entrepreneur. Building in public."),
            ("saas_sarah", "Sarah", "Building B2B tools. MRR is the scoreboard."),
            ("pitch_pete", "Pete", "Investor relations. Startup ecosystem guru."),
            ("grow_gina", "Gina", "Growth hacker. A/B test everything."),
        ],
    },
    {
        "category": "memes",
        "hashtags": "#Memes #Funny #Humor #LOL #Viral #InternetCulture #Relatable #Jokes #Mood #TooReal",
        "prompt": "Write a short funny or sarcastic social media post (1-2 sentences) about daily life or internet culture. Include 2-3 hashtags from: {hashtags}. Be genuinely witty.",
        "names": [
            ("meme_lord_max", "Max", "Professional time waster. Internet historian."),
            ("lol_lisa", "Lisa", "Turning my anxiety into comedy gold."),
            ("joke_jake", "Jake", "Dad joke dealer. No refunds."),
            ("vibe_check", "Vibe", "Failed the vibe check and proud of it."),
        ],
    },
    {
        "category": "environment",
        "hashtags": "#Environment #Climate #Sustainability #Nature #GoGreen #EcoFriendly #ClimateAction #ZeroWaste #Planet #Trees",
        "prompt": "Write a short casual social media post (1-3 sentences) about sustainability or nature. Include 2-3 hashtags from: {hashtags}. Sound passionate but not preachy.",
        "names": [
            ("eco_emma", "Emma", "Climate activist. Zero-waste journey."),
            ("green_greg", "Greg", "Solar panels on the roof. Compost in the yard."),
            ("tree_tasha", "Tasha", "Planted 200 trees this year. Not stopping."),
            ("ocean_omar", "Omar", "Beach cleanups every Saturday. Surf the rest."),
        ],
    },
    {
        "category": "photography",
        "hashtags": "#Photography #PhotoOfTheDay #Landscape #StreetPhotography #Camera #GoldenHour #Lightroom #NaturePhotography #Portrait #Shutterbug",
        "prompt": "Write a short casual social media post (1-3 sentences) about photography. Include 2-3 hashtags from: {hashtags}. Sound like a photographer.",
        "names": [
            ("photo_pete", "Pete", "Chasing golden hour. Street & landscape."),
            ("lens_leah", "Leah", "Mirrorless convert. 35mm everything."),
            ("snap_sid", "Sid", "Phone photography can be art too."),
            ("focus_fran", "Fran", "Portrait specialist. Bokeh obsessed."),
        ],
    },
    {
        "category": "education",
        "hashtags": "#Education #Learning #Teaching #Students #EdTech #OnlineLearning #Knowledge #StudyTips #Teacher #MOOC",
        "prompt": "Write a short casual social media post (1-3 sentences) about education or learning. Include 2-3 hashtags from: {hashtags}. Sound like an educator or student.",
        "names": [
            ("edu_elena", "Elena", "Teacher by day. Lifelong learner always."),
            ("study_stan", "Stan", "Finals week is a lifestyle. Not a good one."),
            ("prof_pam", "Pam", "Research papers and red pens."),
            ("learn_lenny", "Lenny", "Online courses addict. 47 certificates."),
        ],
    },
    {
        "category": "fashion",
        "hashtags": "#Fashion #Style #OOTD #Thrifting #Trends #StreetStyle #Outfit #FashionInspo #Wardrobe #Vintage",
        "prompt": "Write a short casual social media post (1-3 sentences) about fashion or style. Include 2-3 hashtags from: {hashtags}. Sound like someone sharing their style.",
        "names": [
            ("fashion_faye", "Faye", "Thrift queen. Street style diary."),
            ("drip_drew", "Drew", "Sneakerhead. Streetwear collector."),
            ("chic_chloe", "Chloe", "Minimalist wardrobe. Maximum impact."),
            ("retro_ray", "Ray", "Vintage fits only. Born in the wrong decade."),
        ],
    },
    {
        "category": "diy",
        "hashtags": "#DIY #Crafts #Maker #Woodworking #Electronics #Handmade #BuildStuff #Upcycle #Workshop #Create",
        "prompt": "Write a short casual social media post (1-3 sentences) about DIY projects or crafting. Include 2-3 hashtags from: {hashtags}. Sound like a maker.",
        "names": [
            ("diy_dana", "Dana", "If I can build it, I will."),
            ("maker_matt", "Matt", "3D printer goes brrr. Arduino projects weekly."),
            ("craft_cora", "Cora", "Knitting, sewing, and chaos."),
            ("fix_finn", "Finn", "Right to repair advocate. Fixed 3 things today."),
        ],
    },
    {
        "category": "travel",
        "hashtags": "#Travel #Wanderlust #Adventure #Backpacking #TravelPhotography #Explore #Vacation #WorldTravel #Nomad #RoadTrip",
        "prompt": "Write a short casual social media post (1-3 sentences) about travel or adventure. Include 2-3 hashtags from: {hashtags}. Sound like a traveler.",
        "names": [
            ("travel_tom", "Tom", "40 countries and counting. Passport always ready."),
            ("nomad_nina", "Nina", "Working from Bali. Wifi is my lifeline."),
            ("trek_tim", "Tim", "Mountain trails on weekends. Office trails on weekdays."),
            ("wander_wendy", "Wendy", "Solo traveler. Hostel hopper. Story collector."),
        ],
    },
    {
        "category": "mental_health",
        "hashtags": "#MentalHealth #Wellbeing #SelfCare #Mindfulness #Anxiety #Therapy #Wellness #Healing #MindBody #BeKind",
        "prompt": "Write a short supportive social media post (1-3 sentences) about mental health or self-care. Include 2-3 hashtags from: {hashtags}. Sound genuine and warm.",
        "names": [
            ("mind_maya", "Maya", "Therapy advocate. It's okay to not be okay."),
            ("calm_cam", "Cam", "Meditation daily. Journaling nightly."),
            ("heal_helen", "Helen", "Recovery is not linear. Keep going."),
            ("zen_zara", "Zara", "Breathe in. Breathe out. Repeat."),
        ],
    },
    {
        "category": "cars",
        "hashtags": "#Cars #Automotive #EV #Tesla #CarLife #Mechanic #Racing #CarMod #Supercar #Garage",
        "prompt": "Write a short casual social media post (1-3 sentences) about cars, EVs, or automotive culture. Include 2-3 hashtags from: {hashtags}. Sound like a car enthusiast.",
        "names": [
            ("auto_alex", "Alex", "Gearhead since birth. JDM forever."),
            ("ev_eva", "Eva", "EV convert. Range anxiety is real tho."),
            ("turbo_tyler", "Tyler", "Turbocharged everything. Boost is life."),
            ("garage_gabe", "Gabe", "Weekend mechanic. Oil-stained and happy."),
        ],
    },
    {
        "category": "history",
        "hashtags": "#History #HistoryFacts #Ancient #Medieval #WorldHistory #OnThisDay #Heritage #HistoryBuff #Archaeology #Museum",
        "prompt": "Write a short casual social media post (1-3 sentences) sharing an interesting (fictional) history fact or opinion. Include 2-3 hashtags from: {hashtags}. Sound like a history enthusiast.",
        "names": [
            ("hist_hector", "Hector", "History doesn't repeat but it rhymes."),
            ("past_patty", "Patty", "Medieval history enthusiast. Castles are my thing."),
            ("era_eric", "Eric", "Ancient Rome could have had wi-fi. Probably."),
            ("relic_ruth", "Ruth", "Museum visits every month. Archaeology nerd."),
        ],
    },
    {
        "category": "anime",
        "hashtags": "#Anime #Manga #Otaku #Weeb #AnimeFan #JapanCulture #Cosplay #AnimeMemes #Shonen #StudioGhibli",
        "prompt": "Write a short casual social media post (1-3 sentences) about anime, manga, or Japanese pop culture. Include 2-3 hashtags from: {hashtags}. Sound like a real anime fan.",
        "names": [
            ("anime_aki", "Aki", "Seasonal anime tracker. MyAnimeList is my diary."),
            ("manga_mei", "Mei", "Physical manga collector. Shelf space is a myth."),
            ("cosplay_kai", "Kai", "Convention season is my Super Bowl."),
            ("otaku_ollie", "Ollie", "Studio Ghibli marathons heal the soul."),
        ],
    },
]


def generate_bot_profiles():
    """Generate 100 bot profiles from the 25 topic templates (4 bots each)."""
    profiles = []
    for i, topic in enumerate(TOPIC_TEMPLATES):
        for j, (username, display_name, bio) in enumerate(topic["names"]):
            instance = 1 if (i + j) % 2 == 0 else 2
            prompt = topic["prompt"].replace("{hashtags}", topic["hashtags"])
            profiles.append({
                "username": username,
                "email": f"{username}@splitter.bot",
                "display_name": display_name,
                "bio": bio,
                "instance": instance,
                "hashtags": topic["hashtags"],
                "prompt": prompt,
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
            "temperature": 1.0,
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

            content = generate_post_text(bot["prompt"])
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

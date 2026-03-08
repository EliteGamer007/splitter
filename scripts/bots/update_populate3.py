import re

with open('scripts/bots/populate.py', 'r', encoding='utf-8') as f:
    text = f.read()

new_code = '''TOPIC_TEMPLATES = [
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
        profiles.append({
            "username": username,
            "email": f"{username}@bot.local",
            "display_name": display,
            "bio": bio,
            "instance": instance
        })
    return profiles'''

# Now safe replacement via regex
import re
pattern = re.compile(r'TOPIC_TEMPLATES = \[.*?def generate_bot_profiles\(\):\n.*?return profiles', re.DOTALL)
new_text = pattern.sub(new_code, text)

with open('scripts/bots/populate.py', 'w', encoding='utf-8') as f:
    f.write(new_text)

print("Done restoring and updating!")

import re
import random

with open('scripts/bots/populate.py', 'r', encoding='utf-8') as f:
    text = f.read()

start_idx = text.find('TOPIC_TEMPLATES = [')
end_idx = text.find('def generate_bot_profiles():')

new_code = '''TOPIC_TEMPLATES = [
    {
        "category": "splitter_evangelist",
        "hashtags": ["#Splitter", "#Federation", "#Decentralization", "#Web3", "#PrivacyFirst", "#FutureOfSocial"],
        "prompt": "Write a short casual social media post (1-3 sentences) praising Splitter, the new federated social media app we are on. Mention things like decentralization, NO ALGORITHMS, or how it beats Twitter. Include 1-2 hashtags like {hashtags}. Sound like an enthusiastic early adopter."
    },
    {
        "category": "splitter_support",
        "hashtags": ["#SplitterApp", "#TechSupport", "#Federation", "#UIUX", "#OpenSource"],
        "prompt": "Write a short casual social media post (1-2 sentences) about how smooth the Splitter UI is, or how cool it is to see live trending hashtags here. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "tech_general",
        "hashtags": ["#Programming", "#AI", "#Coding", "#Developer", "#GoLang", "#NextJS"],
        "prompt": "Write a short casual social media post (1-3 sentences) about coding, AI, or web development, and how cool it is to build things on Splitter. Include 1-2 hashtags like {hashtags}. Sound like a developer."
    },
    {
        "category": "crypto_decentralization",
        "hashtags": ["#Crypto", "#Blockchain", "#DeFi", "#Web3", "#Decentralization"],
        "prompt": "Write a short casual social media post (1-3 sentences) about cryptography, decentralization, and taking back control of our data away from big tech. Include 1-2 hashtags like {hashtags}."
    },
    {
        "category": "casual_life",
        "hashtags": ["#WeekendVibes", "#Coffee", "#DailyGrind", "#LifeUpdate", "#Chill"],
        "prompt": "Write a short casual social media post (1-3 sentences) about enjoying a coffee, weekend plans, or just chilling. Mention how much nicer it is to post here on Splitter than on other toxic apps. Include 1 hashtag like {hashtags}. Sound like a regular person just posting an update."
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

'''

# We also need to strip out the old function and replace it so it doesn't double print
end_func_idx = text.find('def create_bots(', end_idx)

new_text = text[:start_idx] + new_code + text[end_func_idx:]

with open('scripts/bots/populate.py', 'w', encoding='utf-8') as f:
    f.write(new_text)

print("Successfully replaced templates and bot profiles!")
